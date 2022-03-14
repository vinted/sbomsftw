package github

import (
	"github.com/go-git/go-git/v5"
	"os"
)

const checkoutsPath = "/tmp/checkouts/"

type Repository struct {
	Name        string
	Description string
	URL         string `json:"html_url"`
}

func (r Repository) FsPath() string {
	return checkoutsPath + r.Name
}

func (r Repository) Clone() error {
	_, err := git.PlainClone(checkoutsPath+r.Name, false, &git.CloneOptions{
		URL:      r.URL,
		Progress: os.Stdout,
	})
	return err
}
