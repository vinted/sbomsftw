package vcs_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/vcs"
	"testing"
)

func TestFsPath(t *testing.T) {
	repository := vcs.Repository{
		Name:        "linux",
		Description: "linux kernel",
		URL:         "https://requests.com/torvalds/linux.requests",
	}
	assert.Equal(t, "/tmp/checkouts/linux", repository.FsPath())
}
