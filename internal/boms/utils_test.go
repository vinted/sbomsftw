package boms

import (
	"encoding/json"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"strings"
	"testing"
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
		assert.Nil(t, got)
		assert.ErrorIs(t, err, NoBOMsToMergeError("won't merge an empty slice of BOMs - nothing to do"))
	})

	t.Run("return an error when BOM decoding fails", func(t *testing.T) {
		got, err := Merge(XML, "Invalid BOM 😱 ")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("return an error when trying to decode an XML BOM with JSON type", func(t *testing.T) {
		got, err := Merge(JSON, firstBOM) // Invalid type - input is XML string
		assert.Nil(t, got)
		var e *json.SyntaxError
		assert.ErrorAs(t, err, &e)
	})

	t.Run("return an error when trying to decode BOM with unsupported type", func(t *testing.T) {
		got, err := Merge(BOMType(42), firstBOM) //Unsupported type
		assert.Nil(t, got)
		assert.ErrorIs(t, err, BadBOMTypeError{BOMType: BOMType(42)})
	})

	t.Run("merge multiple-lockfiles BOMs correctly", func(t *testing.T) {
		got, err := Merge(XML, firstBOM, secondBOM, thirdBOM)

		var actualPURLs []string
		for _, component := range *got.Components {
			actualPURLs = append(actualPURLs, component.PackageURL)
		}
		assert.Equal(t, []string{
			"pkg:gem/activesupport@6.1.5",
			"pkg:gem/authorizenet@1.9.7",
			"pkg:gem/concurrent-ruby@1.1.9",
			"pkg:gem/rake@13.0.6",
		}, actualPURLs)
		assert.NoError(t, err)
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
		testBOM, err := ioutil.ReadFile("synthetic/bom-optional-dependencies.json")
		if err != nil {
			t.Fatalf("unable to read a test file: %s", err)
		}
		filteredBOM, err := FilterOutByScope(cdx.ScopeOptional, JSON, string(testBOM))

		got := new(cdx.BOM)
		decoder := cdx.NewBOMDecoder(strings.NewReader(filteredBOM), cdx.BOMFileFormatJSON)
		if err := decoder.Decode(got); err != nil {
			t.Fatalf("unable to decode generated bom: %s", err)
		}
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

	t.Run("return an error when trying to filter BOM with unsupported type", func(t *testing.T) {
		got, err := FilterOutByScope(cdx.ScopeOptional, BOMType(42), "") //Unsupported type
		assert.Empty(t, got)
		assert.ErrorIs(t, err, BadBOMTypeError{BOMType: BOMType(42)})
	})
}
