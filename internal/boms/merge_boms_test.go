package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestMergeBoms(t *testing.T) {

	bomFromFile := func(filename string) *cdx.BOM {
		bomString, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("can't read BOM from file: %s\n", err)
		}
		bom, err := BomStringToCDX(JSON, string(bomString))
		if err != nil {
			t.Fatalf("can't convert BOM string to *cdx.BOM file: %s\n", err)
		}
		return bom
	}

	t.Run("return an error when there are no BOMs to merge", func(t *testing.T) {
		got, err := MergeBoms([]*cdx.BOM{}...)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, UnableToMergeBOMsError("can't merge BOMs - empty list of BOMs supplied"))
	})
	t.Run("return an error when trying to merge nil boms", func(t *testing.T) {
		got, err := MergeBoms(nil, nil)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, UnableToMergeBOMsError("can't merge BOMs - BOM list can't contain elements"))
	})

	t.Run("merge multiple BOMs correctly", func(t *testing.T) {

		firstBOM := bomFromFile("../../integration/testdata/bom-to-merge-1.json")
		secondBOM := bomFromFile("../../integration/testdata/bom-to-merge-2.json")
		thirdBOM := bomFromFile("../../integration/testdata/bom-to-merge-3.json")

		expectedBOM := bomFromFile("../../integration/testdata/expected-merged-boms.json")
		got, err := MergeBoms(firstBOM, secondBOM, thirdBOM)
		require.NoError(t, err)

		assert.Equal(t, *expectedBOM.Components, *got.Components)
		assert.Equal(t, *expectedBOM.Dependencies, *got.Dependencies)
		assert.Equal(t, *expectedBOM.Metadata.Tools, *got.Metadata.Tools)
		assert.Equal(t, *expectedBOM.ExternalReferences, *got.ExternalReferences)
	})
}
