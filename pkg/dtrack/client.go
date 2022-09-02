package dtrack

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var ValidClassifiers = []string{
	"Application", "Container", "Device", "File",
	"Firmware", "Framework", "Library", "Operating_System",
}

/*
GetValidClassifiersString a string representation of all classifiers concatenated with an '/'. This function
is purely used for display purposes in CLI & error messages.
*/
func GetValidClassifiersString() string {
	return strings.Join(ValidClassifiers, "/")
}

type DependencyTrackClient struct {
	baseURL, apiToken string
	classifier        string
	requestTimeout    time.Duration
}

type options struct {
	classifier     string
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

func WithClassifier(classifier string) Option {
	return func(options *options) error {
		for _, c := range ValidClassifiers {
			if strings.EqualFold(c, classifier) {
				options.classifier = classifier

				return nil
			}
		}

		const errorTemplate = "%s is an invalid classifier, must be one of: %s"
		return fmt.Errorf(errorTemplate, classifier, GetValidClassifiersString())
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

	if options.classifier == "" {
		client.classifier = ValidClassifiers[0]
	} else {
		client.classifier = options.classifier
	}

	return client, nil
}
