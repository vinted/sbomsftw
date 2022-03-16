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

func TestExponentialBackoff(t *testing.T) {
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
		assert.Contains(t, err.Error(), "HTTP Request failed: ")
	})

	t.Run("retry upload bom request 3 times on context.DeadlineExceeded errors", func(t *testing.T) {
		hitCounter := 0
		timeoutServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			time.Sleep(200 * time.Millisecond)
			res.WriteHeader(http.StatusOK)
		}))
		defer timeoutServer.Close()

		requestConfig := requests.UploadBOMConfig{
			URL:      timeoutServer.URL,
			APIToken: "test-token",
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 100 * time.Millisecond, // Time out after 100 millis
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}

		repositories, err := requests.UploadBOM(requestConfig)
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), "HTTP Request failed: ")
	})

	t.Run("retry request 3 times on http.StatusTooManyRequests errors", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		tooManyReqsServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTooManyRequests)
		}))
		defer tooManyReqsServer.Close()

		requestConfig := requests.UploadBOMConfig{
			URL:      tooManyReqsServer.URL,
			APIToken: "test-token",
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 100 * time.Millisecond, // Time out after 100 millis
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}

		repositories, err := requests.UploadBOM(requestConfig)
		want := requests.BadStatusError{URL: tooManyReqsServer.URL, Status: http.StatusTooManyRequests}
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, want)
		assert.Equal(t, fmt.Sprintf(errorTemplate, tooManyReqsServer.URL, http.StatusTooManyRequests), err.Error())
	})
}

func TestGetRepositories(t *testing.T) {

	createRequestConfig := func(url string) requests.GetRepositoriesConfig {
		return requests.GetRepositoriesConfig{
			URL:      url,
			APIToken: "test-token",
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 10 * time.Second,
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}
	}

	//Happy path
	t.Run("deserialize repositories correctly on successful response", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			content, _ := ioutil.ReadFile("sample-repos.json")
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(content)
		}))
		defer goodResponseServer.Close()

		repositories, err := requests.GetRepositories(createRequestConfig(goodResponseServer.URL))
		assert.NoError(t, err)
		assert.Equal(t, 1, hitCounter)

		want := []vcs.Repository{
			{Name: "xmlsec", Description: "Ruby bindings for xmlsec", URL: "https://github.com/vinted/xmlsec"},
			{Name: "airbrake", Description: "Airbrake exceptions", URL: "https://github.com/vinted/airbrake-graylog2"},
			{Name: "dotpay", Description: "dotpay.pl gem", URL: "https://github.com/vinted/dotpay"},
		}
		assert.Equal(t, want, repositories)
	})

	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		repositories, err := requests.GetRepositories(createRequestConfig(teapotServer.URL))

		assert.ErrorIs(t, err, requests.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot})
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Equal(t, fmt.Sprintf(errorTemplate, teapotServer.URL, http.StatusTeapot), err.Error())
	})

	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		repositories, err := requests.GetRepositories(createRequestConfig("http://bad url.com"))
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Empty(t, repositories)
		assert.Contains(t, err.Error(), "unable to construct HTTP request: ")
	})

	t.Run("returns json.SyntaxError error on invalid JSON response", func(t *testing.T) {
		hitCounter := 0
		invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write([]byte("Boo 👻 Invalid JSON"))
		}))
		defer invalidJSONServer.Close()

		repositories, err := requests.GetRepositories(createRequestConfig(invalidJSONServer.URL))
		var e *json.SyntaxError
		assert.ErrorAs(t, err, &e)
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Contains(t, err.Error(), "unable to parse JSON: ")
	})
}

func TestUploadBOM(t *testing.T) {

	createRequestConfig := func(url string) requests.UploadBOMConfig {
		return requests.UploadBOMConfig{
			URL: url,
			BackoffConfig: requests.BackoffConfig{
				RequestTimeout: 10 * time.Second,
				BackoffPolicy:  []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			},
		}
	}

	//Happy path
	t.Run("return true and nil error and successful server response", func(t *testing.T) {
		hitCounter := 0
		goodResponseServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusOK)
		}))
		defer goodResponseServer.Close()

		ok, err := requests.UploadBOM(createRequestConfig(goodResponseServer.URL))
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		ok, err := requests.UploadBOM(createRequestConfig(teapotServer.URL))

		assert.False(t, ok)
		assert.ErrorIs(t, err, requests.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot})
	})
	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		ok, err := requests.UploadBOM(createRequestConfig("http://bad url.com"))
		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "unable to construct HTTP request: ")
	})
}
