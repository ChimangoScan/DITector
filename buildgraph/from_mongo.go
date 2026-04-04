package buildgraph

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/NSSL-SJTU/DITector/myutils"
	"go.mongodb.org/mongo-driver/bson"
	mongodb_opts "go.mongodb.org/mongo-driver/mongo/options"
)

type GraphJob struct {
	Registry      string
	RepoNamespace string
	RepoName      string
	TagName       string
	ImageMeta     *myutils.Image
}

// StartFromMongo inicia o processamento paralelo do grafo a partir do MongoDB
func StartFromMongo(page int64, pageSize int64, tagCnt int, pullCountThreshold int64) {
	beginTime := time.Now()
	fmt.Println("Build paralelo iniciado em:", myutils.GetLocalNowTimeStr())

	// Canais de orquestração. Buffers grandes evitam bloqueio entre estágios.
	repoChan := make(chan *myutils.Repository, 4000)
	jobChan := make(chan GraphJob, 20000)
	doneChan := make(chan struct{})

	var wgLoad sync.WaitGroup
	var wgBuild sync.WaitGroup

	// 1. Loader: Lê do MongoDB e joga no repoChan
	go func() {
		loadReposToChannel(page, pageSize, pullCountThreshold, repoChan)
		close(repoChan)
	}()

	// 2. Repo Workers: buscam tags e manifestos via Docker Hub API (I/O bound).
	// Escalamos muito acima de NumCPU pois a maior parte do tempo é espera de rede.
	numRepoWorkers := runtime.NumCPU() * 16
	if numRepoWorkers < 64 {
		numRepoWorkers = 64
	}
	for i := 0; i < numRepoWorkers; i++ {
		wgLoad.Add(1)
		go func() {
			defer wgLoad.Done()
			repoWorker(repoChan, jobChan, tagCnt)
		}()
	}

	// 3. Build Workers: inserem no Neo4j (I/O bound — Bolt protocol over TCP).
	numBuildWorkers := runtime.NumCPU() * 4
	if numBuildWorkers < 16 {
		numBuildWorkers = 16
	}
	for i := 0; i < numBuildWorkers; i++ {
		wgBuild.Add(1)
		go func() {
			defer wgBuild.Done()
			buildGraphWorker(jobChan)
		}()
	}

	// Fechamento em cascata
	go func() {
		wgLoad.Wait()
		close(jobChan)
		wgBuild.Wait()
		close(doneChan)
	}()

	<-doneChan
	fmt.Printf("Build finalizado. Tempo total: %v\n", time.Since(beginTime))
}

func loadReposToChannel(_ int64, _ int64, threshold int64, ch chan *myutils.Repository) {
	for {
		// Resume: only load repos that have NOT been fully processed by a
		// previous run. graph_built_at is set by repoWorker after all tags
		// and images for the repo have been inserted into Neo4j.
		filter := bson.M{
			"pull_count":     bson.M{"$gte": threshold},
			"graph_built_at": bson.M{"$exists": false},
		}
		// Sort by pull_count descending to prioritize influential images
		opts := mongodb_opts.Find().SetSort(bson.M{"pull_count": -1})
		cursor, err := myutils.GlobalDBClient.Mongo.RepoColl.Find(context.Background(), filter, opts)
		if err != nil {
			break
		}
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {
			var r myutils.Repository
			if err := cursor.Decode(&r); err != nil {
				continue
			}
			ch <- &r
		}
		break 
	}
}

// tagConcurrency limits concurrent image-manifest fetches per repo to avoid
// triggering per-repo rate limits on Docker Hub.
const tagConcurrency = 4

func repoWorker(repoChan chan *myutils.Repository, jobChan chan GraphJob, tagCnt int) {
	for repo := range repoChan {
		tags := fetchTags(repo, tagCnt)
		if len(tags) == 0 {
			continue
		}

		sem := make(chan struct{}, tagConcurrency)
		var wg sync.WaitGroup
		for _, tag := range tags {
			wg.Add(1)
			sem <- struct{}{}
			go func(t *myutils.Tag) {
				defer wg.Done()
				defer func() { <-sem }()

				imgs, err := fetchImages(repo, t)
				if err != nil {
					return
				}
				for _, img := range imgs {
					if img.OS == "windows" {
						continue
					}
					jobChan <- GraphJob{
						Registry:      "docker.io",
						RepoNamespace: repo.Namespace,
						RepoName:      repo.Name,
						TagName:       t.Name,
						ImageMeta:     img,
					}
				}
			}(tag)
		}
		wg.Wait()

		if myutils.GlobalDBClient.MongoFlag {
			if err := myutils.GlobalDBClient.Mongo.MarkRepoGraphBuilt(repo.Namespace, repo.Name); err != nil {
				myutils.Logger.Error(fmt.Sprintf("MarkRepoGraphBuilt %s/%s failed: %v", repo.Namespace, repo.Name, err))
			}
		}
	}
}

