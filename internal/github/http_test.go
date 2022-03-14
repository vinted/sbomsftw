package github_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vinted/software-assets/internal/github"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGetRepositoriesBackoff(t *testing.T) {
	var backoffSchedule = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
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

		repositories, err := github.GetRepositories(goodResponseServer.URL, 10*time.Second, backoffSchedule...)
		assert.NoError(t, err)
		assert.Equal(t, 1, hitCounter)

		want := []github.Repository{
			{Name: "xmlsec", Description: "Ruby bindings for xmlsec", URL: "https://github.com/vinted/xmlsec"},
			{Name: "airbrake", Description: "Airbrake exceptions", URL: "https://github.com/vinted/airbrake-graylog2"},
			{Name: "dotpay", Description: "dotpay.pl gem", URL: "https://github.com/vinted/dotpay"},
		}
		assert.Equal(t, want, repositories)
	})

	//Error paths
	t.Run("retry request 3 times on context.DeadlineExceeded errors", func(t *testing.T) {
		hitCounter := 0
		timeoutServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			time.Sleep(200 * time.Millisecond)
			res.WriteHeader(http.StatusOK)
		}))
		defer timeoutServer.Close()

		repositories, err := github.GetRepositories(timeoutServer.URL, 100*time.Millisecond, backoffSchedule...)
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

		repositories, err := github.GetRepositories(tooManyReqsServer.URL, 10*time.Second, backoffSchedule...)
		want := github.BadStatusError{URL: tooManyReqsServer.URL, Status: http.StatusTooManyRequests}
		assert.Empty(t, repositories)
		assert.Equal(t, 3, hitCounter)
		assert.ErrorIs(t, err, want)
		assert.Equal(t, fmt.Sprintf(errorTemplate, tooManyReqsServer.URL, http.StatusTooManyRequests), err.Error())
	})

	t.Run("returns url.InvalidHostError whenever URL is invalid", func(t *testing.T) {
		repositories, err := github.GetRepositories("http://bad url.com", 10*time.Second, backoffSchedule...)
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

		repositories, err := github.GetRepositories(invalidJSONServer.URL, 10*time.Second, backoffSchedule...)
		var e *json.SyntaxError
		assert.ErrorAs(t, err, &e)
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Contains(t, err.Error(), "unable to parse JSON: ")
	})

	t.Run("return BadStatusError on non 200 OK responses", func(t *testing.T) {
		const errorTemplate = "did not get 200 from %s, got %d"
		hitCounter := 0
		teapotServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hitCounter++
			res.WriteHeader(http.StatusTeapot)
		}))
		defer teapotServer.Close()
		repositories, err := github.GetRepositories(teapotServer.URL, 10*time.Second, backoffSchedule...)

		want := github.BadStatusError{URL: teapotServer.URL, Status: http.StatusTeapot}
		assert.ErrorIs(t, err, want)
		assert.Empty(t, repositories)
		assert.Equal(t, 1, hitCounter)
		assert.Equal(t, fmt.Sprintf(errorTemplate, teapotServer.URL, http.StatusTeapot), err.Error())
	})
}
