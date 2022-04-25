package collectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSquashToDirs(t *testing.T) {
	pathsToSquash := []string{
		"/tmp/test/go.mod",
		"/tmp/test/go.sum",
		"/tmp/inner-dir/go.mod",
		"/tmp/inner-dir/go.sum",
		"/tmp/inner-dir/deepest-dir/go.mod",
	}
	want := []string{
		"/tmp/test",
		"/tmp/inner-dir",
		"/tmp/inner-dir/deepest-dir",
	}
	got := SquashToDirs(pathsToSquash)
	assert.ElementsMatch(t, want, got)
}

func TestSplitPaths(t *testing.T) {
	pathsToSplit := []string{
		"/tmp/test/go.mod",
		"/tmp/test/go.sum",
		"/tmp/inner-dir/go.mod",
		"/tmp/inner-dir/go.sum",
		"/tmp/inner-dir/deepest-dir/go.mod",
	}
	got := SplitPaths(pathsToSplit)
	assert.ElementsMatch(t, []string{"go.mod", "go.sum"}, got["/tmp/test"])
	assert.ElementsMatch(t, []string{"go.mod", "go.sum"}, got["/tmp/inner-dir"])
	assert.ElementsMatch(t, []string{"go.mod"}, got["/tmp/inner-dir/deepest-dir"])
}
