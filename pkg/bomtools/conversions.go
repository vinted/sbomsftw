package bomtools

import (
	"bytes"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"strings"
)

func StringToCDX(bom []byte) (*cyclonedx.BOM, error) {
	cdx := new(cyclonedx.BOM)
	decoder := cyclonedx.NewBOMDecoder(strings.NewReader(string(bom)), cyclonedx.BOMFileFormatJSON)
	if err := decoder.Decode(cdx); err != nil {
		return nil, fmt.Errorf("can't decode string to cdx.BOM: %w", err)
	}
	return cdx, nil
}

func CDXToString(cdx *cyclonedx.BOM) (string, error) {
	bomBytes := &bytes.Buffer{}
	encoder := cyclonedx.NewBOMEncoder(bomBytes, cyclonedx.BOMFileFormatJSON)
	encoder.SetPretty(true)
	if err := encoder.Encode(cdx); err != nil {
		return "", fmt.Errorf("can't encode cdx.BOM to string: %w", err)
	}
	return bomBytes.String(), nil
}
