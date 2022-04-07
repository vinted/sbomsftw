package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

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
          <id>Apache 2</id>
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
		got, err := Merge([]*cdx.BOM{}...)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, UnableToMergeBOMsError("can't merge BOMs - empty list of BOMs supplied"))
	})
	t.Run("return an error when trying to merge nil boms", func(t *testing.T) {
		got, err := Merge(nil, nil)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, UnableToMergeBOMsError("can't merge BOMs - BOM list can't contain elements"))
	})

	t.Run("merge multiple BOMs correctly", func(t *testing.T) {
		first, _ := BomStringToCDX(XML, firstBOM)
		second, _ := BomStringToCDX(XML, secondBOM)
		third, _ := BomStringToCDX(XML, thirdBOM)

		got, err := Merge(first, second, third)
		assert.NoError(t, err)

		var purlsToLicenses = make(map[string][]cdx.LicenseChoice)
		for _, component := range *got.Components {
			purlsToLicenses[component.PackageURL] = *component.Licenses
		}

		assert.Equal(t, 1, len(purlsToLicenses["pkg:gem/activesupport@6.1.5"]))
		assert.Equal(t, "MIT", purlsToLicenses["pkg:gem/activesupport@6.1.5"][0].License.ID)

		assert.Equal(t, 1, len(purlsToLicenses["pkg:gem/authorizenet@1.9.7"]))
		assert.Equal(t, "Apache 2", purlsToLicenses["pkg:gem/authorizenet@1.9.7"][0].License.ID)

		assert.Equal(t, 1, len(purlsToLicenses["pkg:gem/concurrent-ruby@1.1.9"]))
		assert.Equal(t, "MIT", purlsToLicenses["pkg:gem/concurrent-ruby@1.1.9"][0].License.ID)

		assert.Equal(t, 1, len(purlsToLicenses["pkg:gem/rake@13.0.6"]))
		assert.Equal(t, "MIT", purlsToLicenses["pkg:gem/rake@13.0.6"][0].License.ID)
	})
}

func assertMergeURLs(t *testing.T, got *cdx.BOM) {
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

func TestFilterOptionalDependencies(t *testing.T) {

	t.Run("filter out optional dependencies correctly", func(t *testing.T) {
		testBOM, err := ioutil.ReadFile("testdata/bom-optional-dependencies.json")
		if err != nil {
			t.Fatalf("can't read a test file: %s", err)
		}
		bom, err := BomStringToCDX(JSON, string(testBOM))
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

func TestAttachCPEs(t *testing.T) {
	bom, err := BomStringToCDX(XML, firstBOM)
	assert.NoError(t, err)

	var got []string
	for _, c := range *attachCPEs(bom).Components {
		got = append(got, c.CPE)
	}
	assert.Equal(t, []string{
		"cpe:2.3:a:activesupport:activesupport:6.1.5:*:*:*:*:*:*:*",
		"cpe:2.3:a:authorizenet:authorizenet:1.9.7:*:*:*:*:*:*:*",
	}, got)
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

func TestBOMConversions(t *testing.T) {
	t.Run("convert XML bom to JSON bom correctly", func(t *testing.T) {
		got, err := ConvertBetweenTypes(XML, JSON, xmlBOM)
		assert.NoError(t, err)
		assert.Equal(t, jsonBOM, got)
	})
	t.Run("convert JSON bom to XML bom correctly", func(t *testing.T) {
		got, err := ConvertBetweenTypes(JSON, XML, jsonBOM)
		assert.NoError(t, err)
		assert.Equal(t, xmlBOM, got)
	})
	t.Run("return an error when converting from unsupported type", func(t *testing.T) {
		got, err := ConvertBetweenTypes(BOMType(42), XML, jsonBOM) //Unsupported type
		assert.Empty(t, got)
		assert.ErrorIs(t, err, BadBOMTypeError{BOMType: BOMType(42)})
	})
	t.Run("return an error when converting to unsupported type", func(t *testing.T) {
		got, err := ConvertBetweenTypes(XML, BOMType(42), xmlBOM) //Unsupported type
		assert.Empty(t, got)
		assert.ErrorIs(t, err, BadBOMTypeError{BOMType: BOMType(42)})
	})
	t.Run("return an error when converting a JSON BOM with XML type", func(t *testing.T) {
		got, err := ConvertBetweenTypes(XML, JSON, jsonBOM)
		assert.Empty(t, got)
		assert.ErrorIs(t, err, io.EOF)
	})
}

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
	got := squashRoots(rootsToSquash)
	assert.ElementsMatch(t, want, got)
}
