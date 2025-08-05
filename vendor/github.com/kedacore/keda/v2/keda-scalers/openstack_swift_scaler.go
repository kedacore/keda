package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultOnlyFiles             = false
	defaultObjectCount           = 2
	defaultActivationObjectCount = 0
	defaultObjectLimit           = ""
	defaultObjectPrefix          = ""
	defaultObjectDelimiter       = ""
	defaultHTTPClientTimeout     = 30
)

type openstackSwiftMetadata struct {
	swiftURL              string
	containerName         string
	objectCount           int64
	activationObjectCount int64
	objectPrefix          string
	objectDelimiter       string
	objectLimit           string
	httpClientTimeout     int
	onlyFiles             bool
	triggerIndex          int
}

type openstackSwiftAuthenticationMetadata struct {
	userID              string
	password            string
	projectID           string
	authURL             string
	appCredentialID     string
	appCredentialSecret string
	regionName          string
}

type openstackSwiftScaler struct {
	metricType  v2.MetricTargetType
	metadata    *openstackSwiftMetadata
	swiftClient *gophercloud.ServiceClient
	logger      logr.Logger
}

// NewOpenstackSwiftScaler creates a new OpenStack Swift scaler
func NewOpenstackSwiftScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	var swiftClient *gophercloud.ServiceClient

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "openstack_swift_scaler")

	openstackSwiftMetadata, err := parseOpenstackSwiftMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift metadata: %w", err)
	}

	authMetadata, err := parseOpenstackSwiftAuthenticationMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift authentication metadata: %w", err)
	}

	// Initialize Gophercloud client
	var authOpts *gophercloud.AuthOptions

	// User chose the "application_credentials" authentication method
	if authMetadata.appCredentialID != "" {
		authOpts = &gophercloud.AuthOptions{
			IdentityEndpoint:            authMetadata.authURL,
			ApplicationCredentialID:     authMetadata.appCredentialID,
			ApplicationCredentialSecret: authMetadata.appCredentialSecret,
		}
	} else if authMetadata.userID != "" {
		// User chose the "password" authentication method
		authOpts = &gophercloud.AuthOptions{
			IdentityEndpoint: authMetadata.authURL,
			UserID:           authMetadata.userID,
			Password:         authMetadata.password,
			Scope: &gophercloud.AuthScope{
				ProjectID: authMetadata.projectID,
			},
		}
	}

	provider, err := openstack.AuthenticatedClient(*authOpts)
	if err != nil {
		return nil, fmt.Errorf("error getting openstack client: %w", err)
	}

	swiftClient, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{Region: authMetadata.regionName})
	if err != nil {
		return nil, fmt.Errorf("error getting openstack swift client: %w", err)
	}

	if openstackSwiftMetadata.swiftURL == "" {
		openstackSwiftMetadata.swiftURL = swiftClient.Endpoint
	} else {
		swiftClient.Endpoint = openstackSwiftMetadata.swiftURL
	}

	return &openstackSwiftScaler{
		metricType:  metricType,
		metadata:    openstackSwiftMetadata,
		swiftClient: swiftClient,
		logger:      logger,
	}, nil
}

