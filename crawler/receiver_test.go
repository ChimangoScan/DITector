package crawler

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestRepository___String(t *testing.T) {
	// 用xmrig2021/r2021的数据做了测试
	repoMeta := `{
    "user": "xmrig2021",
    "name": "r2021",
    "namespace": "xmrig2021",
    "repository_type": "image",
    "status": 1,
    "status_description": "active",
    "description": "",
    "is_private": false,
    "is_automated": false,
    "can_edit": false,
    "star_count": 0,
    "pull_count": 117656,
    "last_updated": "2021-05-25T22:27:22.013449Z",
    "date_registered": "2021-05-19T21:09:34.187706Z",
    "collaborator_count": 0,
    "affiliation": null,
    "hub_user": "xmrig2021",
    "has_starred": false,
    "full_description": "",
    "permissions": {
        "read": true,
        "write": false,
        "admin": false
    },
    "media_types": [
        "application/vnd.docker.container.image.v1+json"
    ],
    "content_types": [
        "image"
    ]
}`
	var repo Repository__
	if err := json.Unmarshal([]byte(repoMeta), &repo); err != nil {
		t.Fatal("[ERROR] Json unmarshal repoMeta to Repository__: ", err)
	}

	repoTags := `{
    "count": 1,
    "next": null,
    "previous": null,
    "results": [
        {
            "creator": 12987695,
            "id": 150467810,
            "images": [
                {
                    "architecture": "amd64",
                    "features": "",
                    "variant": null,
                    "digest": "sha256:6c74ec72ad3fdf3a018f0b74edbbe194dfcd03ba9316c5cb4620ad84e7829a5d",
                    "os": "linux",
                    "os_features": "",
                    "os_version": null,
                    "size": 8988370,
                    "status": "inactive",
                    "last_pulled": "2022-11-08T11:51:53.932662Z",
                    "last_pushed": "2021-05-25T22:27:21.712654Z"
                }
            ],
            "last_updated": "2021-05-25T22:27:21.712654Z",
            "last_updater": 12987695,
            "last_updater_username": "xmrig2021",
            "name": "latest",
            "repository": 13731312,
            "full_size": 8988370,
            "v2": true,
            "tag_status": "inactive",
            "tag_last_pulled": "2022-11-08T11:51:53.932662Z",
            "tag_last_pushed": "2021-05-25T22:27:21.712654Z",
            "media_type": "application/vnd.docker.container.image.v1+json",
            "content_type": "image"
        }
    ]
}`
	var tags TagReceiver__
	if err := json.Unmarshal([]byte(repoTags), &tags); err != nil {
		t.Fatal("[ERROR] Json unmarshal repoTags to TagReceiver: ", err)
	}
	repo.Tags = append(repo.Tags, tags.Results...)

	repoArchs := []string{
		`[
    {
        "architecture": "amd64",
        "features": null,
        "variant": null,
        "digest": "sha256:6c74ec72ad3fdf3a018f0b74edbbe194dfcd03ba9316c5cb4620ad84e7829a5d",
        "layers": [
            {
                "digest": "sha256:540db60ca9383eac9e418f78490994d0af424aab7bf6d0e47ac8ed4e2e9bcbba",
                "size": 2811969,
                "instruction": "ADD file:8ec69d882e7f29f0652d537557160e638168550f738d0d49f90a7ef96bf31787 in / "
            },
            {
                "size": 0,
                "instruction": " CMD [\"/bin/sh\"]"
            },
            {
                "size": 0,
                "instruction": " LABEL Name=alpinexmrig Version=0.0.1"
            },
            {
                "digest": "sha256:7e714a6c8606416bb61fc76eb91e00e937c90df603de2f12b394f9f1da0c4575",
                "size": 6176401,
                "instruction": "/bin/sh -c echo 'https://dl-cdn.alpinelinux.org/alpine/edge/community' >> /etc/apk/repositories     && echo 'http://dl-cdn.alpinelinux.org/alpine/edge/testing' >> /etc/apk/repositories     && apk add xmrig"
            },
            {
                "size": 0,
                "instruction": " CMD [\"xmrig\" \"--url\" \"172.241.166.110:443\" \"--threads\" \"4\" \"--tls\" \"--no-color\" \"--donate-level=1\"]"
            }
        ],
        "os": "linux",
        "os_features": null,
        "os_version": null,
        "size": 8988370,
        "status": "inactive",
        "last_pulled": "2022-11-08T11:51:53.932662Z",
        "last_pushed": "2021-05-25T22:27:21.712654Z"
    }
]`,
	}
	for i, _ := range repo.Tags {
		if err := json.Unmarshal([]byte(repoArchs[i]), &repo.Tags[i].Archs); err != nil {
			t.Fatal("[ERROR] Json unmarshal repoArchs to Arch__: ", err)
		}
		//fmt.Printf("%p\n", &repo.Tags[i])
		//fmt.Printf("%p\n", &a)
	}

	fmt.Println(repo)
}
