package dtrack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/vinted/software-assets/pkg"
)

const (
	uploadSBOMsPath = "/api/v1/bom"
	projectPath     = "/api/v1/project"
)

// error templates.
const (
	cantMarshalPayload              = "can't convert provided payload into JSON: %w"
	cantUnmarshalResponse           = "can't decode JSON response from %s: %w"
	cantConstructHTTPRequest        = "can't construct HTTP request to %s: %w"
	cantPerformHTTPRequest          = "can't perform HTTP request to %s: %w"
	cantPerformRequestCantCloseBody = "request to %s failed: %w also failed to close response body: %v"
)

// appendURLPath append a specified path to the base URL & return the new URL.
func (d DependencyTrackClient) appendURLPath(pathToAppend string) string {
	baseURL, _ := url.Parse(d.baseURL) // Ignore error - we already validated base URL earlier on.
	baseURL.Path = path.Join(baseURL.Path, pathToAppend)

	return baseURL.String()
}

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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(cantMarshalPayload, err)
	}

	requestURL := d.appendURLPath(projectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, requestURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf(cantConstructHTTPRequest, requestURL, err)
	}

	d.setRequiredHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(cantPerformHTTPRequest, requestURL, err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if err != nil {
			if closeErr != nil {
				err = fmt.Errorf(cantPerformRequestCantCloseBody, requestURL, err, closeErr)
			}

			return
		}
		err = closeErr
	}()

	if resp.StatusCode != http.StatusCreated {
		// Don't return the error straight up - mind the defer above.
		err = pkg.BadStatusError{Status: resp.StatusCode, URL: requestURL}
		return "", err
	}

	type projectMetadata struct {
		UUID string
	}

	var metadata projectMetadata
	if err = json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		err = fmt.Errorf(cantUnmarshalResponse, requestURL, err)
		return "", err
	}

	return metadata.UUID, err // Return only UUID since it's the only relevant field we need from project creation.
}

// updateSBOMs updates SBOMs inside Dependency Track based on the payload supplied
func (d DependencyTrackClient) updateSBOMs(ctx context.Context, payload updateSBOMsPayload) error {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(cantMarshalPayload, err)
	}

	requestURL := d.appendURLPath(uploadSBOMsPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, requestURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf(cantConstructHTTPRequest, requestURL, err)
	}

	d.setRequiredHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(cantPerformHTTPRequest, requestURL, err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if err != nil {
			if closeErr != nil {
				err = fmt.Errorf(cantPerformRequestCantCloseBody, requestURL, err, closeErr)
			}
			return
		}
		err = closeErr
	}()

	if resp.StatusCode != http.StatusOK {
		// Don't return the error straight up - mind the defer above.
		err = pkg.BadStatusError{Status: resp.StatusCode, URL: requestURL}
		return err
	}

	return err
}

/*
UploadSBOMs upload SBOMs to Dependency Track based on the payload supplied. If the project doesn't exist - it is
automatically created.
*/
func (d DependencyTrackClient) UploadSBOMs(ctx context.Context, payload UploadSBOMsPayload) error {
	_, err := d.createProject(ctx, createProjectPayload{
		Tags:       payload.Tags,
		CodeOwners: payload.CodeOwners,
		Classifier: d.classifier,
		Name:       payload.ProjectName,
	})
	if err != nil {
		var e pkg.BadStatusError
		if ok := errors.As(err, &e); !ok {
			return err
		}
		if e.Status != http.StatusConflict {
			return err
		}
	}

	return d.updateSBOMs(ctx, updateSBOMsPayload{
		Sboms:       payload.Sboms,
		Tags:        payload.Tags,
		ProjectName: payload.ProjectName,
	})
}
