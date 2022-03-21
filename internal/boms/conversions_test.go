package boms

import (
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

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
