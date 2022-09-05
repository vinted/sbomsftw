package bomtools

import (
	"os"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeBoms(t *testing.T) {
	bomFromFile := func(filename string) *cdx.BOM {
		bomString, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("can't read BOM from file: %s\n", err)
		}
		bom, err := StringToCDX(bomString)
		if err != nil {
			t.Fatalf("can't convert BOM string to *cdx.BOM file: %s\n", err)
		}

		return bom
	}

	t.Run("normalize PURLs correctly", func(t *testing.T) {
		got := normalizePURLs(bomFromFile("../../integration/test/bomtools/normalize-purls-bom.json"))
		require.NotNil(t, got)
		require.NotNil(t, got.Components)

		var gotPURLs []string
		for _, c := range *got.Components {
			gotPURLs = append(gotPURLs, c.PackageURL)
		}
		assert.ElementsMatch(t, []string{
			"pkg:npm/actions/core@1.2.4",
			"pkg:npm/actions/core@1.2.4",
			"pkg:npm/actions/artifact@0.3.2",
			"pkg:npm/actions/artifact@0.3.2",
			"pkg:golang/github.com/pelletier/go-toml@1.8.1",
			"pkg:golang/github.com/pelletier/go-toml@1.8.1",
			"pkg:deb/busybox@1:1.30.1-4",
		}, gotPURLs)
	})

	t.Run("return errors when merging nil or empty list collectors", func(t *testing.T) {
		got, err := MergeBoms([]*cdx.BOM{}...)
		assert.Nil(t, got)
		assert.ErrorIs(t, ErrNoBOMsToMerge, err)

		got, err = MergeBoms(nil, nil)
		assert.Nil(t, got)
		assert.ErrorIs(t, ErrNoBOMsToMerge, err)
	})

	t.Run("merge multiple BOMs correctly", func(t *testing.T) {
		firstBOM := bomFromFile("../../integration/test/bomtools/bom-to-merge-1.json")
		secondBOM := bomFromFile("../../integration/test/bomtools/bom-to-merge-2.json")
		thirdBOM := bomFromFile("../../integration/test/bomtools/bom-to-merge-3.json")

		expectedBOM := bomFromFile("../../integration/test/bomtools/expected-merged-boms.json")
		got, err := MergeBoms(firstBOM, secondBOM, thirdBOM)
		require.NoError(t, err)

		assert.Equal(t, *expectedBOM.Components, *got.Components)
		assert.Equal(t, *expectedBOM.Metadata.Tools, *got.Metadata.Tools)
	})
}
