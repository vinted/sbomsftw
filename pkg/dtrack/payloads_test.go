package dtrack

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	t.Run("marshal CreateProjectPayload instances correctly", func(t *testing.T) {
		const (
			testProjectName      = "some-random-project-name"
			testProjectTag       = "some-random-project-tag"
			testProjectCodeOwner = "some-random-code-owner"
		)

		got, err := json.Marshal(CreateProjectPayload{
			Name:       testProjectName,
			CodeOwners: []string{testProjectCodeOwner},
			Tags:       []string{testProjectTag},
		})
		require.NoError(t, err)

		type projectTag struct {
			Name string `json:"name"`
		}

		type unmarshalled struct {
			Name       string       `json:"name"`
			Tags       []projectTag `json:"tags"`
			CodeOwners string       `json:"description"`
			Version    string       `json:"version"`
		}

		var u unmarshalled

		err = json.Unmarshal(got, &u)
		require.NoError(t, err)

		assert.Equal(t, testProjectName, u.Name)
		assert.Equal(t, "CODE OWNERS:\n"+testProjectCodeOwner, u.CodeOwners)
		assert.Equal(t, []projectTag{{Name: testProjectTag}}, u.Tags)
		// Don't check the actual date as it always changes - just assert on regex
		assert.Regexp(t, regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`), u.Version)
	})

	t.Run("marshal UploadSBOMsPayload instances correctly", func(t *testing.T) {
		const (
			projectUUID = "9488d60b-eb11-4569-8a7c-129d37b111e3"
			sbomsB64    = "ewogICJib21Gb3JtYXQiOiAiIiwKICAic3BlY1ZlcnNpb24iOiAiIiwKICAidmVyc2lvbiI6IDAKfQo="
		)

		got, err := json.Marshal(UploadSBOMsPayload{
			Sboms:       new(cdx.BOM), // zeroed mem value - b64 is always the same
			ProjectUUID: projectUUID,
		})
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("{\"bom\":\"%s\",\"project\":\"%s\"}", sbomsB64, projectUUID), string(got))
	})
}

func TestGetTruncatedCodeOwners(t *testing.T) {
	t.Run("filter out emails with 'users.noreply.github.com' domains", func(t *testing.T) {
		got := CreateProjectPayload{
			Name:       "some-random-name",
			Tags:       nil,
			CodeOwners: []string{"7+test@users.noreply.github.com", "john.doe@example.com", "jane.doe@pm.me"},
		}.getTruncatedCodeOwners()

		want := "CODE OWNERS:\njohn.doe@example.com\njane.doe@pm.me"
		assert.Equal(t, want, got)
	})

	t.Run("filter out emails that contain unicode characters", func(t *testing.T) {
		got := CreateProjectPayload{
			Name:       "some-random-name",
			Tags:       nil,
			CodeOwners: []string{"7+test@ačiū.com", "john.doe@example.com", "jane.doe@pm.me"},
		}.getTruncatedCodeOwners()

		want := "CODE OWNERS:\njohn.doe@example.com\njane.doe@pm.me"
		assert.Equal(t, want, got)
	})

	t.Run("order vinted.com contributors at the top", func(t *testing.T) {
		got := CreateProjectPayload{
			Name:       "some-random-name",
			Tags:       nil,
			CodeOwners: []string{"john.doe@pm.me", "john.smith@acme.com", "jane.doe@vinted.com", "jane@vinted.com"},
		}.getTruncatedCodeOwners()

		want := "CODE OWNERS:\njane.doe@vinted.com\njane@vinted.com\njohn.doe@pm.me\njohn.smith@acme.com"
		assert.Equal(t, want, got)
	})

	t.Run("truncate code owners to a maximum of 255 characters", func(t *testing.T) {
		got := CreateProjectPayload{
			Name:       "some-random-name",
			Tags:       nil,
			CodeOwners: []string{strings.Repeat("A", 300)}, // 300 A's
		}.getTruncatedCodeOwners()

		want := "CODE OWNERS:\n" + strings.Repeat("A", 242)
		assert.Equal(t, want, got)
	})
}
