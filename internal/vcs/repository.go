package vcs

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const checkoutsPath = "/tmp/checkouts/"

type Repository struct {
	Name        string
	Description string
	Archived    bool
	Language    string
	URL         string `json:"html_url"`
}

func (r Repository) FsPath() string {
	return checkoutsPath + r.Name
}

func (r Repository) Clone(username, apiToken string) error {
	_, err := git.PlainClone(checkoutsPath+r.Name, false, &git.CloneOptions{
		URL:  r.URL,
		Auth: &http.BasicAuth{Username: username, Password: apiToken},
	})
	return err
}
