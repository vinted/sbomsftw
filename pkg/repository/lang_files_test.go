package repository

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLanguageFilesByPredicate(t *testing.T) {
	var fileMatchAttempts []string
	predicate := func(isDir bool, filename string) bool {
		filename = filepath.Base(filename)
		fileMatchAttempts = append(fileMatchAttempts, filename)
		if isDir {
			return false
		}
		return filename == "Packages" || filename == "Packages.lock"
	}

	var (
		testFS = fstest.MapFS{
			"test-repository/Packages":                            {},
			"test-repository/ignore.txt":                          {},
			"test-repository/Packages.lock":                       {},
			"test-repository/inner-dir":                           {Mode: fs.ModeDir},
			"test-repository/inner-dir/.git":                      {Mode: fs.ModeDir},
			"test-repository/inner-dir/.git/Packages":             {},
			"test-repository/inner-dir/test/Packages":             {},
			"test-repository/inner-dir/tests/Packages":            {},
			"test-repository/inner-dir/Packages":                  {Mode: fs.ModeDir},
			"test-repository/inner-dir/Packages.lock":             {Mode: fs.ModeDir},
			"test-repository/inner-dir/testing/Packages":          {},
			"test-repository/inner-dir/deepest-dir/Packages.lock": {},
		}
		expectedBOMRoots = []string{
			"test-repository/Packages",
			"test-repository/Packages.lock",
			"test-repository/inner-dir/deepest-dir/Packages.lock",
			"test-repository/inner-dir/testing/Packages",
		}
	)

	t.Run("correct BOM roots are found based on the predicate provided", func(t *testing.T) {
		bomRoots, err := languageFilesByPredicate(testFS, predicate)
		require.NoError(t, err)
		assert.Equal(t, expectedBOMRoots, bomRoots)

		assert.Equal(t, []string{
			".", "test-repository", "Packages", "Packages.lock", "ignore.txt", "inner-dir",
			"Packages", "Packages.lock", "deepest-dir", "Packages.lock", "testing", "Packages",
		}, fileMatchAttempts)
	})

	t.Run("error is returned whenever FS walk fails", func(t *testing.T) {
		roots, err := languageFilesByPredicate(os.DirFS("/non-existing"), nil)
		assert.NotNil(t, err)
		assert.Empty(t, roots)
	})
}

func TestRelativeToAbsolutePaths(t *testing.T) {
	relativePaths := []string{
		"Packages",
		"Packages.lock",
		"inner-dir/Packages.lock",
		"inner-dir/deepest-dir/Packages.lock",
	}
	expectedPaths := []string{
		"/tmp/test-repository/Packages",
		"/tmp/test-repository/Packages.lock",
		"/tmp/test-repository/inner-dir/Packages.lock",
		"/tmp/test-repository/inner-dir/deepest-dir/Packages.lock",
	}
	got := relativeToAbsolutePaths("/tmp/test-repository", relativePaths...)
	assert.Equal(t, expectedPaths, got)
}
