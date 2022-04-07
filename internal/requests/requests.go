package requests

/*
Package requests provides an API for:
Retrieving organization repositories from GitHub
Uploading generated BOM to dependency track.

Every request that this package performs is enhanced with exponential backoff.
*/

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vinted/software-assets/internal/vcs"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type BackoffConfig struct {
	RequestTimeout time.Duration
	BackoffPolicy  []time.Duration
}

type GetRepositoriesConfig struct {
	BackoffConfig
	URL, Username, APIToken string
}

type UploadBOMConfig struct {
	BackoffConfig
	AutoCreate                              bool
	URL, APIToken, ProjectName, BOMContents string
}

type BadStatusError struct {
	URL    string
	Status int
}

func (b BadStatusError) Error() string {
	return fmt.Sprintf("did not get 200 from %s, got %d", b.URL, b.Status)
}

func NewGetRepositoriesConfig(url, username, apiToken string) GetRepositoriesConfig {
	return GetRepositoriesConfig{
		URL:      url,
		Username: username,
		APIToken: apiToken,
		BackoffConfig: BackoffConfig{
			RequestTimeout: 10 * time.Second, //Good defaults
			BackoffPolicy:  []time.Duration{4 * time.Second, 8 * time.Second, 14 * time.Second},
		},
	}
}

func NewUploadBOMConfig(url, apiToken, projectName, bomContents string) UploadBOMConfig {
	return UploadBOMConfig{
		URL:         url,
		APIToken:    apiToken,
		AutoCreate:  true, // 99% of times we want this
		ProjectName: projectName,
		BOMContents: bomContents,
		BackoffConfig: BackoffConfig{
			RequestTimeout: 10 * time.Second, //Good defaults
			BackoffPolicy:  []time.Duration{4 * time.Second, 8 * time.Second, 14 * time.Second},
		},
	}
}

type response interface {
	[]vcs.Repository | bool
}

// Exponential backoff
func exponentialBackoff[T response](request func() (T, error), backoff ...time.Duration) (result T, err error) {
	shouldRetry := func(err error) bool {
		var e BadStatusError
		if ok := errors.As(err, &e); ok && e.Status == http.StatusTooManyRequests {
			return true
		}
		return errors.Is(err, context.DeadlineExceeded)
	}
	result, err = request()
	if err == nil {
		return result, nil
	}
	if !shouldRetry(err) {
		return result, err
	}

	for _, b := range backoff {
		time.Sleep(b)
		result, err = request()

		if err == nil {
			return result, nil
		}
		if !shouldRetry(err) {
			return result, err
		}
	}
	return result, err

}

//GetRepositories performs HTTP GET request to the provided GitHub URL.
//The provided URL should be in the form 'https://api.github.com/orgs/ORG-NAME/repos'.
//This function also takes a timeout for the HTTP request and an optional backoff varargs.
//If the backoff varargs are supplied and request fails, this function will reattempt the HTTP request
//with exponential backoff provided. The backoff kicks in only if the error is a timeout error or HTTP
//too many requests error. Returns a slice of repositories fetched or an error if something goes wrong.
func GetRepositories(conf GetRepositoriesConfig) ([]vcs.Repository, error) {
	getRepositories := func() ([]vcs.Repository, error) {
		ctx, cancel := context.WithTimeout(context.Background(), conf.RequestTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, conf.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to construct HTTP request: %w", err)
		}
		//req.SetBasicAuth(conf.Username, conf.APIToken)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP Request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, BadStatusError{Status: resp.StatusCode, URL: conf.URL}
		}
		var repositories []vcs.Repository
		if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
			return nil, fmt.Errorf("unable to parse JSON: %w", err)
		}
		return repositories, nil
	}

	return exponentialBackoff[[]vcs.Repository](getRepositories, conf.BackoffPolicy...)
}

func WalkRepositories(conf GetRepositoriesConfig, callback func(repos []vcs.Repository)) error {
	endpoint, err := url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("can't walk repository with malformed URL - %s: %w", conf.URL, err)
	}

	page := 1
	for {
		fmt.Printf("page - %d\n", page)
		//Update URL with the incremented page number each time
		query := endpoint.Query()
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		conf.URL = endpoint.String()

		repositories, err := GetRepositories(conf)
		if err != nil {
			return fmt.Errorf("repository walking failed: %w", err)
		}
		if len(repositories) == 0 {
			return nil //Done all repositories have been walked
		}
		callback(repositories)
		page++
	}
}

func UploadBOM(conf UploadBOMConfig) (bool, error) {
	uploadBOM := func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), conf.RequestTimeout)
		defer cancel()
		// Build the required payload for Dependency Track
		payload, err := json.Marshal(map[string]string{
			"projectName":    conf.ProjectName,
			"autoCreate":     strconv.FormatBool(conf.AutoCreate),
			"projectVersion": time.Now().Format("2006-01-02 15:04:05"),
			"bom":            base64.StdEncoding.EncodeToString([]byte(conf.BOMContents)),
		})
		//TODO Check what happens when we have multiple-lockfiles project versions, add a data time as a version
		if err != nil {
			return false, fmt.Errorf("unable to create JSON payload for uploading to Dependency track %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, conf.URL, bytes.NewBuffer(payload))
		if err != nil {
			return false, fmt.Errorf("unable to construct HTTP request: %w", err)
		}
		req.Header.Set("X-Api-Key", conf.APIToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("HTTP Request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, BadStatusError{Status: resp.StatusCode, URL: conf.URL}
		}
		return true, nil
	}

	return exponentialBackoff[bool](uploadBOM, conf.BackoffPolicy...)
}
