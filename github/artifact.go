package github

import (
	"encoding/json"
	"fmt"
	"time"
)

type ArtifactResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}
type Artifact struct {
	ID                 int       `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	SizeInBytes        int       `json:"size_in_bytes"`
	URL                string    `json:"url"`
	ArchiveDownloadURL string    `json:"archive_download_url"`
	Expired            bool      `json:"expired"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func GetArtifacts(owner, repo string) (artifactResponse ArtifactResponse, err error) {
	body, err := get(fmt.Sprintf("repos/%s/%s/actions/artifacts", owner, repo))
	if nil != err {
		return
	}
	err = json.Unmarshal(body, &artifactResponse)
	return
}

func DeleteArtifact(owner, repo string, id int) (err error) {
	_, err = delete(fmt.Sprintf("repos/%s/%s/actions/artifacts/%d", owner, repo, id))
	return
}

func DownloadArtifact(owner, repo string, id int) (body []byte, err error) {
	body, err = get(fmt.Sprintf("repos/%s/%s/actions/artifacts/%d/zip", owner, repo, id))
	return
}
