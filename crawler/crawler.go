package crawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NSSL-SJTU/DITector/myutils"
)

// Alphabet for DFS keyword generation
const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789-_"

// RepositorySearchResponse for parsing Docker Hub search results
type RepositorySearchResponse struct {
	Count   int                 `json:"count"`
	Results []myutils.Repository `json:"results"`
}

// ParallelCrawler handles the distributed crawling logic
type ParallelCrawler struct {
	WorkerCount int
	KeywordChan chan string
	WG          sync.WaitGroup
	HTTPClient  *http.Client
}

// NewParallelCrawler initializes a new crawler
func NewParallelCrawler(workers int) *ParallelCrawler {
	return &ParallelCrawler{
		WorkerCount: workers,
		KeywordChan: make(chan string, 1000000), // Buffer for DFS keywords
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Start initiates the parallel crawl
func (pc *ParallelCrawler) Start() {
	myutils.Logger.Info(fmt.Sprintf("Starting Parallel Crawler with %d workers", pc.WorkerCount))

	// Launch workers
	for i := 0; i < pc.WorkerCount; i++ {
		pc.WG.Add(1)
		go pc.worker()
	}

	// Initial seed keywords
	for _, char := range alphabet {
		pc.KeywordChan <- string(char)
	}

	// We don't close the channel here because DFS will add more keywords
	// Monitoring logic or a specific stop signal can be added later
	pc.WG.Wait()
}

func (pc *ParallelCrawler) worker() {
	defer pc.WG.Done()
	for keyword := range pc.KeywordChan {
		pc.crawlKeyword(keyword)
	}
}

func (pc *ParallelCrawler) crawlKeyword(keyword string) {
	// 1. Check first page to get count
	url := myutils.GetRegURL(keyword, "community", "1", "100")
	resp, err := pc.HTTPClient.Get(url)
	if err != nil {
		myutils.Logger.Error(fmt.Sprintf("Request failed for keyword [%s]: %v", keyword, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		myutils.Logger.Warn(fmt.Sprintf("Keyword [%s] got status %d", keyword, resp.StatusCode))
		return
	}

	var searchRes RepositorySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		myutils.Logger.Error(fmt.Sprintf("JSON decode failed for keyword [%s]: %v", keyword, err))
		return
	}

	// 2. DFS Strategy
	if searchRes.Count >= 10000 && len(keyword) < 5 { // Limit depth to 5 chars to avoid infinite loops
		myutils.Logger.Info(fmt.Sprintf("Keyword [%s] has %d results. Deepening DFS...", keyword, searchRes.Count))
		for _, char := range alphabet {
			pc.KeywordChan <- keyword + string(char)
		}
	} else if searchRes.Count > 0 {
		// 3. Process results and Paginate
		myutils.Logger.Info(fmt.Sprintf("Keyword [%s] found %d repositories. Scraping...", keyword, searchRes.Count))
		pc.scrapeAllPages(keyword, searchRes.Count)
	}
}

func (pc *ParallelCrawler) scrapeAllPages(keyword string, totalCount int) {
	totalPages := (totalCount / 100) + 1
	if totalPages > 100 { totalPages = 100 } // Docker Hub search API hard limit is 100 pages

	for page := 1; page <= totalPages; page++ {
		url := myutils.GetRegURL(keyword, "community", fmt.Sprintf("%d", page), "100")
		pc.processPage(url)
	}
}

func (pc *ParallelCrawler) processPage(url string) {
	resp, err := pc.HTTPClient.Get(url)
	if err != nil {
		myutils.Logger.Error(fmt.Sprintf("Page request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	var searchRes RepositorySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return
	}

	for _, repo := range searchRes.Results {
		// Save to MongoDB
		if myutils.GlobalDBClient.MongoFlag {
			err := myutils.GlobalDBClient.Mongo.UpsertRepository(&repo)
			if err != nil {
				myutils.Logger.Error(fmt.Sprintf("Failed to upsert repo %s/%s: %v", repo.Namespace, repo.Name, err))
			}
		}
	}
}
