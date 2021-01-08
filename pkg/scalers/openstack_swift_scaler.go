package scalers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultOnlyFiles         = false
	defaultObjectCount       = 2
	defaultObjectLimit       = ""
	defaultObjectPrefix      = ""
	defaultObjectDelimiter   = ""
	defaultHTTPClientTimeout = 30
)

type openstackSwiftMetadata struct {
	swiftURL          string
	containerName     string
	objectCount       int
	objectPrefix      string
	objectDelimiter   string
	objectLimit       string
	httpClientTimeout int
	onlyFiles         bool
}

type openstackSwiftAuthenticationMetadata struct {
	userID              string
	password            string
	projectID           string
	authURL             string
	appCredentialID     string
	appCredentialSecret string
}

type openstackSwiftScaler struct {
	metadata     *openstackSwiftMetadata
	authMetadata *openstack.KeystoneAuthMetadata
}

var openstackSwiftLog = logf.Log.WithName("openstack_swift_scaler")

func (s *openstackSwiftScaler) getOpenstackSwiftContainerObjectCount() (int, error) {
	var token string
	var swiftURL string = s.metadata.swiftURL
	var containerName string = s.metadata.containerName

	isValid, validationError := openstack.IsTokenValid(*s.authMetadata)

	if validationError != nil {
		openstackSwiftLog.Error(validationError, "scaler could not validate the token for authentication")
		return 0, validationError
	}

	if !isValid {
		var tokenRequestError error
		token, tokenRequestError = s.authMetadata.GetToken()
		s.authMetadata.AuthToken = token
		if tokenRequestError != nil {
			openstackSwiftLog.Error(tokenRequestError, "error requesting token for authentication")
			return 0, tokenRequestError
		}
	}

	token = s.authMetadata.AuthToken

	swiftContainerURL, err := url.Parse(swiftURL)

	if err != nil {
		openstackSwiftLog.Error(err, fmt.Sprintf("the swiftURL is invalid: %s. You might have forgotten to provide the either 'http' or 'https' in the URL. Check our documentation to see if you missed something", swiftURL))
		return 0, fmt.Errorf("the swiftURL is invalid: %s", err.Error())
	}

	swiftContainerURL.Path = path.Join(swiftContainerURL.Path, containerName)

	swiftRequest, _ := http.NewRequest("GET", swiftContainerURL.String(), nil)

	swiftRequest.Header.Set("X-Auth-Token", token)

	query := swiftRequest.URL.Query()
	query.Add("prefix", s.metadata.objectPrefix)
	query.Add("delimiter", s.metadata.objectDelimiter)

	// If scaler wants to scale based on only files,  we first need to query all objects, then filter files and finally limit the result to the specified query limit
	if !s.metadata.onlyFiles {
		query.Add("limit", s.metadata.objectLimit)
	}

	swiftRequest.URL.RawQuery = query.Encode()

	resp, requestError := s.authMetadata.HTTPClient.Do(swiftRequest)

	if requestError != nil {
		openstackSwiftLog.Error(requestError, fmt.Sprintf("error getting metrics for container '%s'. You probably specified the wrong swift URL or the URL is not reachable", containerName))
		return 0, requestError
	}

	defer resp.Body.Close()

	body, readError := ioutil.ReadAll(resp.Body)

	if readError != nil {
		openstackSwiftLog.Error(readError, "could not read response body from Swift API")
		return 0, readError
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var objectsList = strings.Split(strings.TrimSpace(string(body)), "\n")

		// If onlyFiles is set to "true", return the total amount of files (excluding empty objects/folders)
		if s.metadata.onlyFiles {
			var count int = 0
			for i := 0; i < len(objectsList); i++ {
				if !strings.HasSuffix(objectsList[i], "/") {
					count++
				}
			}

			if s.metadata.objectLimit != defaultObjectLimit {
				objectLimit, conversionError := strconv.Atoi(s.metadata.objectLimit)

				if conversionError != nil {
					openstackSwiftLog.Error(err, fmt.Sprintf("the objectLimit value provided is invalid: %v", s.metadata.objectLimit))
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
			return len(objectsList), nil
		}

		// Finally, if nothing is set, return the standard total amount of objects inside the container
		objectCount, conversionError := strconv.Atoi(resp.Header["X-Container-Object-Count"][0])
		return objectCount, conversionError
	}

	if resp.StatusCode == 401 {
		openstackSwiftLog.Error(nil, "the retrieved token is not a valid token. Provide the correct auth credentials so the scaler can retrieve a valid access token (Unauthorized)")
		return 0, fmt.Errorf("the retrieved token is not a valid token. Provide the correct auth credentials so the scaler can retrieve a valid access token (Unauthorized)")
	}

	if resp.StatusCode == 403 {
		openstackSwiftLog.Error(nil, "the retrieved token is a valid token, but it does not have sufficient permission to retrieve Swift and/or container metadata (Forbidden)")
		return 0, fmt.Errorf("the retrieved token is a valid token, but it does not have sufficient permission to retrieve Swift and/or container metadata (Forbidden)")
	}

	if resp.StatusCode == 404 {
		openstackSwiftLog.Error(nil, fmt.Sprintf("the container '%s' does not exist (Not Found)", containerName))
		return 0, fmt.Errorf("the container '%s' does not exist (Not Found)", containerName)
	}

	return 0, fmt.Errorf(string(body))
}

// NewOpenstackSwiftScaler creates a new swift scaler
func NewOpenstackSwiftScaler(config *ScalerConfig) (Scaler, error) {
	var keystoneAuth *openstack.KeystoneAuthMetadata

	openstackSwiftMetadata, err := parseOpenstackSwiftMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift metadata: %s", err)
	}

	authMetadata, err := parseOpenstackSwiftAuthenticationMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift authentication metadata: %s", err)
	}

	// User chose the "application_credentials" authentication method
	if authMetadata.appCredentialID != "" {
		keystoneAuth, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialID, authMetadata.appCredentialSecret, openstackSwiftMetadata.httpClientTimeout)
		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %s", err)
		}
	} else {
		// User chose the "password" authentication method
		if authMetadata.userID != "" {
			keystoneAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, authMetadata.projectID, openstackSwiftMetadata.httpClientTimeout)
			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %s", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	return &openstackSwiftScaler{
		metadata:     openstackSwiftMetadata,
		authMetadata: keystoneAuth,
	}, nil
}

