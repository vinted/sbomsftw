package bomtools

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestFilterOptionalDependencies(t *testing.T) {

	t.Run("filter out optional dependencies correctly", func(t *testing.T) {
		testBOM, err := ioutil.ReadFile("../../integration/testdata/bom-optional-dependencies.json")
		if err != nil {
			t.Fatalf("can't read a test file: %s", err)
		}
		bom, err := StringToCDX(testBOM)
		if err != nil {
			t.Fatalf("can't convert BOM string to cdx.BOM instance %s", err)
		}

		got := FilterOutByScope(bom, cdx.ScopeOptional)

		var actualPURLs []string
		for _, component := range *got.Components {
			actualPURLs = append(actualPURLs, component.PackageURL)
		}

		expectedPURLs := []string{
			"pkg:npm/actions/artifact@0.3.2",
			"pkg:npm/actions/core@1.2.4",
			"pkg:npm/actions/github@2.2.0",
			"pkg:npm/actions/http-client@1.0.8",
			"pkg:npm/lhci/cli@0.4.1",
			"pkg:npm/lhci/utils@0.4.1",
			"pkg:npm/lhci/utils@0.4.0",
			"pkg:npm/is-windows@1.0.2",
			"pkg:npm/lodash@4.5.0",
		}
		assert.Equal(t, expectedPURLs, actualPURLs)
	})

	t.Run("return an error when trying to filter dependencies from malformed BOMs", func(t *testing.T) {
		assert.Nil(t, FilterOutByScope(nil, cdx.ScopeOptional)) // Return nil when trying to filter from nil

		//Return the same BOM when trying to filter a bom with nil Components
		bom := new(cdx.BOM)
		assert.Equal(t, bom, FilterOutByScope(bom, cdx.ScopeOptional))
	})
}

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

func TestSquashRoots(t *testing.T) {
	rootsToSquash := []string{
		"/tmp/test/go.mod",
		"/tmp/test/go.sum",
		"/tmp/inner-dir/go.mod",
		"/tmp/inner-dir/go.sum",
		"/tmp/inner-dir/deepest-dir/go.mod",
		"/tmp/inner-dir/deepest-dir/go.sum",
	}
	want := []string{
		"/tmp/test",
		"/tmp/inner-dir",
		"/tmp/inner-dir/deepest-dir",
	}
	got := SquashRoots(rootsToSquash)
	assert.ElementsMatch(t, want, got)
}
