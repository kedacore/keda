package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	serviceTypesAuthorityEndpoint = "https://service-types.openstack.org/service-types.json"
	defaultHTTPClientTimeout      = 30
)

type serviceTypesRequest struct {
	AllTypesByServiceType   map[string][]string       `json:"all_types_by_service_type"`
	Forward                 map[string][]string       `json:"forward"`
	PrimaryServiceByProject map[string]serviceMapping `json:"primary_service_by_project"`
	Reverse                 map[string]string         `json:"reverse"`
	ServiceTypesByProject   map[string][]string       `json:"service_types_by_project"`
	Services                []serviceMapping          `json:"services"`
	SHA                     string                    `json:"sha"`
	Version                 string                    `json:"version"`
}

type serviceMapping struct {
	Aliases      []string `json:"aliases,omitempty"`
	APIReference string   `json:"api_reference"`
	Project      string   `json:"project"`
	ServiceType  string   `json:"service_type"`
}

// GetServiceTypes retrieves all historical OpenStack Service Types for a given OpenStack project
func GetServiceTypes(ctx context.Context, projectName string) ([]string, error) {
	var serviceTypesRequest serviceTypesRequest

	var httpClient = kedautil.CreateHTTPClient(defaultHTTPClientTimeout*time.Second, false)

	var url = serviceTypesAuthorityEndpoint

	getServiceTypes, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		return []string{}, err
	}

	resp, err := httpClient.Do(getServiceTypes)

	if err != nil || resp.Status >= "300" {
		return []string{}, nil
	}

	defer resp.Body.Close()

	jsonErr := json.NewDecoder(resp.Body).Decode(&serviceTypesRequest)

	if jsonErr != nil {
		return []string{}, jsonErr
	}

	var serviceTypes = serviceTypesRequest.PrimaryServiceByProject[projectName].Aliases

	if len(serviceTypes) == 0 {
		var serviceType = serviceTypesRequest.PrimaryServiceByProject[projectName].ServiceType

		if serviceType != "" {
			return []string{serviceType}, nil
		}

		return []string{}, fmt.Errorf("project is not an official OpenStack project")
	}

	return serviceTypes, nil
}