// fetchTags returns tags from the MongoDB cache when all entries carry image
// references (i.e. were persisted by a previous run). Falls back to the live
// Docker Hub API otherwise.
func fetchTags(repo *myutils.Repository, tagCnt int) []*myutils.Tag {
	if myutils.GlobalDBClient.MongoFlag {
		tags, err := myutils.GlobalDBClient.Mongo.FindAllTagsByRepoName(repo.Namespace, repo.Name)
		if err == nil && allTagsHaveImages(tags) {
			return tags
		}
	}
	tags, err := myutils.ReqTagsMetadata(repo.Namespace, repo.Name, 1, tagCnt)
	if err != nil {
		return nil
	}
	return tags
}

// allTagsHaveImages reports whether every tag carries at least one image
// reference, which indicates the tag set was previously persisted with full
// metadata. An empty slice returns false to force an API fetch.
func allTagsHaveImages(tags []*myutils.Tag) bool {
	if len(tags) == 0 {
		return false
	}
	for _, t := range tags {
		if len(t.Images) == 0 {
			return false
		}
	}
	return true
}

// fetchImages returns full image metadata for a tag. It first attempts to
// reconstruct the image list from MongoDB (cache hit path: zero API calls).
// On any cache miss it falls back to the live API and persists the result.
func fetchImages(repo *myutils.Repository, t *myutils.Tag) ([]*myutils.Image, error) {
	if myutils.GlobalDBClient.MongoFlag && len(t.Images) > 0 {
		if imgs, ok := loadImagesFromCache(t.Images); ok {
			return imgs, nil
		}
	}
	return fetchAndPersistImages(repo, t)
}

// loadImagesFromCache looks up every ImageInTag digest in ImgColl.
// Returns (imgs, true) only when every document is found with layer data
// populated — a partial cache is treated as a miss to preserve consistency.
func loadImagesFromCache(refs []myutils.ImageInTag) ([]*myutils.Image, bool) {
	imgs := make([]*myutils.Image, 0, len(refs))
	for _, ref := range refs {
		img, err := myutils.GlobalDBClient.Mongo.FindImageByDigest(ref.Digest)
		if err != nil || len(img.Layers) == 0 {
			return nil, false
		}
		imgs = append(imgs, img)
	}
	return imgs, true
}

// fetchAndPersistImages calls the live Docker Hub API for image manifests and
// writes both the tag document and each image document to MongoDB so that
// subsequent runs can serve them from cache.
func fetchAndPersistImages(repo *myutils.Repository, t *myutils.Tag) ([]*myutils.Image, error) {
	imgs, err := myutils.ReqImagesMetadata(repo.Namespace, repo.Name, t.Name)
	if err != nil {
		return nil, err
	}
	if !myutils.GlobalDBClient.MongoFlag {
		return imgs, nil
	}
	for _, img := range imgs {
		if err := myutils.GlobalDBClient.Mongo.UpdateImage(img); err != nil {
			myutils.Logger.Error(fmt.Sprintf("UpdateImage %s failed: %v", img.Digest, err))
		}
	}
	t.Images = make([]myutils.ImageInTag, 0, len(imgs))
	for _, img := range imgs {
		t.Images = append(t.Images, myutils.ImageInTag{
			Architecture: img.Architecture,
			OS:           img.OS,
			Digest:       img.Digest,
			Size:         img.Size,
		})
	}
	if err := myutils.GlobalDBClient.Mongo.UpdateTag(t); err != nil {
		myutils.Logger.Error(fmt.Sprintf("UpdateTag %s/%s:%s failed: %v", repo.Namespace, repo.Name, t.Name, err))
	}
	return imgs, nil
}

func buildGraphWorker(jobChan chan GraphJob) {
	for job := range jobChan {
		id := fmt.Sprintf("%s/%s/%s:%s@%s", job.Registry, job.RepoNamespace, job.RepoName, job.TagName, job.ImageMeta.Digest)
		myutils.GlobalDBClient.Neo4j.InsertImageToNeo4j(id, job.ImageMeta)
		myutils.Logger.Info(fmt.Sprintf("Inserido no Neo4j: %s", id))
	}
}
