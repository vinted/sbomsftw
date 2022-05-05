package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultRequestTimeout = 10

type BackoffConfig struct {
	RequestTimeout time.Duration
	BackoffPolicy  []time.Duration
}

type GetRepositoriesConfig struct {
	BackoffConfig
	ctx                         context.Context
	URL, Username, APIToken     string
	IncludeArchivedRepositories bool
}

type UploadBOMConfig struct {
	BackoffConfig
	ctx                                     context.Context
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

func NewGetRepositoriesConfig(ctx context.Context, url, username, apiToken string) GetRepositoriesConfig {
	return GetRepositoriesConfig{
		ctx:                         ctx,
		URL:                         url,
		Username:                    username,
		APIToken:                    apiToken,
		IncludeArchivedRepositories: false,
		BackoffConfig: BackoffConfig{
			RequestTimeout: defaultRequestTimeout * time.Second, // Good defaults
			BackoffPolicy:  []time.Duration{4 * time.Second, 8 * time.Second, 14 * time.Second},
		},
	}
}

func NewUploadBOMConfig(ctx context.Context, url, apiToken, projectName, bomContents string) UploadBOMConfig {
	return UploadBOMConfig{
		ctx:         ctx,
		URL:         url,
		APIToken:    apiToken,
		AutoCreate:  true, // 99% of times we want this
		ProjectName: projectName,
		BOMContents: bomContents,
		BackoffConfig: BackoffConfig{
			RequestTimeout: defaultRequestTimeout * time.Second, // Good defaults
			BackoffPolicy:  []time.Duration{4 * time.Second, 8 * time.Second, 14 * time.Second},
		},
	}
}

type repositoryMapping struct {
	Name     string
	Archived bool
	Language string
	URL      string `json:"html_url"`
}

type response interface {
	[]repositoryMapping | bool
}

// Exponential backoff.
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

// GetRepositories performs HTTP GET request to the provided GitHub URL.
// The provided URL should be in the form 'https://api.github.com/orgs/ORG-NAME/repos'.
// This function also takes a timeout for the HTTP request and an optional backoff varargs.
// If the backoff varargs are supplied and request fails, this function will reattempt the HTTP request
// with exponential backoff provided. The backoff kicks in only if the error is a timeout error or HTTP
// too many requests error. Returns a slice of repositories fetched or an error if something goes wrong.
func GetRepositories(conf GetRepositoriesConfig) ([]repositoryMapping, error) {
	getRepositories := func() ([]repositoryMapping, error) {
		ctx, cancel := context.WithTimeout(conf.ctx, conf.RequestTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, conf.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to construct HTTP request: %w", err)
		}
		req.SetBasicAuth(conf.Username, conf.APIToken)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP Request failed: %w", err)
		}

		defer func() {
			closeErr := resp.Body.Close()
			if err != nil {
				if closeErr != nil {
					err = fmt.Errorf("GetRepositories: %w can't close response body %v", err, closeErr)
				}

				return
			}
			err = closeErr
		}()

		if resp.StatusCode != http.StatusOK {
			err = BadStatusError{Status: resp.StatusCode, URL: conf.URL}
			return nil, err
		}

		var repositories []repositoryMapping
		if err = json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
			err = fmt.Errorf("unable to parse JSON: %w", err)
			return nil, err
		}

		if conf.IncludeArchivedRepositories {
			return repositories, nil
		}

		var validRepositories []repositoryMapping
		for _, r := range repositories {
			if !r.Archived {
				validRepositories = append(validRepositories, r)
			}
		}
		return validRepositories, nil
	}

	return exponentialBackoff(getRepositories, conf.BackoffPolicy...)
}

func WalkRepositories(conf GetRepositoriesConfig, callback func(repositoryURLs []string)) error {
	endpoint, err := url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("can't walk repository with malformed URL - %s: %w", conf.URL, err)
	}

	page := 1
	for {
		query := endpoint.Query()
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		conf.URL = endpoint.String()

		repositories, err := GetRepositories(conf)
		if err != nil {
			return fmt.Errorf("repository walking failed: %w", err)
		}
		if len(repositories) == 0 {
			return nil // Done all repositories have been walked
		}
		var repositoryURLs []string
		for _, r := range repositories {
			repositoryURLs = append(repositoryURLs, r.URL)
		}
		callback(repositoryURLs)
		page++
	}
}

// UploadBOM uploads BOM to Dependency Track based on the configuration given
func UploadBOM(conf UploadBOMConfig) (bool, error) {
	uploadBOM := func() (bool, error) {
		ctx, cancel := context.WithTimeout(conf.ctx, conf.RequestTimeout)
		defer cancel()
		// Build the required payload for Dependency Track
		payload, err := json.Marshal(map[string]string{
			"projectName":    conf.ProjectName,
			"autoCreate":     strconv.FormatBool(conf.AutoCreate),
			"projectVersion": time.Now().Format("2006-01-02 15:04:05"),
			"bom":            base64.StdEncoding.EncodeToString([]byte(conf.BOMContents)),
		})
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
			return false, fmt.Errorf("uploadBOM: HTTP Request failed: %w", err)
		}
		defer func() {
			closeErr := resp.Body.Close()
			if err != nil {
				if closeErr != nil {
					err = fmt.Errorf("UploadBOM: %w can't close response body %v", err, closeErr)
				}
				return
			}
			err = closeErr
		}()
		if resp.StatusCode != http.StatusOK {
			err = BadStatusError{Status: resp.StatusCode, URL: conf.URL}
			return false, err
		}
		return true, err
	}

	return exponentialBackoff(uploadBOM, conf.BackoffPolicy...)
}
