package dtrack

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

type DependencyTrackClient struct {
	baseURL, apiToken string
	requestTimeout    time.Duration
}

type options struct {
	requestTimeout time.Duration
}

type Option func(options *options) error

// WithRequestTimeout add a request timeout in seconds.
func WithRequestTimeout(requestTimeout int) Option {
	return func(options *options) error {
		if requestTimeout <= 0 {
			return errors.New("request timeout must be higher than zero")
		}

		options.requestTimeout = time.Second * time.Duration(requestTimeout)

		return nil
	}
}

// NewClient create a new Dependency Track client. A valid base URL & API token is required.
func NewClient(baseURL, apiToken string, opts ...Option) (*DependencyTrackClient, error) {
	if _, err := url.Parse(baseURL); err != nil { // Validate supplied URL early on
		return nil, fmt.Errorf("can't parse base URL: %w", err)
	}

	if apiToken == "" {
		return nil, errors.New("api token can't be empty")
	}

	var options options
	for _, opt := range opts {
		err := opt(&options)
		if err != nil {
			return nil, err
		}
	}

	client := new(DependencyTrackClient)

	// Mandatory parameters
	client.baseURL = baseURL
	client.apiToken = apiToken
	// Optional parameters
	const defaultTimeout = 60

	if options.requestTimeout == 0 { // If timeout is not provided - use default value
		client.requestTimeout = time.Second * time.Duration(defaultTimeout)
	} else {
		client.requestTimeout = options.requestTimeout
	}

	return client, nil
}
