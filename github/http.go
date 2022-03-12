package github

/*
Package github provides API for retrieving organization repositories and mapping them to types
that can later on be used in SBOM collection
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type BadStatusError struct {
	URL    string
	Status int
}

func (b BadStatusError) Error() string {
	return fmt.Sprintf("did not get 200 from %s, got %d", b.URL, b.Status)
}

//GetRepositories performs HTTP GET request to the provided GitHub URL.
//The provided URL should be in the form 'https://api.github.com/orgs/ORG-NAME/repos'.
//This function also takes a timeout for the HTTP request and an optional backoff varargs.
//If the backoff varargs are supplied and request fails, this function will reattempt the HTTP request
//with exponential backoff provided. The backoff kicks in only if the error is a timeout error or HTTP
//too many requests error. Returns a slice of repositories fetched or an error if something goes wrong.
func GetRepositories(url string, timeout time.Duration, backoff ...time.Duration) ([]Repository, error) {
	shouldRetry := func(err error) bool {
		var e BadStatusError
		if ok := errors.As(err, &e); ok && e.Status == http.StatusTooManyRequests {
			return true
		}
		return errors.Is(err, context.DeadlineExceeded)
	}

	var err error
	var repositories []Repository
	repositories, err = getRepositoriesInternal(url, timeout)
	if err == nil {
		return repositories, nil
	}
	if !shouldRetry(err) {
		return nil, err
	}

	for _, b := range backoff {
		time.Sleep(b)
		repositories, err = getRepositoriesInternal(url, timeout)

		if err == nil {
			return repositories, nil
		}
		if !shouldRetry(err) {
			return nil, err
		}
	}
	return nil, err
}

func getRepositoriesInternal(url string, timeout time.Duration) ([]Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to construct HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP Request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, BadStatusError{Status: resp.StatusCode, URL: url}
	}
	defer resp.Body.Close()
	var repositories []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %w", err)
	}
	return repositories, nil
}
