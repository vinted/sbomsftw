package dtrack

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

const (
	varcharMaxSize   = 255
	codeOwnersPrefix = "CODE OWNERS:\n"
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
	CodeOwners []string
}

func (c CreateProjectPayload) getCodeOwners() string {
	codeOwners := codeOwnersPrefix + strings.Join(c.CodeOwners, "\n")
	if (len(codeOwners)) > varcharMaxSize {
		return c.getTruncatedCodeOwners()
	}

	return codeOwners
}

////TODO: Temporary workaround! Dependency Track only supports project descriptions that are less then 255 characters.
////TODO: Remove this when DB is Altered from VARCHAR to TEXT column types
func (c CreateProjectPayload) getTruncatedCodeOwners() string {
	isASCII := func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] > unicode.MaxASCII {
				return false
			}
		}

		return true
	}

	vintedContributors := make([]string, 0, len(c.CodeOwners))
	otherContributors := make([]string, 0, len(c.CodeOwners))

	for _, codeOwner := range c.CodeOwners {
		if !isASCII(codeOwner) || strings.HasSuffix(codeOwner, "@users.noreply.github.com") {
			continue
		}
		if strings.HasSuffix(codeOwner, "@vinted.com") { // Temporary solution
			vintedContributors = append(vintedContributors, codeOwner)
			continue
		}

		otherContributors = append(otherContributors, codeOwner)
	}

	codeOwnersConcat := codeOwnersPrefix + strings.Join(append(vintedContributors, otherContributors...), "\n")
	if len(codeOwnersConcat) <= varcharMaxSize {
		return codeOwnersConcat
	}

	return codeOwnersConcat[:varcharMaxSize]
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
		"description": c.getCodeOwners(),
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
