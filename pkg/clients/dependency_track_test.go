package clients

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyTrackClientCreation(t *testing.T) {
	t.Run("return an error when creating a client with invalid URL", func(t *testing.T) {
		client, err := NewDependencyTrackClient("https://invalid url.com", "token")

		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Nil(t, client)
	})

	t.Run("return an error when creating a client with empty API token", func(t *testing.T) {
		client, err := NewDependencyTrackClient("https://url.com", "")

		assert.Nil(t, client)
		assert.NotNil(t, err)
	})

	t.Run("apply request timeout option when it's set", func(t *testing.T) {
		const timeout = 666
		client, _ := NewDependencyTrackClient("https://url.com", "token", WithRequestTimeout(timeout))

		assert.Equal(t, time.Second*time.Duration(timeout), client.requestTimeout)
	})
}

// Test payload for SBOMs upload.
const (
	autoCreate   = true
	sbomsContent = "sample-sboms-for-testing"
	projectName  = "sample-project-name-for-testing"
)

// Helper function for uploading SBOMs.
func executeSBOMsUpload(t *testing.T, endpoint, apiKey string) error {
	t.Helper()

	client, err := NewDependencyTrackClient(endpoint, apiKey)
	if err != nil {
		t.Fatalf("can't create dependency track client: %s", err)
	}

	return client.UploadSBOMs(context.Background(), projectName, autoCreate, sbomsContent)
}

func TestUploadSBOMs(t *testing.T) {
	const apiKeyForTesting = "some-random-api-key"

	t.Run("construct HTTP payload correctly", func(t *testing.T) {
		type uploadSBOMsPayload struct {
			AutoCreate, Bom, ProjectName, ProjectVersion string
		}

		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)

			var got uploadSBOMsPayload
			require.NoError(t, json.NewDecoder(req.Body).Decode(&got)) // Decode into got var

			assert.NotEmpty(t, got.ProjectVersion)
			assert.Equal(t, projectName, got.ProjectName)
			assert.Equal(t, strconv.FormatBool(autoCreate), got.AutoCreate)
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(sbomsContent)), got.Bom)
		}))
		defer server.Close()

		_ = executeSBOMsUpload(t, server.URL, apiKeyForTesting)
	})

	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusTeapot)
		}))
		defer server.Close()

		err := executeSBOMsUpload(t, server.URL, apiKeyForTesting)

		var e BadStatusError
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
