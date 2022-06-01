package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const (
	uploadSBOMsPath = "/api/v1/bom"
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

func NewDependencyTrackClient(baseURL, apiToken string, opts ...Option) (*DependencyTrackClient, error) {
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
	const defaultTimeout = 30

	if options.requestTimeout == 0 {
		client.requestTimeout = time.Second * time.Duration(defaultTimeout)
	} else {
		client.requestTimeout = options.requestTimeout
	}

	return client, nil
}

/*UploadSBOMs uploads SBOMs to Dependency Track.
projectName - Dependency Track project name to use.
autoCreate - Automatically create project in Dependency Track if it doesn't exist.
sboms - SBOMs to upload, must be a valid CycloneDX JSON string.
*/
func (d DependencyTrackClient) UploadSBOMs(ctx context.Context, projectName string, autoCreate bool, sboms string) error {
	baseURL, _ := url.Parse(d.baseURL) // Ignore error - we already validated base URL earlier on
	baseURL.Path = path.Join(baseURL.Path, uploadSBOMsPath)
	uploadURL := baseURL.String()

	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()
	// Build the required payload for Dependency Track
	payload, err := json.Marshal(map[string]string{
		"projectName":    projectName,
		"autoCreate":     strconv.FormatBool(autoCreate),
		"projectVersion": time.Now().Format("2006-01-02 15:04:05"),
		"bom":            base64.StdEncoding.EncodeToString([]byte(sboms)),
	})
	if err != nil {
		return fmt.Errorf("can't construct JSON payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("can't construct HTTP request: %w", err)
	}

	req.Header.Set("X-Api-Key", d.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if err != nil {
			if closeErr != nil {
				err = fmt.Errorf("UploadSBOMs: %w can't close response body %v", err, closeErr)
			}
			return
		}
		err = closeErr
	}()

	if resp.StatusCode != http.StatusOK {
		// Don't return the error straight up - mind the defer above
		err = BadStatusError{Status: resp.StatusCode, URL: uploadURL}

		return err
	}

	return err
}
