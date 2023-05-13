package buildgraph

// 用于json marshal和unmarshal的接收器模板

type Repository struct {
	User            string `json:"user"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	RepositoryType  string `json:"repository_type"`
	Description     string `json:"description"`
	IsPrivate       bool   `json:"is_private"`
	IsAutomated     bool   `json:"is_automated"`
	StarCount       int    `json:"star_count"`
	PullCount       int64  `json:"pull_count"`
	LastUpdated     string `json:"last_updated"`
	DateRegistered  string `json:"date_registered"`
	FullDescription string `json:"full_description,omitempty"`
}

type Tag struct {
	Namespace           string
	Repository          string
	Name                string `json:"name"`
	LastUpdated         string `json:"last_updated"`
	LastUpdaterUsername string `json:"last_updater_username"`
	TagLastPulled       string `json:"tag_last_pulled"`
	TagLastPushed       string `json:"tag_last_pushed"`
	MediaType           string `json:"media_type"`
	ContentType         string `json:"content_type"`
}

type Image struct {
	Namespace  string
	Repository string
	Tag        string
	Arch       *Arch
}

type Arch struct {
	Architecture string  `json:"architecture"`
	Features     string  `json:"features"`
	Variant      string  `json:"variant"`
	Digest       string  `json:"digest"`
	Layers       []Layer `json:"layers"`
	OS           string  `json:"os"`
	Size         int64   `json:"size"`
	Status       string  `json:"status"`
	LastPulled   string  `json:"last_pulled"`
	LastPushed   string  `json:"last_pushed"`
}

type Layer struct {
	Digest      string `json:"digest,omitempty"`
	Size        int64  `json:"size"`
	Instruction string `json:"instruction"`
}
