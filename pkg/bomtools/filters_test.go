package bomtools

import (
	"os"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
)

func TestFilterOptionalDependencies(t *testing.T) {
	t.Run("filter out optional dependencies correctly", func(t *testing.T) {
		const testFilePath = "../../integration/test/bomtools/bom-with-optional-dependencies.json"
		testBOM, err := os.ReadFile(testFilePath)
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

		// Return the same BOM when trying to filter a bom with nil Components
		bom := new(cdx.BOM)
		assert.Equal(t, bom, FilterOutByScope(bom, cdx.ScopeOptional))
	})
}

func TestFilterOutComponentsWithoutAType(t *testing.T) {
	t.Run("filter out malformed components correctly", func(t *testing.T) {
		const testFilePath = "../../integration/test/bomtools/bom-with-malformed-components.json"
		testBOM, err := os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("can't read a test file: %s", err)
		}
		bom, err := StringToCDX(testBOM)
		if err != nil {
			t.Fatalf("can't convert BOM string to cdx.BOM instance %s", err)
		}

		got := FilterOutComponentsWithoutAType(bom)

		var actualPURLs []string
		for _, component := range *got.Components {
			actualPURLs = append(actualPURLs, component.PackageURL)
		}

		expectedPURLs := []string{
			"pkg:npm/actions/http-client@1.0.8",
			"pkg:cargo/boring@2.0.0",
		}
		assert.Equal(t, expectedPURLs, actualPURLs)
	})

	t.Run("return an error when trying to filter dependencies from malformed BOMs", func(t *testing.T) {
		assert.Nil(t, FilterOutComponentsWithoutAType(nil)) // Return nil when trying to filter from nil

		// Return the same BOM when trying to filter a bom with nil Components
		bom := new(cdx.BOM)
		assert.Equal(t, bom, FilterOutComponentsWithoutAType(bom))
	})
}
