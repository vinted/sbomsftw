package boms

import (
	"bytes"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"strconv"
	"strings"
)

type BOMType int

const (
	JSON BOMType = iota
	XML
)

func (d BOMType) String() string {
	switch d {
	case JSON:
		return "JSON"
	case XML:
		return "XML"
	default:
		return "Unknown type: " + strconv.Itoa(int(d))
	}
}

type BadBOMTypeError struct{ BOMType BOMType }

func (e BadBOMTypeError) Error() string {
	return fmt.Sprintf("unknown bom type %d", e.BOMType)
}

func BomStringToCDX(format BOMType, bom string) (*cdx.BOM, error) {
	var decoder cdx.BOMDecoder
	switch format {
	case XML:
		decoder = cdx.NewBOMDecoder(strings.NewReader(bom), cdx.BOMFileFormatXML)
	case JSON:
		decoder = cdx.NewBOMDecoder(strings.NewReader(bom), cdx.BOMFileFormatJSON)
	default:
		return nil, BadBOMTypeError{BOMType: format}
	}
	cdxBOM := new(cdx.BOM)
	if err := decoder.Decode(cdxBOM); err != nil {
		return nil, fmt.Errorf("unable to decode string to cdx.BOM: %w", err)
	}
	return cdxBOM, nil
}

func CdxToBOMString(format BOMType, cdxBOM *cdx.BOM) (string, error) {
	result := &bytes.Buffer{}
	var encoder cdx.BOMEncoder
	switch format {
	case XML:
		encoder = cdx.NewBOMEncoder(result, cdx.BOMFileFormatXML)
	case JSON:
		encoder = cdx.NewBOMEncoder(result, cdx.BOMFileFormatJSON)
	default:
		return "", BadBOMTypeError{BOMType: format}
	}
	encoder.SetPretty(true)
	if err := encoder.Encode(cdxBOM); err != nil {
		return "", fmt.Errorf("unable to encode cdx.BOM to string: %w", err)
	}
	return result.String(), nil
}

func ConvertBetweenTypes(inFormat BOMType, outFormat BOMType, bom string) (string, error) {
	cdxBOM, err := BomStringToCDX(inFormat, bom)
	if err != nil {
		return "", fmt.Errorf("can't convert bom from %s to %s: %w", inFormat, outFormat, err)
	}
	return CdxToBOMString(outFormat, cdxBOM)
}
