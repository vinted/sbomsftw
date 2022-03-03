package bomtools

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToCDX(t *testing.T) {
	t.Run("convert valid SBOM string to CycloneDX instance", func(t *testing.T) {
		bomString, err := os.ReadFile("../../integration/test/bomtools/sample-bom.json")
		require.NoError(t, err)

		cdx, err := StringToCDX(bomString)
		require.NoError(t, err)

		var gotPURLs []string
		for _, c := range *cdx.Components {
			gotPURLs = append(gotPURLs, c.PackageURL)
		}
		assert.ElementsMatch(t, []string{
			"pkg:gem/mustermann@1.1.1",
			"pkg:gem/rack@2.2.3",
		}, gotPURLs)
	})

	t.Run("return an error when converting invalid SBOM string", func(t *testing.T) {
		bom, err := StringToCDX([]byte("Invalid SBOM string"))
		require.Nil(t, bom)
		require.NotNil(t, err)
	})
}

func TestXMLStringToJSONCDX(t *testing.T) {
	bomString, err := os.ReadFile("../../integration/test/bomtools/sample-bom.xml")
	require.NoError(t, err)

	cdx, err := XMLStringToJSONCDX(bomString)
	require.NoError(t, err)

	var gotPURLs []string
	for _, c := range *cdx.Components {
		gotPURLs = append(gotPURLs, c.PackageURL)
	}
	assert.ElementsMatch(t, []string{
		"pkg:npm/ckeditor@4.0.1",
		"pkg:npm/jquery@2.1.4",
	}, gotPURLs)
}
