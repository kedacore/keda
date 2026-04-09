package fleetcontrol

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// CreateBlob creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlGetConfiguration(
	entityGUID string,
	organizationID string,
	getConfigurationMode GetConfigurationMode,
	version int,
) (*GetConfigurationResponse, error) {
	return a.FleetControlGetConfigurationWithContext(
		context.Background(),
		entityGUID,
		organizationID,
		getConfigurationMode,
		version,
	)
}

// CreatePolicyWithContext creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlGetConfigurationWithContext(
	ctx context.Context,
	entityGUID string,
	organizationID string,
	getConfigurationMode GetConfigurationMode,
	version int,
) (*GetConfigurationResponse, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("no organization ID specified")

	}

	versionQueryParameterAppender := ""
	if version >= 1 {
		versionQueryParameterAppender = fmt.Sprintf("?version=%d", version)
	}

	// Build the URL
	url := a.config.Region().BlobServiceURL(
		fmt.Sprintf(
			"/organizations/%s/%s/%s%s",
			organizationID,
			string(getConfigurationMode),
			entityGUID,
			versionQueryParameterAppender,
		))

	// Make a direct HTTP GET request to bypass JSON unmarshaling
	// The blob service returns plain text (YAML/JSON config), not JSON-encoded data
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers (API key)
	if a.config.PersonalAPIKey != "" {
		httpReq.Header.Set("Api-Key", a.config.PersonalAPIKey)
	}

	// Set Content-Type and User-Agent headers
	httpReq.Header.Set("Content-Type", "application/json")
	if a.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", a.config.UserAgent)
	}

	// Use the configured HTTP client to make the request
	// Note: We can't use a.client.Do() because it does JSON unmarshaling
	// So we create a basic HTTP client for this raw request
	httpClient := &http.Client{}
	if a.config.Timeout != nil {
		httpClient.Timeout = *a.config.Timeout
	}
	if a.config.HTTPTransport != nil {
		httpClient.Transport = a.config.HTTPTransport
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	// Check for error status codes
	if httpResp.StatusCode != http.StatusOK {
		if httpResp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("unexpected status code: %d", httpResp.StatusCode)
	}

	// Read the raw response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert to GetConfigurationResponse (string)
	resp := GetConfigurationResponse(string(body))
	return &resp, nil
}

// FleetControlGetConfigurationVersions retrieves all versions of a configuration.
func (a *Fleetcontrol) FleetControlGetConfigurationVersions(
	entityGUID string,
	organizationID string,
) (*GetConfigurationVersionsResponse, error) {
	return a.FleetControlGetConfigurationVersionsWithContext(
		context.Background(),
		entityGUID,
		organizationID,
	)
}

// FleetControlGetConfigurationVersionsWithContext retrieves all versions of a configuration with context.
func (a *Fleetcontrol) FleetControlGetConfigurationVersionsWithContext(
	ctx context.Context,
	entityGUID string,
	organizationID string,
) (*GetConfigurationVersionsResponse, error) {
	var resp GetConfigurationVersionsResponse

	if organizationID == "" {
		return nil, fmt.Errorf("no organization ID specified")
	}

	_, err := a.client.GetWithContext(
		ctx,
		a.config.Region().BlobServiceURL(
			fmt.Sprintf(
				"/organizations/%s/AgentConfigurations/%s/versions",
				organizationID,
				entityGUID,
			)),
		nil,
		&resp,
	)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateBlob creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlCreateConfiguration(
	requestBody interface{},
	customHeaders interface{},
	organizationID string,
) (*CreateConfigurationResponse, error) {
	return a.FleetControlCreateConfigurationWithContext(
		context.Background(),
		requestBody,
		customHeaders,
		organizationID,
	)
}

// CreatePolicyWithContext creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlCreateConfigurationWithContext(
	ctx context.Context,
	reqBody interface{},
	customHeaders interface{},
	organizationID string,
) (*CreateConfigurationResponse, error) {
	resp := CreateConfigurationResponse{}

	if organizationID == "" {
		return nil, fmt.Errorf("no organization ID specified")

	}

	_, err := a.client.PostWithContext(
		ctx,
		a.config.Region().BlobServiceURL(fmt.Sprintf("/organizations/%s/AgentConfigurations", organizationID)),
		customHeaders,
		reqBody,
		&resp,
	)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type CreateConfigurationResponse struct {
	BlobId                  string                     `json:"blobId,omitempty"`
	ConfigurationEntityGUID string                     `json:"entityGuid,omitempty"`
	ConfigurationVersion    ConfigurationVersionEntity `json:"blobVersionEntity,omitempty"`
}

type GetConfigurationResponse string

type GetConfigurationVersionsResponse struct {
	Versions []ConfigurationVersion `json:"versions"`
	Cursor   *string                `json:"cursor"`
}

type ConfigurationVersion struct {
	EntityGUID string `json:"entity_guid"`
	BlobID     string `json:"blob_id"`
	Version    string `json:"version"`
	Timestamp  string `json:"timestamp"`
}

type DeleteBlobResponse struct {
	Response string `json:"response,omitempty"`
}

type ConfigurationVersionEntity struct {
	ConfigurationVersionEntityGUID string `json:"entityGuid,omitempty"`
	ConfigurationVersionNumber     int    `json:"version,omitempty"`
}

type GetConfigurationMode string

var GetConfigurationModeTypes = struct {
	ConfigEntity        GetConfigurationMode
	ConfigVersionEntity GetConfigurationMode
}{
	ConfigEntity:        "AgentConfigurations",
	ConfigVersionEntity: "AgentConfigurationVersions",
}

// CreateBlob creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlDeleteConfiguration(
	blobEntityGUID string,
	organizationID string,
) (*DeleteBlobResponse, error) {
	return a.FleetControlDeleteConfigurationWithContext(
		context.Background(),
		blobEntityGUID,
		organizationID,
	)
}

// CreatePolicyWithContext creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlDeleteConfigurationWithContext(
	ctx context.Context,
	blobEntityGUID string,
	organizationID string,
) (*DeleteBlobResponse, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("no organization ID specified")

	}

	_, err := a.client.DeleteWithContext(
		ctx,
		a.config.Region().BlobServiceURL(fmt.Sprintf("/organizations/%s/AgentConfigurations/%s", organizationID, blobEntityGUID)),
		nil,
		nil, // No response body expected from configuration deletion
	)

	if err != nil {
		return nil, err
	}

	return &DeleteBlobResponse{}, nil
}

// CreateBlob creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlDeleteConfigurationVersion(
	configurationVersionGUID string,
	organizationID string,
) error {
	return a.FleetControlDeleteConfigurationVersionWithContext(
		context.Background(),
		configurationVersionGUID,
		organizationID,
	)
}

// CreatePolicyWithContext creates a new alert policy for a given account.
func (a *Fleetcontrol) FleetControlDeleteConfigurationVersionWithContext(
	ctx context.Context,
	configurationVersionGUID string,
	organizationID string,
) error {
	if organizationID == "" {
		return fmt.Errorf("no organization ID specified")

	}

	_, err := a.client.DeleteWithContext(
		ctx,
		a.config.Region().BlobServiceURL(fmt.Sprintf("/organizations/%s/AgentConfigurationVersions/%s", organizationID, configurationVersionGUID)),
		nil,
		nil, // No response body expected from version deletion
	)

	if err != nil {
		return err
	}

	return nil
}