func parseOpenstackSwiftMetadata(config *ScalerConfig) (*openstackSwiftMetadata, error) {
	meta := openstackSwiftMetadata{}

	if val, ok := config.TriggerMetadata["swiftURL"]; ok {
		meta.swiftURL = val
	} else {
		return nil, fmt.Errorf("no swiftURL given")
	}

	if val, ok := config.TriggerMetadata["containerName"]; ok {
		meta.containerName = val
	} else {
		return nil, fmt.Errorf("no containerName was provided")
	}

	if val, ok := config.TriggerMetadata["objectCount"]; ok {
		targetObjectCount, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("objectCount parsing error: %s", err.Error())
		}
		meta.objectCount = targetObjectCount
	} else {
		meta.objectCount = defaultObjectCount
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
			return nil, fmt.Errorf("httpClientTimeout parsing error: %s", err.Error())
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

	return &meta, nil
}

func parseOpenstackSwiftAuthenticationMetadata(config *ScalerConfig) (*openstackSwiftAuthenticationMetadata, error) {
	authMeta := openstackSwiftAuthenticationMetadata{}

	if config.AuthParams["authURL"] != "" {
		authMeta.authURL = config.AuthParams["authURL"]
	} else {
		return nil, fmt.Errorf("authURL doesn't exist in the authParams")
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

func (s *openstackSwiftScaler) IsActive(ctx context.Context) (bool, error) {
	objectCount, err := s.getOpenstackSwiftContainerObjectCount()

	if err != nil {
		return false, err
	}

	return objectCount > 0, nil
}

func (s *openstackSwiftScaler) Close() error {
	return nil
}

func (s *openstackSwiftScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	objectCount, err := s.getOpenstackSwiftContainerObjectCount()

	if err != nil {
		openstackSwiftLog.Error(err, "error getting objectCount")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(objectCount), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *openstackSwiftScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetObjectCount := resource.NewQuantity(int64(s.metadata.objectCount), resource.DecimalSI)

	var metricName string

	if s.metadata.objectPrefix != "" {
		metricName = fmt.Sprintf("%s-%s", s.metadata.containerName, s.metadata.objectPrefix)
	} else {
		metricName = s.metadata.containerName
	}

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "openstack-swift", metricName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetObjectCount,
		},
	}

	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}
