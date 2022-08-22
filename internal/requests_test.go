package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/vinted/software-assets/pkg"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Error prefixes used for assertions.
const (
	requestFailed         = "HTTP Request failed: "
	parsingJSONFailed     = "unable to parse JSON: "
	requestCreationFailed = "unable to construct HTTP request:"
	repositoryWalkFailed  = "can't walk repository with malformed URL -"
)

func TestExponentialBackoff(t *testing.T) {
	t.Run("retry get repositories request 3 times on context.DeadlineExceeded errors", func(t *testing.T) {
		hitCounter := 0
		timeoutServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			time.Sleep(200 * time.Millisecond)
			res.WriteHeader(http.StatusOK)
		}))
		defer timeoutServer.Close()

		requestConfig := GetRepositoriesConfig{
			ctx:      context.Background(),
			URL:      timeoutServer.URL,
			APIToken: "test-token",
			BackoffConfig: BackoffConfig{
				RequestTimeout: 100 * time.Millisecond, // Time out after 100 millis
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}

		repositories, err := GetRepositories(requestConfig)
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), requestFailed)
	})
}

func TestGetRepositories(t *testing.T) {
	// Happy path
	t.Run("deserialize repositories correctly on successful response", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			content, _ := os.ReadFile("../integration/testdata/requests/sample-repos.json")
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(content)
		}))
		defer goodResponseServer.Close()

		repositories, err := GetRepositories(createGetRepositoriesConfig(goodResponseServer.URL))
		require.NoError(t, err)
		assert.Equal(t, 1, hitCounter)

		want := []repositoryMapping{
			{Name: "xmlsec", Archived: false, Language: "C", URL: "https://github.com/vinted/xmlsec"},
			{Name: "dotpay", Archived: false, Language: "Ruby", URL: "https://github.com/vinted/dotpay"},
		}
		assert.Equal(t, want, repositories)
	})

	// Errors path
	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		const errorTemplate = "did not get a successful response from %s, got %d"
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		repositories, err := GetRepositories(createGetRepositoriesConfig(teapotServer.URL))

		assert.ErrorIs(t, err, pkg.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot})
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Equal(t, fmt.Sprintf(errorTemplate, teapotServer.URL, http.StatusTeapot), err.Error())
	})

	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		repositories, err := GetRepositories(createGetRepositoriesConfig("http://bad url.com"))
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Empty(t, repositories)
		assert.Contains(t, err.Error(), requestCreationFailed)
	})

	t.Run("returns json.SyntaxError error on invalid JSON response", func(t *testing.T) {
		hitCounter := 0
		invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write([]byte("Boo 👻 Invalid JSON"))
		}))
		defer invalidJSONServer.Close()

		repositories, err := GetRepositories(createGetRepositoriesConfig(invalidJSONServer.URL))
		var e *json.SyntaxError
		assert.ErrorAs(t, err, &e)
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Contains(t, err.Error(), parsingJSONFailed)
	})
}

func TestWalkRepositories(t *testing.T) {
	t.Run("walks repositories correctly on successful responses", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			var content []byte
			switch req.FormValue("page") {
			case "1":
				content, _ = os.ReadFile("../integration/testdata/requests/repos-page-1.json")
			case "2":
				content, _ = os.ReadFile("../integration/testdata/requests/repos-page-2.json")
			case "3":
				content, _ = os.ReadFile("../integration/testdata/requests/repos-page-3.json")
			default:
				content = []byte("[]") // empty response
			}
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(content)
		}))
		defer goodResponseServer.Close()

		var collectedRepos []string
		reqConf := createGetRepositoriesConfig(goodResponseServer.URL)
		err := WalkRepositories(reqConf, func(repos []string) {
			collectedRepos = append(collectedRepos, repos...)
		})
		require.NoError(t, err)
		assert.Equal(t, 4, hitCounter)

		expectedRepos := []string{
			"https://github.com/vinted/xmlsec",
			"https://github.com/vinted/airbrake-graylog2",
			"https://github.com/vinted/facebook-android-sdk",
			"https://github.com/vinted/PhotoView",
		}
		assert.Equal(t, expectedRepos, collectedRepos)
	})

	t.Run("return an error when request config contains invalid URL", func(t *testing.T) {
		err := WalkRepositories(createGetRepositoriesConfig("http://bad url.com"), nil)
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Contains(t, err.Error(), repositoryWalkFailed)
	})
}

func createGetRepositoriesConfig(url string) GetRepositoriesConfig {
	return GetRepositoriesConfig{
		ctx:                         context.Background(),
		URL:                         url,
		APIToken:                    "test-token",
		IncludeArchivedRepositories: false,
		BackoffConfig: BackoffConfig{
			RequestTimeout: 10 * time.Second,
			BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		},
	}
}
