package dtrack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vinted/sbomsftw/pkg"
)

const (
	exponentialBackoffRetryCount          = 5
	exponentialBackoffTimeoutMilliseconds = 500
)

const (
	uploadSBOMsPath = "/api/v1/bom"
	projectPath     = "/api/v1/project"
)

// error templates.
const (
	cantMarshalPayload              = "can't convert provided payload into JSON: %w"
	cantUnmarshalResponse           = "can't unmarshal JSON response from %s: %w"
	cantConstructHTTPRequest        = "can't construct HTTP request to %s: %w"
	cantPerformHTTPRequest          = "can't perform HTTP request to %s: %w"
	cantPerformRequestCantCloseBody = "request to %s failed: %w also failed to close response body: %v"
)

// setRequiredHeaders set mandatory HTTP headers for requests to Dependency Track to succeed.
func (d DependencyTrackClient) setRequiredHeaders(req *http.Request) {
	req.Header.Set("X-Api-Key", d.apiToken)
	req.Header.Set("Content-Type", "application/json")
}

/*
createProject create a project inside Dependency Track based on the payload supplied.
Upon successful project creation this method returns a project UUID. This UUID can later on
be used for SBOM upload.
*/
func (d DependencyTrackClient) createProject(ctx context.Context, payload createProjectPayload) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()

	dtCreateProjectEndpoint, err := url.JoinPath(d.dependencyTrackUrl, projectPath)
	if err != nil {
		return "", fmt.Errorf("error joining base URL: %w", err)
	}

	log.
		WithField("payloadName", payload.Name).
		Debugf("Create project with payload name: %s", payload.Name)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(cantMarshalPayload, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, dtCreateProjectEndpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf(cantConstructHTTPRequest, dtCreateProjectEndpoint, err)
	}

	d.setRequiredHeaders(request)
	response, err := d.performRequest(
		request,
		exponentialBackoffRetryCount,
		exponentialBackoffTimeoutMilliseconds*time.Millisecond,
	)

	if err != nil {
		return "", err
	}

	type projectMetadata struct {
		UUID string
	}

	var metadata projectMetadata
	if err = json.Unmarshal(response, &metadata); err != nil {
		err = fmt.Errorf(cantUnmarshalResponse, dtCreateProjectEndpoint, err)
		return "", err
	}

	log.
		WithField("metadataUUID", metadata.UUID).
		Debugf("Created a new Dependency Track project with UUID %s", metadata.UUID)

	return metadata.UUID, err
}

