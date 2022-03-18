package requests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/requests"
	"github.com/vinted/software-assets/internal/vcs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

//Error prefixes used for assertions
const (
	requestFailed         = "HTTP Request failed: "
	parsingJSONFailed     = "unable to parse JSON: "
	requestCreationFailed = "unable to construct HTTP request:"
	repositoryWalkFailed  = "can't walk repository with malformed URL -"
)

func TestExponentialBackoff(t *testing.T) {

	createUploadBOMConfig := func(url string) requests.UploadBOMConfig {
		return requests.UploadBOMConfig{
			URL:      url,
			APIToken: "test-token",
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 100 * time.Millisecond, // Time out after 100 millis
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}
	}

	t.Run("retry get repositories request 3 times on context.DeadlineExceeded errors", func(t *testing.T) {
		hitCounter := 0
		timeoutServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			time.Sleep(200 * time.Millisecond)
			res.WriteHeader(http.StatusOK)
		}))
		defer timeoutServer.Close()

		requestConfig := requests.GetRepositoriesConfig{
			URL:      timeoutServer.URL,
			APIToken: "test-token",
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 100 * time.Millisecond, // Time out after 100 millis
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}

		repositories, err := requests.GetRepositories(requestConfig)
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), requestFailed)
	})

	t.Run("retry upload bom request 3 times on context.DeadlineExceeded errors", func(t *testing.T) {
		hitCounter := 0
		timeoutServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			time.Sleep(200 * time.Millisecond)
			res.WriteHeader(http.StatusOK)
		}))
		defer timeoutServer.Close()

		ok, err := requests.UploadBOM(createUploadBOMConfig(timeoutServer.URL))
		assert.False(t, ok)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), requestFailed)
	})

	t.Run("retry request 3 times on http.StatusTooManyRequests errors", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		tooManyReqsServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTooManyRequests)
		}))
		defer tooManyReqsServer.Close()

		repositories, err := requests.UploadBOM(createUploadBOMConfig(tooManyReqsServer.URL))
		want := requests.BadStatusError{URL: tooManyReqsServer.URL, Status: http.StatusTooManyRequests}
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, want)
		assert.Equal(t, fmt.Sprintf(errorTemplate, tooManyReqsServer.URL, http.StatusTooManyRequests), err.Error())
	})
}

func TestGetRepositories(t *testing.T) {

	//Happy path
	t.Run("deserialize repositories correctly on successful response", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			content, _ := ioutil.ReadFile("test-data/sample-repos.json")
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(content)
		}))
		defer goodResponseServer.Close()

		repositories, err := requests.GetRepositories(createGetRepositoriesConfig(goodResponseServer.URL))
		assert.NoError(t, err)
		assert.Equal(t, 1, hitCounter)

		want := []vcs.Repository{
			{Name: "xmlsec", Description: "Ruby bindings for xmlsec", Archived: false, URL: "https://github.com/vinted/xmlsec"},
			{Name: "airbrake", Description: "Airbrake exceptions", Archived: true, URL: "https://github.com/vinted/airbrake-graylog2"},
			{Name: "dotpay", Description: "dotpay.pl gem", Archived: false, URL: "https://github.com/vinted/dotpay"},
		}
		assert.Equal(t, want, repositories)
	})

	//Errors path
	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		repositories, err := requests.GetRepositories(createGetRepositoriesConfig(teapotServer.URL))

		assert.ErrorIs(t, err, requests.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot})
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Equal(t, fmt.Sprintf(errorTemplate, teapotServer.URL, http.StatusTeapot), err.Error())
	})

	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		repositories, err := requests.GetRepositories(createGetRepositoriesConfig("http://bad url.com"))
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

		repositories, err := requests.GetRepositories(createGetRepositoriesConfig(invalidJSONServer.URL))
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
				content, _ = ioutil.ReadFile("test-data/repos-page-1.json")
			case "2":
				content, _ = ioutil.ReadFile("test-data/repos-page-2.json")
			case "3":
				content, _ = ioutil.ReadFile("test-data/repos-page-3.json")
			default:
				content = []byte("[]") //empty response
			}
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(content)
		}))
		defer goodResponseServer.Close()

		var collectedRepos []vcs.Repository
		reqConf := createGetRepositoriesConfig(goodResponseServer.URL)
		err := requests.WalkRepositories(reqConf, func(repos []vcs.Repository) {
			collectedRepos = append(collectedRepos, repos...)
		})
		assert.NoError(t, err)
		assert.Equal(t, 4, hitCounter)

		expectedRepos := []vcs.Repository{
			{Name: "xmlsec", Description: "Ruby bindings for xmlsec", URL: "https://github.com/vinted/xmlsec"},
			{Name: "airbrake", Description: "Airbrake exceptions", Archived: true, URL: "https://github.com/vinted/airbrake-graylog2"},
			{Name: "facebook-android-sdk", Description: "Android Facebook SDK", URL: "https://github.com/vinted/facebook-android-sdk"},
			{Name: "PhotoView", Description: "ImageView for Android", URL: "https://github.com/vinted/PhotoView"},
		}
		assert.Equal(t, expectedRepos, collectedRepos)
	})

	t.Run("return an error when request config contains invalid URL", func(t *testing.T) {
		err := requests.WalkRepositories(createGetRepositoriesConfig("http://bad url.com"), nil)
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Contains(t, err.Error(), repositoryWalkFailed)
	})
}

func TestUploadBOM(t *testing.T) {

	//Happy path
	t.Run("return true and nil error and successful server response", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusOK)
		}))
		defer goodResponseServer.Close()

		ok, err := requests.UploadBOM(createUploadBOMConfig(goodResponseServer.URL))
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	//Errors path
	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		ok, err := requests.UploadBOM(createUploadBOMConfig(teapotServer.URL))

		assert.False(t, ok)
		assert.ErrorIs(t, err, requests.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot})
	})

	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		ok, err := requests.UploadBOM(createUploadBOMConfig("http://bad url.com"))
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), requestCreationFailed)
	})
}

func createGetRepositoriesConfig(url string) requests.GetRepositoriesConfig {
	return requests.GetRepositoriesConfig{
		URL:      url,
		APIToken: "test-token",
		BackoffConfig: requests.BackoffConfig{
			RequestTimeout: 10 * time.Second,
			BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		},
	}
}

func createUploadBOMConfig(url string) requests.UploadBOMConfig {
	return requests.UploadBOMConfig{
		URL: url,
		BackoffConfig: requests.BackoffConfig{
			RequestTimeout: 10 * time.Second,
			BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		},
	}
}
