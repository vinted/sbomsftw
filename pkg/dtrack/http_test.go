package dtrack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vinted/sbomsftw/pkg"

	cdx "github.com/CycloneDX/cyclonedx-go"

	"github.com/stretchr/testify/assert"
)

const apiKeyForTesting = "some-random-api-key"

// Helper function for uploading SBOMs.
func executeSBOMsUpload(t *testing.T, endpoint, apiKey string) error {
	t.Helper()

	client, err := NewClient(endpoint, apiKey)
	if err != nil {
		t.Fatalf("can't create dependency track client: %s", err)
	}

	bom := &cdx.BOM{ // Address the error in the new() call.
		BOMFormat:   "CycloneDX",
		Version:     1,
		SpecVersion: cdx.SpecVersion(5),
	}

	payload := updateSBOMsPayload{
		Sboms:       bom,
		ProjectName: "some-random-project-name",
		Tags:        []string{"some-random-project-tag"},
	}

	return client.updateDependencyTrackSBOMs(context.Background(), payload)
}

// Helper function for creating a project
func executeCreateProject(t *testing.T, endpoint, apiKey string) (string, error) {
	t.Helper()

	client, err := NewClient(endpoint, apiKey)
	if err != nil {
		t.Fatalf("can't create dependency track client: %s", err)
	}

	return client.createProject(context.Background(), createProjectPayload{})
}

func TestCreateProject(t *testing.T) {
	t.Run("append '/api/v1/project' to base URL when creating project", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			t.Log(req.URL.Path)
			res.WriteHeader(http.StatusCreated)
			assert.Equal(t, "/api/v1/project", req.URL.Path)
		}))
		defer server.Close()

		_, _ = executeCreateProject(t, server.URL, apiKeyForTesting)
	})

	t.Run("return BadStatusError on non 201 CREATED responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusTeapot)
		}))
		defer server.Close()

		_, err := executeCreateProject(t, server.URL, apiKeyForTesting)

		var e pkg.BadStatusError
		assert.ErrorAs(t, err, &e)
	})

	t.Run("return json.SyntaxError error when JSON decoding fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusCreated) // Return StatusCreated & no JSON
		}))
		defer server.Close()

		_, err := executeCreateProject(t, server.URL, apiKeyForTesting)

		var syntaxError *json.SyntaxError
		if !errors.As(err, &syntaxError) {
			t.Error("Error returned is not json.SyntaxError")
		}
	})

	t.Run("set X-Api-Key & Content-Type request headers correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusCreated)
			assert.Equal(t, apiKeyForTesting, req.Header.Get("X-Api-Key"))
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		}))
		defer server.Close()

		_, _ = executeCreateProject(t, server.URL, apiKeyForTesting)
	})

	t.Run("return a project UUID after successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusCreated)
			_, _ = res.Write([]byte(`{
   "name":"some-random-name",
   "description":"some-random-description",
   "classifier":"APPLICATION",
   "uuid":"0035979c-22be-4718-9caf-17f27e05d1b1",
   "properties":[
      
   ],
   "tags":[
      {
         "name":"some-random-tag"
      }
   ],
   "active":true
}`)) // Ignore return values - we don't really care about them in this test
		}))
		defer server.Close()

		uuid, err := executeCreateProject(t, server.URL, apiKeyForTesting)

		assert.NoError(t, err)
		assert.Equal(t, "0035979c-22be-4718-9caf-17f27e05d1b1", uuid)
	})
}

func TestUploadSBOMs(t *testing.T) {
	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusTeapot)
		}))
		defer server.Close()

		err := executeSBOMsUpload(t, server.URL, apiKeyForTesting)

		var e pkg.BadStatusError
		assert.ErrorAs(t, err, &e)
	})

	t.Run("return nil error on successful 200 OK responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		err := executeSBOMsUpload(t, server.URL, apiKeyForTesting)
		assert.NoError(t, err)
	})

	t.Run("append '/api/v1/bom' to base URL when uploading SBOMs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)
			assert.Equal(t, "/api/v1/bom", req.URL.Path)
		}))
		defer server.Close()

		_ = executeSBOMsUpload(t, server.URL, apiKeyForTesting)
	})

	t.Run("set X-Api-Key & Content-Type request headers correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)
			assert.Equal(t, apiKeyForTesting, req.Header.Get("X-Api-Key"))
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		}))
		defer server.Close()

		_ = executeSBOMsUpload(t, server.URL, apiKeyForTesting)
	})
}
