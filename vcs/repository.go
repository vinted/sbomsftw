package vcs

import (
	"github.com/go-git/go-git/v5"
	"os"
)

const checkoutsPath = "/tmp/checkouts/"

type Repository struct {
	Name   string
	VCSUrl string
}

func (r Repository) FsPath() string {
	return checkoutsPath + r.Name
}

func (r Repository) Clone() error {
	_, err := git.PlainClone(checkoutsPath+r.Name, false, &git.CloneOptions{
		URL:      r.VCSUrl,
		Progress: os.Stdout,
	})
	return err
}
