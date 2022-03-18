package vcs

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"os"
)

const checkoutsPath = "/tmp/checkouts/"

type Repository struct {
	Name        string
	Description string
	Archived    bool
	URL         string `json:"html_url"`
}

func (r Repository) FsPath() string {
	return checkoutsPath + r.Name
}

func (r Repository) Clone(username, apiToken string) error {
	_, err := git.PlainClone(checkoutsPath+r.Name, false, &git.CloneOptions{
		URL:      r.URL,
		Progress: os.Stdout,
		Auth:     &http.BasicAuth{Username: username, Password: apiToken},
	})
	return err
}
