package scalers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type openstackSwiftMetadata struct {
	SwiftURL              string `keda:"name=swiftURL,              order=triggerMetadata, optional"`
	ContainerName         string `keda:"name=containerName,         order=triggerMetadata"`
	ObjectCount           int64  `keda:"name=objectCount,           order=triggerMetadata, default=2"`
	ActivationObjectCount int64  `keda:"name=activationObjectCount, order=triggerMetadata, default=0"`
	ObjectPrefix          string `keda:"name=objectPrefix,          order=triggerMetadata, optional"`
	ObjectDelimiter       string `keda:"name=objectDelimiter,       order=triggerMetadata, optional"`
	ObjectLimit           string `keda:"name=objectLimit,           order=triggerMetadata, optional"`
	HTTPClientTimeout     int    `keda:"name=timeout,               order=triggerMetadata, default=30"`
	OnlyFiles             bool   `keda:"name=onlyFiles,             order=triggerMetadata, optional"`

	// Authentication fields
	UserID              string `keda:"name=userID,              order=authParams, optional"`
	Password            string `keda:"name=password,            order=authParams, optional"`
	ProjectID           string `keda:"name=projectID,           order=authParams, optional"`
	AuthURL             string `keda:"name=authURL,             order=authParams"`
	AppCredentialID     string `keda:"name=appCredentialID,     order=authParams, optional"`
	AppCredentialSecret string `keda:"name=appCredentialSecret, order=authParams, optional"`
	RegionName          string `keda:"name=regionName,          order=authParams, optional"`

	metricName   string
	triggerIndex int
}

func (m *openstackSwiftMetadata) Validate() error {
	if m.UserID != "" {
		if m.Password == "" {
			return fmt.Errorf("password must be specified when using userID")
		}
		if m.ProjectID == "" {
			return fmt.Errorf("projectID must be specified when using userID")
		}
		return nil
	}

	if m.AppCredentialID != "" {
		if m.AppCredentialSecret == "" {
			return fmt.Errorf("appCredentialSecret must be specified when using appCredentialID")
		}
		return nil
	}

	return fmt.Errorf("either userID or appCredentialID must be specified")
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

	// Initialize Gophercloud client
	var authOpts *gophercloud.AuthOptions

	// User chose the "application_credentials" authentication method
	if openstackSwiftMetadata.AppCredentialID != "" {
		authOpts = &gophercloud.AuthOptions{
			IdentityEndpoint:            openstackSwiftMetadata.AuthURL,
			ApplicationCredentialID:     openstackSwiftMetadata.AppCredentialID,
			ApplicationCredentialSecret: openstackSwiftMetadata.AppCredentialSecret,
		}
	} else if openstackSwiftMetadata.UserID != "" {
		// User chose the "password" authentication method
		authOpts = &gophercloud.AuthOptions{
			IdentityEndpoint: openstackSwiftMetadata.AuthURL,
			UserID:           openstackSwiftMetadata.UserID,
			Password:         openstackSwiftMetadata.Password,
			Scope: &gophercloud.AuthScope{
				ProjectID: openstackSwiftMetadata.ProjectID,
			},
		}
	}

	provider, err := openstack.AuthenticatedClient(*authOpts)
	if err != nil {
		return nil, fmt.Errorf("error getting openstack client: %w", err)
	}

	swiftClient, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{Region: openstackSwiftMetadata.RegionName})
	if err != nil {
		return nil, fmt.Errorf("error getting openstack swift client: %w", err)
	}

	if openstackSwiftMetadata.SwiftURL == "" {
		openstackSwiftMetadata.SwiftURL = swiftClient.Endpoint
	} else {
		swiftClient.Endpoint = openstackSwiftMetadata.SwiftURL
	}

	return &openstackSwiftScaler{
		metricType:  metricType,
		metadata:    openstackSwiftMetadata,
		swiftClient: swiftClient,
		logger:      logger,
	}, nil
}

func parseOpenstackSwiftMetadata(config *scalersconfig.ScalerConfig) (*openstackSwiftMetadata, error) {
	meta := &openstackSwiftMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing openstack swift metadata: %w", err)
	}

	if err := meta.Validate(); err != nil {
		return nil, err
	}

	var metricName string
	if meta.ObjectPrefix != "" {
		metricName = fmt.Sprintf("%s-%s", meta.ContainerName, meta.ObjectPrefix)
	} else {
		metricName = meta.ContainerName
	}
	metricName = kedautil.NormalizeString(fmt.Sprintf("openstack-swift-%s", metricName))
	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, metricName)

	return meta, nil
}

func (s *openstackSwiftScaler) Close(context.Context) error {
	if s.swiftClient != nil {
		s.swiftClient.HTTPClient.CloseIdleConnections()
	}
	return nil
}

func (s *openstackSwiftScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	containerName := s.metadata.ContainerName
	container, err := containers.Get(s.swiftClient, containerName, containers.GetOpts{}).Extract()
	if err != nil {
		s.logger.Error(err, "error getting container details")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	objectCount := container.ObjectCount
	metric := GenerateMetricInMili(metricName, float64(objectCount))

	return []external_metrics.ExternalMetricValue{metric}, objectCount > s.metadata.ActivationObjectCount, nil
}

func (s *openstackSwiftScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.ObjectCount),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}
