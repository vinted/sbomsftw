package vcs_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/vcs"
	"testing"
)

func TestFsPath(t *testing.T) {
	repository := vcs.Repository{
		Name:   "linux",
		VCSUrl: "https://github.com/torvalds/linux.vcs",
	}
	assert.Equal(t, "/tmp/checkouts/linux", repository.FsPath())
}
