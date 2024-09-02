package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	gh "github.com/vinted/sbomsftw/pkg/github"

	"github.com/vinted/sbomsftw/pkg"
)

const defaultRequestTimeout = 10

type BackoffConfig struct {
	RequestTimeout time.Duration
	BackoffPolicy  []time.Duration
}

type GetRepositoriesConfig struct {
	BackoffConfig
	ctx                                   context.Context
	URL, Username, APIToken, Organization string
	IncludeArchivedRepositories           bool
}

func NewGetRepositoriesConfig(ctx context.Context, url, username, apiToken string, org string) GetRepositoriesConfig {
	if org != "" {
		apiToken = gh.GenerateGithubAppTokenInternal(org)
	}

	return GetRepositoriesConfig{
		ctx:                         ctx,
		URL:                         url,
		Username:                    username,
		APIToken:                    apiToken,
		Organization:                org,
		IncludeArchivedRepositories: false,
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
		var e pkg.BadStatusError
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
			err = pkg.BadStatusError{Status: resp.StatusCode, URL: conf.URL}
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

		return repositories, nil
	}
	return exponentialBackoff(getRepositories, conf.BackoffPolicy...)
}

func WalkRepositories(conf GetRepositoriesConfig, callback func(repositoryURLs []string, apiToken string), pageCount, pageIndex int64) error {
	var repositories []repositoryMapping
	var err error
	regenCount := 0
	repositoriesLen := 0

	endpoint, err := url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("can't walk repository with malformed URL - %s: %w", conf.URL, err)
	}

	start := (pageIndex * pageCount) + 1
	end := int(start + pageCount - 1)
	page := int(start)
	for {
		if page > end {
			log.WithField("request github", endpoint.String()).Infof("returning due to page limit, page: %d", page)
			return nil
		}
		query := endpoint.Query()
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		conf.URL = endpoint.String()
		log.WithField("request github", endpoint.String()).Infof("Getting query for page %d", page)
		repositories, err = GetRepositories(conf)
		repositoriesLen += len(repositories)
		if err != nil {
			if regenCount < 1 {
				token, errToken := RegenerateGithubToken(conf.Organization)
				if errToken != nil {
					return fmt.Errorf("repository walking regen failed: %w", errToken)
				}
				// Regenerate the token and retry
				conf.APIToken = token

				// Try fetching the repositories again with the new token
				repositories, err = GetRepositories(conf)
				if err != nil {
					return fmt.Errorf("repository walking after regen failed: %w", err)
				}
				regenCount += 1
			} else {
				return fmt.Errorf("repository walking failed: %w", err)
			}
		} else {
			// Reset regen count upon a successful fetch
			regenCount = 0
		}

		var validRepositories []repositoryMapping
		var archivedRepositories []repositoryMapping
		for _, r := range repositories {
			if !r.Archived {
				validRepositories = append(validRepositories, r)
			} else if r.Archived {
				archivedRepositories = append(archivedRepositories, r)
			}
		}

		if len(validRepositories) == 0 && len(archivedRepositories) == 0 {
			log.WithField("request github return", endpoint.String()).Infof("returning with page %d", page)
			return nil // Done, all repositories have been walked
		}

		var repositoryURLs []string
		for _, r := range validRepositories {
			repositoryURLs = append(repositoryURLs, r.URL)
		}
		log.Infof("total repository count scanned %d", repositoriesLen)
		callback(repositoryURLs, conf.APIToken)
		// reset regen count
		page++
	}
}

func RegenerateGithubToken(org string) (string, error) {
	log.Infof("Trying to generate github token for %s", org)
	if org == "" {
		return "", fmt.Errorf("no org specified to regen")
	}
	token := gh.GenerateGithubAppTokenInternal(org)
	if token == "" {
		return "", fmt.Errorf("could not fetch new github app token")
	}
	return token, nil
}
