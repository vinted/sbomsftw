package sboms

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

func TestFindRoots(t *testing.T) {

	var fileMatchAttempts []string
	predicate := func(filename string) bool {
		fileMatchAttempts = append(fileMatchAttempts, filename)
		return filename == "Packages" || filename == "Packages.lock"
	}

	var (
		testFS = fstest.MapFS{
			"test-repository/Packages":                            {},
			"test-repository/ignore.txt":                          {},
			"test-repository/Packages.lock":                       {},
			"test-repository/inner-dir":                           {Mode: fs.ModeDir},
			"test-repository/inner-dir/Packages":                  {Mode: fs.ModeDir},
			"test-repository/inner-dir/Packages.lock":             {Mode: fs.ModeDir},
			"test-repository/inner-dir/deepest-dir/Packages.lock": {},
		}
		expectedBOMRoots = []string{
			"test-repository/Packages",
			"test-repository/Packages.lock",
			"test-repository/inner-dir/deepest-dir/Packages.lock",
		}
	)

	t.Run("correct BOM roots are found based on the predicate provided", func(t *testing.T) {
		bomRoots, err := FindRoots(testFS, predicate)
		assert.NoError(t, err)
		assert.Equal(t, expectedBOMRoots, bomRoots)

		assert.Equal(t, []string{".", "test-repository", "Packages", "Packages.lock", "ignore.txt", "inner-dir",
			"Packages", "Packages.lock", "deepest-dir", "Packages.lock"}, fileMatchAttempts)
	})

	t.Run("error is returned whenever FS walk fails", func(t *testing.T) {
		roots, err := FindRoots(os.DirFS("/non-existing"), nil)

		assert.Empty(t, roots)

		var e *fs.PathError
		assert.ErrorAs(t, err, &e)
		assert.Contains(t, err.Error(), "unable to walk file system path:")
	})
}

func TestRelativeToAbsoluteRoots(t *testing.T) {
	relativeRoots := []string{
		"Packages",
		"Packages.lock",
		"inner-dir/Packages.lock",
		"inner-dir/deepest-dir/Packages.lock",
	}
	expectedRoots := []string{
		"/tmp/test-repository/Packages",
		"/tmp/test-repository/Packages.lock",
		"/tmp/test-repository/inner-dir/Packages.lock",
		"/tmp/test-repository/inner-dir/deepest-dir/Packages.lock",
	}
	got := RelativeToAbsoluteRoots(relativeRoots, "/tmp/test-repository")
	assert.Equal(t, expectedRoots, got)
}
