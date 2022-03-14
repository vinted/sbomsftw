package github_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/github"
	"testing"
)

func TestFsPath(t *testing.T) {
	repository := github.Repository{
		Name:        "linux",
		Description: "linux kernel",
		URL:         "https://github.com/torvalds/linux.github",
	}
	assert.Equal(t, "/tmp/checkouts/linux", repository.FsPath())
}
