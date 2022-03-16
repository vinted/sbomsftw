package collectors

import (
	"encoding/json"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"io"
	"io/fs"
	"os"
	"strings"
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
		bomRoots, err := findRoots(testFS, predicate)
		assert.NoError(t, err)
		assert.Equal(t, expectedBOMRoots, bomRoots)

		assert.Equal(t, []string{".", "test-repository", "Packages", "Packages.lock", "ignore.txt", "inner-dir",
			"Packages", "Packages.lock", "deepest-dir", "Packages.lock"}, fileMatchAttempts)
	})

	t.Run("error is returned whenever FS walk fails", func(t *testing.T) {
		roots, err := findRoots(os.DirFS("/non-existing"), nil)

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
	got := relativeToAbsoluteRoots("/tmp/test-repository", relativeRoots...)
	assert.Equal(t, expectedRoots, got)
}

func TestNormalizeRoots(t *testing.T) {
	rawRoots := []string{
		"/dir/Packages",
		"/dir/Packages.lock",
		"/dir/inner-dir/Packages",
		"/dir/inner-dir/deep-dir/Packages",
		"/dir/inner-dir/deep-dir/Packages.lock",
		"/dir/inner-dir/deep-dir/deepest-dir/Packages.lock",
	}
	expectedRoots := []string{
		"/dir/Packages.lock",
		"/dir/inner-dir/Packages",
		"/dir/inner-dir/deep-dir/Packages.lock",
		"/dir/inner-dir/deep-dir/deepest-dir/Packages.lock",
	}
	assert.ElementsMatch(t, expectedRoots, normalizeRoots("Packages.lock", rawRoots...))
}

/*
Test data for the TestMerge function below.
First BOM contains `pkg:gem/activesupport@6.1.5` and `pkg:gem/authorizenet@1.9.7` libraries.
Second BOM contains `pkg:gem/rake@13.0.6`.
This BOM contains `pkg:gem/rake@13.0.6` (duplicate), `pkg:gem/authorizenet@1.9.7` (duplicate) and
`pkg:gem/concurrent-ruby@1.1.9` (uniques). These three SBOMs are being used as input to the Merge function.
The result of a Merge function must return a single merged BOM of unique elements: (in a JSON format)
`pkg:gem/activesupport@6.1.5`, `pkg:gem/authorizenet@1.9.7`, `pkg:gem/concurrent-ruby@1.1.9` and `pkg:gem/rake@13.0.6`.
*/
const (
	firstBOM = `<bom xmlns="http://cyclonedx.org/schema/bom/1.1" version="1" serialNumber="urn:uuid:56eb618f-1b08-4dd9-ab63-75479b7db344">
  <components>
    <component type="library">
      <name>activesupport</name>
      <version>6.1.5</version>
      <description>A toolkit of support libraries and Ruby core extensions extracted from the Rails framework.</description>
      <hashes>
        <hash alg="SHA-256">a4cfdb0f1fa3527fdf5729fc34ebb028802320c335f6313ebfd010357ab3b84a</hash>
      </hashes>
      <licenses>
        <license>
          <id>MIT</id>
        </license>
      </licenses>
      <purl>pkg:gem/activesupport@6.1.5</purl>
    </component>
    <component type="library">
      <name>authorizenet</name>
      <version>1.9.7</version>
      <description>Authorize.Net Payments SDK</description>
      <hashes>
        <hash alg="SHA-256">59d8c44a7ddfdedd737047154902e5b119bedd48956390c2015ef17b87f58510</hash>
      </hashes>
      <licenses>
        <license>
          <name>https://requests.com/AuthorizeNet/sdk-ruby/blob/master/LICENSE.txt</name>
        </license>
      </licenses>
      <purl>pkg:gem/authorizenet@1.9.7</purl>
    </component>
  </components>
</bom>`
	secondBOM = `<?xml version="1.0" encoding="UTF-8"?>
<bom xmlns="http://cyclonedx.org/schema/bom/1.1" version="1" serialNumber="urn:uuid:a5464ac2-4540-40c0-a787-33446419bfdd">
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
	thirdBOM = `<?xml version="1.0" encoding="UTF-8"?>
<bom xmlns="http://cyclonedx.org/schema/bom/1.1" version="1" serialNumber="urn:uuid:56eb618f-1b08-4dd9-ab63-75479b7db344">
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
    <component type="library">
      <name>authorizenet</name>
      <version>1.9.7</version>
      <description>Authorize.Net Payments SDK</description>
      <hashes>
        <hash alg="SHA-256">59d8c44a7ddfdedd737047154902e5b119bedd48956390c2015ef17b87f58510</hash>
      </hashes>
      <licenses>
        <license>
          <name>https://requests.com/AuthorizeNet/sdk-ruby/blob/master/LICENSE.txt</name>
        </license>
      </licenses>
      <purl>pkg:gem/authorizenet@1.9.7</purl>
    </component>
    <component type="library">
      <name>concurrent-ruby</name>
      <version>1.1.9</version>
      <description>Modern concurrency tools for Ruby. Inspired by Erlang, Clojure, Scala, Haskell, F#, C#, Java, and classic concurrency patterns.</description>
      <hashes>
        <hash alg="SHA-256">0ec0846d991c38f355b4228ad8ea77aa69c3fdaa320cd574dafedc10c4688a5b</hash>
      </hashes>
      <licenses>
        <license>
          <id>MIT</id>
        </license>
      </licenses>
      <purl>pkg:gem/concurrent-ruby@1.1.9</purl>
    </component>
  </components>
</bom>`
)

func TestMerge(t *testing.T) {

	t.Run("return an error when there are no BOMs to merge", func(t *testing.T) {
		got, err := Merge(XML, []string{}...)
		assert.Empty(t, got)
		assert.ErrorIs(t, err, NoBOMsToMergeError("won't merge an empty slice of BOMs - nothing to do"))
	})

	t.Run("return an error when BOM decoding fails", func(t *testing.T) {
		got, err := Merge(XML, "Invalid BOM 😱 ")
		assert.Empty(t, got)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("return an error when trying to decode an XML BOM with JSON type", func(t *testing.T) {
		got, err := Merge(JSON, firstBOM) // Invalid type - input is XML string
		assert.Empty(t, got)
		var e *json.SyntaxError
		assert.ErrorAs(t, err, &e)
	})

	t.Run("return an error when trying to decode BOM with unsupported type", func(t *testing.T) {
		got, err := Merge(BOMType(42), firstBOM) //Unsupported type
		assert.Empty(t, got)
		assert.ErrorIs(t, err, BadBOMTypeError{BOMType: BOMType(42)})
	})

	t.Run("merge multiple BOMs correctly", func(t *testing.T) {
		bomJSON, err := Merge(XML, firstBOM, secondBOM, thirdBOM)
		assert.NoError(t, err)

		got := new(cdx.BOM)
		decoder := cdx.NewBOMDecoder(strings.NewReader(bomJSON), cdx.BOMFileFormatJSON)
		if err := decoder.Decode(got); err != nil {
			t.Fatalf("unable to decode generated bom: %s", err)
		}

		assertUnionPURLs(t, got)
	})
}

func assertUnionPURLs(t *testing.T, got *cdx.BOM) {
	expectedPURLs := []string{
		"pkg:gem/activesupport@6.1.5",
		"pkg:gem/authorizenet@1.9.7",
		"pkg:gem/concurrent-ruby@1.1.9",
		"pkg:gem/rake@13.0.6",
	}
	var actualPURLs []string
	for _, component := range *got.Components {
		actualPURLs = append(actualPURLs, component.PackageURL)
	}
	assert.Equal(t, expectedPURLs, actualPURLs)
}
