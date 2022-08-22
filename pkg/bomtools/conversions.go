package bomtools

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CycloneDX/cyclonedx-go"
)

// StringToCDX convert SBOM in JSON string format to cyclonedx.BOM instance.
func StringToCDX(bom []byte) (*cyclonedx.BOM, error) {
	cdx := new(cyclonedx.BOM)
	decoder := cyclonedx.NewBOMDecoder(strings.NewReader(string(bom)), cyclonedx.BOMFileFormatJSON)

	if err := decoder.Decode(cdx); err != nil {
		return nil, fmt.Errorf("can't decode string to cdx.BOM: %w", err)
	}

	return cdx, nil
}

// CDXToString convert cyclonedx.BOM instance to a JSON string.
func CDXToString(cdx *cyclonedx.BOM) (string, error) {
	bomBytes := &bytes.Buffer{}
	encoder := cyclonedx.NewBOMEncoder(bomBytes, cyclonedx.BOMFileFormatJSON)
	encoder.SetPretty(true)

	if err := encoder.Encode(cdx); err != nil {
		return "", fmt.Errorf("can't encode cdx.BOM to string: %w", err)
	}

	return bomBytes.String(), nil
}

// XMLStringToJSONCDX convert SBOM string in XML format to a cyclonedx.BOM instance with JSON type.
func XMLStringToJSONCDX(bom []byte) (*cyclonedx.BOM, error) {
	cdx := new(cyclonedx.BOM)
	decoder := cyclonedx.NewBOMDecoder(strings.NewReader(string(bom)), cyclonedx.BOMFileFormatXML)

	// Convert to cyclonedx.BOM instance first
	if err := decoder.Decode(cdx); err != nil {
		return nil, fmt.Errorf("can't decode XML string to cdx.BOM: %w", err)
	}

	// Convert to JSON string
	bomString, err := CDXToString(cdx)
	if err != nil {
		return nil, err
	}

	// Finally return a cyclonedx.BOM instance with JSON type.
	return StringToCDX([]byte(bomString))
}
