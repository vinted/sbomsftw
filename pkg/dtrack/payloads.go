package dtrack

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type BadStatusError struct {
	URL    string
	Status int
}

func (b BadStatusError) Error() string {
	return fmt.Sprintf("did not get a successful response from %s, got %d", b.URL, b.Status)
}

type CreateProjectPayload struct {
	Name       string
	Tags       []string
	CodeOwners string
}

func (c CreateProjectPayload) MarshalJSON() ([]byte, error) {
	type projectTag struct {
		Name string `json:"name"`
	}

	mappedTags := make([]projectTag, 0, len(c.Tags))

	for _, t := range c.Tags {
		mappedTags = append(mappedTags, projectTag{Name: t})
	}

	return json.Marshal(map[string]any{
		"name":        c.Name,
		"tags":        mappedTags,
		"description": c.CodeOwners,
		"version":     time.Now().Format("2006-01-02 15:04:05"),
	})
}

type UploadSBOMsPayload struct {
	Sboms       *cdx.BOM
	ProjectUUID string
}

func (c UploadSBOMsPayload) MarshalJSON() ([]byte, error) {
	sbomsStr, err := bomtools.CDXToString(c.Sboms)
	if err != nil {
		return nil, fmt.Errorf("can't convert *cdx.BOM type Sboms to string")
	}

	return json.Marshal(map[string]string{
		"project": c.ProjectUUID,
		"bom":     base64.StdEncoding.EncodeToString([]byte(sbomsStr)),
	})
}
