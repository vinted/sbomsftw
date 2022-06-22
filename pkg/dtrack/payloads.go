package dtrack

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
)

const (
	varcharMaxSize   = 255
	codeOwnersPrefix = "CODE OWNERS:\n"
)

type UploadSBOMsPayload struct {
	Sboms            *cdx.BOM
	ProjectName      string
	Tags, CodeOwners []string
}

type createProjectPayload struct {
	Name       string
	Tags       []string
	CodeOwners []string
}

func (c createProjectPayload) getCodeOwners() string {
	codeOwners := codeOwnersPrefix + strings.Join(c.CodeOwners, "\n")
	if (len(codeOwners)) > varcharMaxSize {
		return c.getTruncatedCodeOwners()
	}

	return codeOwners
}

////TODO: Temporary workaround! Dependency Track only supports project descriptions that are less then 255 characters.
////TODO: Remove this when DB is Altered from VARCHAR to TEXT column types
func (c createProjectPayload) getTruncatedCodeOwners() string {
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

func (c createProjectPayload) MarshalJSON() ([]byte, error) {
	type projectTag struct {
		Name string `json:"name"`
	}

	mappedTags := make([]projectTag, 0, len(c.Tags))

	for _, t := range c.Tags {
		mappedTags = append(mappedTags, projectTag{Name: t})
	}

	// project version is the SHA256 sum of all project tags concatenated with '/' + project name
	versionHash := sha256.Sum256([]byte(strings.Join(append(c.Tags, c.Name), "/")))

	return json.Marshal(map[string]any{
		"name":        c.Name,
		"tags":        mappedTags,
		"description": c.getCodeOwners(),
		"version":     fmt.Sprintf("%x", versionHash),
	})
}

type updateSBOMsPayload struct {
	Sboms       *cdx.BOM
	Tags        []string
	ProjectName string
}

func (c updateSBOMsPayload) MarshalJSON() ([]byte, error) {
	sbomsStr, err := bomtools.CDXToString(c.Sboms)
	if err != nil {
		return nil, fmt.Errorf("can't convert *cdx.BOM type Sboms to string")
	}

	// project version is the SHA256 sum of all project tags concatenated with '/' + project name
	versionHash := sha256.Sum256([]byte(strings.Join(append(c.Tags, c.ProjectName), "/")))

	return json.Marshal(map[string]string{
		"projectName":    c.ProjectName,
		"projectVersion": fmt.Sprintf("%x", versionHash),
		"bom":            base64.StdEncoding.EncodeToString([]byte(sbomsStr)),
	})
}