func parseOpenstackSwiftMetadata(config *scalersconfig.ScalerConfig) (*openstackSwiftMetadata, error) {
	meta := openstackSwiftMetadata{}

	if val, ok := config.TriggerMetadata["swiftURL"]; ok {
		meta.swiftURL = val
	} else {
		meta.swiftURL = ""
	}

	if val, ok := config.TriggerMetadata["containerName"]; ok {
		meta.containerName = val
	} else {
		return nil, fmt.Errorf("no containerName was provided")
	}

	if val, ok := config.TriggerMetadata["objectCount"]; ok {
		objectCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("objectCount parsing error: %w", err)
		}
		meta.objectCount = objectCount
	} else {
		meta.objectCount = defaultObjectCount
	}

	if val, ok := config.TriggerMetadata["activationObjectCount"]; ok {
		activationObjectCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationObjectCount parsing error: %w", err)
		}
		meta.activationObjectCount = activationObjectCount
	} else {
		meta.activationObjectCount = defaultActivationObjectCount
	}

	if val, ok := config.TriggerMetadata["objectPrefix"]; ok {
		meta.objectPrefix = val
	} else {
		meta.objectPrefix = defaultObjectPrefix
	}

	if val, ok := config.TriggerMetadata["objectDelimiter"]; ok {
		meta.objectDelimiter = val
	} else {
		meta.objectDelimiter = defaultObjectDelimiter
	}

	if val, ok := config.TriggerMetadata["timeout"]; ok {
		httpClientTimeout, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("httpClientTimeout parsing error: %w", err)
		}
		meta.httpClientTimeout = httpClientTimeout
	} else {
		meta.httpClientTimeout = defaultHTTPClientTimeout
	}

	if val, ok := config.TriggerMetadata["onlyFiles"]; ok {
		isOnlyFiles, conversionError := strconv.ParseBool(val)
		if conversionError != nil {
			return nil, fmt.Errorf("onlyFiles parsing error: %s", conversionError.Error())
		}
		meta.onlyFiles = isOnlyFiles
	} else {
		meta.onlyFiles = defaultOnlyFiles
	}

	if val, ok := config.TriggerMetadata["objectLimit"]; ok {
		meta.objectLimit = val
	} else {
		meta.objectLimit = defaultObjectLimit
	}
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func parseOpenstackSwiftAuthenticationMetadata(config *scalersconfig.ScalerConfig) (*openstackSwiftAuthenticationMetadata, error) {
	authMeta := openstackSwiftAuthenticationMetadata{}

	if config.AuthParams["authURL"] != "" {
		authMeta.authURL = config.AuthParams["authURL"]
	} else {
		return nil, fmt.Errorf("authURL doesn't exist in the authParams")
	}

	if config.AuthParams["regionName"] != "" {
		authMeta.regionName = config.AuthParams["regionName"]
	} else {
		authMeta.regionName = ""
	}

	if config.AuthParams["userID"] != "" {
		authMeta.userID = config.AuthParams["userID"]

		if config.AuthParams["password"] != "" {
			authMeta.password = config.AuthParams["password"]
		} else {
			return nil, fmt.Errorf("password doesn't exist in the authParams")
		}

		if config.AuthParams["projectID"] != "" {
			authMeta.projectID = config.AuthParams["projectID"]
		} else {
			return nil, fmt.Errorf("projectID doesn't exist in the authParams")
		}
	} else {
		if config.AuthParams["appCredentialID"] != "" {
			authMeta.appCredentialID = config.AuthParams["appCredentialID"]

			if config.AuthParams["appCredentialSecret"] != "" {
				authMeta.appCredentialSecret = config.AuthParams["appCredentialSecret"]
			} else {
				return nil, fmt.Errorf("appCredentialSecret doesn't exist in the authParams")
			}
		} else {
			return nil, fmt.Errorf("neither userID or appCredentialID exist in the authParams")
		}
	}

	return &authMeta, nil
}

func (s *openstackSwiftScaler) Close(context.Context) error {
	if s.swiftClient != nil {
		s.swiftClient.HTTPClient.CloseIdleConnections()
	}
	return nil
}

func (s *openstackSwiftScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	containerName := s.metadata.containerName
	container, err := containers.Get(s.swiftClient, containerName, containers.GetOpts{}).Extract()
	if err != nil {
		s.logger.Error(err, "error getting container details")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	objectCount := container.ObjectCount
	metric := GenerateMetricInMili(metricName, float64(objectCount))

	return []external_metrics.ExternalMetricValue{metric}, objectCount > s.metadata.activationObjectCount, nil
}

func (s *openstackSwiftScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricName string

	if s.metadata.objectPrefix != "" {
		metricName = fmt.Sprintf("%s-%s", s.metadata.containerName, s.metadata.objectPrefix)
	} else {
		metricName = s.metadata.containerName
	}

	metricName = kedautil.NormalizeString(fmt.Sprintf("openstack-swift-%s", metricName))

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.objectCount),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}
