package bomtools

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRoots(t *testing.T) {

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
		bomRoots, err := findRoots(testFS, predicate)
		require.NoError(t, err)
		assert.Equal(t, expectedBOMRoots, bomRoots)

		assert.Equal(t, []string{".", "test-repository", "Packages", "Packages.lock", "ignore.txt", "inner-dir",
			"Packages", "Packages.lock", "deepest-dir", "Packages.lock"}, fileMatchAttempts)
	})

	t.Run("error is returned whenever FS walk fails", func(t *testing.T) {
		roots, err := findRoots(os.DirFS("/non-existing"), nil)
		assert.NotNil(t, err)
		assert.Empty(t, roots)
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
	got := relativeToAbsoluteRoots("/tmp/test-repository", relativeRoots...)
	assert.Equal(t, expectedRoots, got)
}

const xmlBOM = `<?xml version="1.0" encoding="UTF-8"?>
<bom xmlns="" serialNumber="urn:uuid:a5464ac2-4540-40c0-a787-33446419bfdd" version="1">
  <components>
    <component type="library">
      <name>rake</name>
      <version>13.0.6</version>
      <description>Rake is a Make-like program implemented in Ruby</description>
      <hashes>
        <hash alg="SHA-256">5ce4bf5037b4196c24ac62834d8db1ce175470391026bd9e557d669beeb19097</hash>
      </hashes>
      <licenses>
        <license>
          <id>MIT</id>
        </license>
      </licenses>
      <purl>pkg:gem/rake@13.0.6</purl>
    </component>
  </components>
</bom>`

const jsonBOM = `{
  "bomFormat": "",
  "specVersion": "",
  "serialNumber": "urn:uuid:a5464ac2-4540-40c0-a787-33446419bfdd",
  "version": 1,
  "components": [
    {
      "type": "library",
      "name": "rake",
      "version": "13.0.6",
      "description": "Rake is a Make-like program implemented in Ruby",
      "hashes": [
        {
          "alg": "SHA-256",
          "content": "5ce4bf5037b4196c24ac62834d8db1ce175470391026bd9e557d669beeb19097"
        }
      ],
      "licenses": [
        {
          "license": {
            "id": "MIT"
          }
        }
      ],
      "purl": "pkg:gem/rake@13.0.6"
    }
  ]
}
`
