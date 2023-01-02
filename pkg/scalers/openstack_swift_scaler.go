package scalers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
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
	scalerIndex           int
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
	swiftClient openstack.Client
	logger      logr.Logger
}

func (s *openstackSwiftScaler) getOpenstackSwiftContainerObjectCount(ctx context.Context) (int64, error) {
	var containerName = s.metadata.containerName
	var swiftURL = s.metadata.swiftURL

	isValid, err := s.swiftClient.IsTokenValid(ctx)

	if err != nil {
		s.logger.Error(err, "scaler could not validate the token for authentication")
		return 0, err
	}

	if !isValid {
		err := s.swiftClient.RenewToken(ctx)

		if err != nil {
			s.logger.Error(err, "error requesting token for authentication")
			return 0, err
		}
	}

	token := s.swiftClient.Token

	swiftContainerURL, err := url.Parse(swiftURL)

	if err != nil {
		s.logger.Error(err, fmt.Sprintf("the swiftURL is invalid: %s. You might have forgotten to provide the either 'http' or 'https' in the URL. Check our documentation to see if you missed something", swiftURL))
		return 0, fmt.Errorf("the swiftURL is invalid: %w", err)
	}

	swiftContainerURL.Path = path.Join(swiftContainerURL.Path, containerName)

	swiftRequest, _ := http.NewRequestWithContext(ctx, "GET", swiftContainerURL.String(), nil)

	swiftRequest.Header.Set("X-Auth-Token", token)

	query := swiftRequest.URL.Query()
	query.Add("prefix", s.metadata.objectPrefix)
	query.Add("delimiter", s.metadata.objectDelimiter)

	// If scaler wants to scale based on only files, we first need to query all objects, then filter files and finally limit the result to the specified query limit
	if !s.metadata.onlyFiles {
		query.Add("limit", s.metadata.objectLimit)
	}

	swiftRequest.URL.RawQuery = query.Encode()

	resp, requestError := s.swiftClient.HTTPClient.Do(swiftRequest)

	if requestError != nil {
		s.logger.Error(requestError, fmt.Sprintf("error getting metrics for container '%s'. You probably specified the wrong swift URL or the URL is not reachable", containerName))
		return 0, requestError
	}

	defer resp.Body.Close()

	body, readError := io.ReadAll(resp.Body)

	if readError != nil {
		s.logger.Error(readError, "could not read response body from Swift API")
		return 0, readError
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		var objectsList = strings.Split(strings.TrimSpace(string(body)), "\n")

		// If onlyFiles is set to "true", return the total amount of files (excluding empty objects/folders)
		if s.metadata.onlyFiles {
			var count int64
			for i := 0; i < len(objectsList); i++ {
				if !strings.HasSuffix(objectsList[i], "/") {
					count++
				}
			}

			if s.metadata.objectLimit != defaultObjectLimit {
				objectLimit, conversionError := strconv.ParseInt(s.metadata.objectLimit, 10, 64)

				if conversionError != nil {
					s.logger.Error(err, fmt.Sprintf("the objectLimit value provided is invalid: %v", s.metadata.objectLimit))
					return 0, conversionError
				}

				if objectLimit <= count && s.metadata.objectLimit != defaultObjectLimit {
					return objectLimit, nil
				}
			}

			return count, nil
		}

		// Otherwise, if either prefix and/or delimiter are provided, return the total amount of objects
		if s.metadata.objectPrefix != defaultObjectPrefix || s.metadata.objectDelimiter != defaultObjectDelimiter {
			return int64(len(objectsList)), nil
		}

		// Finally, if nothing is set, return the standard total amount of objects inside the container
		objectCount, conversionError := strconv.ParseInt(resp.Header["X-Container-Object-Count"][0], 10, 64)
		return objectCount, conversionError
	}

	if resp.StatusCode == http.StatusUnauthorized {
		s.logger.Error(nil, "the retrieved token is not a valid token. Provide the correct auth credentials so the scaler can retrieve a valid access token (Unauthorized)")
		return 0, fmt.Errorf("the retrieved token is not a valid token. Provide the correct auth credentials so the scaler can retrieve a valid access token (Unauthorized)")
	}

	if resp.StatusCode == http.StatusForbidden {
		s.logger.Error(nil, "the retrieved token is a valid token, but it does not have sufficient permission to retrieve Swift and/or container metadata (Forbidden)")
		return 0, fmt.Errorf("the retrieved token is a valid token, but it does not have sufficient permission to retrieve Swift and/or container metadata (Forbidden)")
	}

	if resp.StatusCode == http.StatusNotFound {
		s.logger.Error(nil, fmt.Sprintf("the container '%s' does not exist (Not Found)", containerName))
		return 0, fmt.Errorf("the container '%s' does not exist (Not Found)", containerName)
	}

	return 0, fmt.Errorf(string(body))
}

// NewOpenstackSwiftScaler creates a new OpenStack Swift scaler
func NewOpenstackSwiftScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	var authRequest *openstack.KeystoneAuthRequest

	var swiftClient openstack.Client

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

	// User chose the "application_credentials" authentication method
	if authMetadata.appCredentialID != "" {
		authRequest, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialID, authMetadata.appCredentialSecret, openstackSwiftMetadata.httpClientTimeout)
		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %w", err)
		}
	} else {
		// User chose the "password" authentication method
		if authMetadata.userID != "" {
			authRequest, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, authMetadata.projectID, openstackSwiftMetadata.httpClientTimeout)
			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	if openstackSwiftMetadata.swiftURL == "" {
		// Request a Client with a token and the Swift API endpoint
		swiftClient, err = authRequest.RequestClient(ctx, "swift", authMetadata.regionName)

		if err != nil {
			return nil, fmt.Errorf("swiftURL was not provided and the scaler could not retrieve it dinamically using the OpenStack catalog: %w", err)
		}

		openstackSwiftMetadata.swiftURL = swiftClient.URL
	} else {
		// Request a Client with a token, but not the Swift API endpoint
		swiftClient, err = authRequest.RequestClient(ctx)

		if err != nil {
			return nil, err
		}

		swiftClient.URL = openstackSwiftMetadata.swiftURL
	}

	return &openstackSwiftScaler{
		metricType:  metricType,
		metadata:    openstackSwiftMetadata,
		swiftClient: swiftClient,
		logger:      logger,
	}, nil
}

func parseOpenstackSwiftMetadata(config *ScalerConfig) (*openstackSwiftMetadata, error) {
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
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func parseOpenstackSwiftAuthenticationMetadata(config *ScalerConfig) (*openstackSwiftAuthenticationMetadata, error) {
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
	return nil
}

func (s *openstackSwiftScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	objectCount, err := s.getOpenstackSwiftContainerObjectCount(ctx)

	if err != nil {
		s.logger.Error(err, "error getting objectCount")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

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
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.objectCount),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}