func (d DependencyTrackClient) updateDependencyTrackSBOMs(ctx context.Context, payload updateSBOMsPayload) error {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()

	dtUploadSBOMsEndpoint, err := url.JoinPath(d.dependencyTrackUrl, uploadSBOMsPath)
	if err != nil {
		return fmt.Errorf("error joining base URL: %w", err)
	}

	jsonPayload, err := json.Marshal(payload)
	log.
		WithField("funcType", "updateSBOM").
		Debugf("Updating project with payload: %s", payload.ProjectName)

	if err != nil {
		return fmt.Errorf(cantMarshalPayload, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, dtUploadSBOMsEndpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf(cantConstructHTTPRequest, dtUploadSBOMsEndpoint, err)
	}

	d.setRequiredHeaders(request)
	_, err = d.performRequest(
		request,
		exponentialBackoffRetryCount,
		exponentialBackoffTimeoutMilliseconds*time.Millisecond,
	)

	return err
}

func (
	d DependencyTrackClient,
) UploadSBOMs(
	ctx context.Context,
	payload UploadSBOMsPayload,
) error {
	if d.middleware {
		return d.uploadSBOMsToMiddleware(ctx, payload)
	}
	return d.uploadSBOMsToDependencyTrack(ctx, payload)
}

func (
	d DependencyTrackClient,
) uploadSBOMsToDependencyTrack(
	ctx context.Context,
	payload UploadSBOMsPayload,
) error {
	_, err := d.createProject(ctx, createProjectPayload{
		Tags:       payload.Tags,
		CodeOwners: payload.CodeOwners,
		Classifier: d.classifier,
		Name:       payload.ProjectName,
	})
	log.WithField("funcType", "uploadSBOM").Debugf("SBOM Create : %s", err)
	if err != nil {
		var e pkg.BadStatusError
		if ok := errors.As(err, &e); !ok {
			return err
		}
		if e.Status != http.StatusConflict {
			return err
		}
	}
	log.WithField("funcType", "uploadSBOM").Debugf("SBOM is performing an update")
	return d.updateDependencyTrackSBOMs(ctx, updateSBOMsPayload{
		Sboms:       payload.Sboms,
		Tags:        payload.Tags,
		ProjectName: payload.ProjectName,
	})
}

func (
	d DependencyTrackClient,
) uploadSBOMsToMiddleware(
	ctx context.Context,
	payload UploadSBOMsPayload,
) error {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = d.performPostRequestToMiddleware(
		ctx,
		"/sbom",
		jsonPayload,
	)
	if err != nil {
		return fmt.Errorf("error posting sbom to middleware - %w", err)
	}
	return nil
}

func (
	d DependencyTrackClient,
) performPostRequestToMiddleware(
	ctx context.Context,
	endpoint string,
	body []byte,
) ([]byte, error) {
	fullUrl, err := url.JoinPath(d.middlewareUrl, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error joining base URL: %w", err)
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing base URL: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedUrl.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("can't create HTTP request to %s: %w", fullUrl, err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth(d.middlewareUser, d.middlewarePass)

	return d.performRequest(
		request,
		exponentialBackoffRetryCount,
		exponentialBackoffTimeoutMilliseconds*time.Millisecond,
	)
}

func (
	d DependencyTrackClient,
) performRequest(
	request *http.Request,
	retryCount int,
	timeout time.Duration,
) ([]byte, error) {
	// Retry only if there were issues with the connection
	// or similar errors. If we managed to reach the server
	// and it provided us a response, we will accept that
	// response.
	for try := 1; try <= retryCount; try++ {
		response, err := d.httpClient.Do(request)

		if contextCancelled(err) {
			log.Warnf(
				"%s %s (%d/%d) | HTTP request cancelled...",
				request.Method,
				request.URL.String(),
				try,
				retryCount,
			)
			return nil, err
		}

		if err != nil {
			retryIn := getBackoffDuration(try, timeout)
			log.Warnf(
				"%s %s (%d/%d) | HTTP Request failed: \"%s\", retrying in %v...\n",
				request.Method,
				request.URL.String(),
				try,
				retryCount,
				err,
				retryIn,
			)
			time.Sleep(retryIn)
			continue
		}

		if !isResponseSuccess(response) {
			return nil, pkg.BadStatusError{
				URL:    request.URL.String(),
				Status: response.StatusCode,
			}
		}

		// TODO: This might require rethinking later. For now
		//       we just log that a failure happens, but we
		//       might want to return the error and mark this
		//       as a failure.
		defer func() {
			err := response.Body.Close()
			if err != nil {
				log.Warnf(
					"%s %s (%d/%d) | Closing HTTP Request failed: \"%s\"...\n",
					request.Method,
					request.URL.String(),
					try,
					retryCount,
					err,
				)
			}
		}()

		return io.ReadAll(response.Body)
	}

	return nil, fmt.Errorf(
		"%s %s | request failed for all %d attempts",
		request.Method,
		request.URL.String(),
		retryCount,
	)
}

func contextCancelled(err error) bool {
	if err == nil {
		return false
	}

	urlErr, ok := err.(*url.Error)
	if !ok {
		return false
	}

	return urlErr.Err == context.Canceled
}

func isResponseSuccess(response *http.Response) bool {
	return response.StatusCode >= 200 && response.StatusCode < 300
}

func getBackoffDuration(try int, timeout time.Duration) time.Duration {
	if try < 1 {
		return timeout
	}

	multiplier := math.Pow(2, float64(try))
	return timeout * time.Duration(multiplier)
}
