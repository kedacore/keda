package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/azure"
	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/gcp"
	"github.com/akeylesslabs/akeyless-go/v5"
	akeyless_client "github.com/akeylesslabs/akeyless-go/v5"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/mitchellh/mapstructure"
)

const (
	AUTH_ACCESS_KEY                = "access_key"
	AUTH_AWS_IAM                   = "aws_iam"
	AUTH_K8S                       = "k8s"
	AUTH_GCP                       = "gcp"
	AUTH_AZURE_AD                  = "azure_ad"
	PUBLIC_GATEWAY_URL             = "https://api.akeyless.io"
	USER_AGENT                     = "keda.sh"
	STATIC_SECRET_RESPONSE         = "STATIC_SECRET"
	DYNAMIC_SECRET_RESPONSE        = "DYNAMIC_SECRET"
	ROTATED_SECRET_RESPONSE        = "ROTATED_SECRET"
	ALL_SECRET_TYPES               = "all"
	CLIENT_SOURCE                  = "akeylessclienttype"
	K8S_SERVICE_ACCOUNT_TOKEN_FILE = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var supportedSecretTypes = []string{STATIC_SECRET_RESPONSE, DYNAMIC_SECRET_RESPONSE, ROTATED_SECRET_RESPONSE}

type AkeylessHandler struct {
	akeyless *kedav1alpha1.Akeyless
	client   *akeyless_client.V2ApiService
	token    string
	logger   logr.Logger
}

// NewAkeylessHandler creates a new AkeylessHandler
func NewAkeylessHandler(a *kedav1alpha1.Akeyless, logger logr.Logger) *AkeylessHandler {
	return &AkeylessHandler{
		akeyless: a,
		logger:   logger,
	}
}

// Initialize the AkeylessHandler
func (h *AkeylessHandler) Initialize(ctx context.Context) error {

	// Validate Gateway URL is not empty and is a valid URL
	if h.akeyless.GatewayUrl == "" {
		h.logger.Info(fmt.Sprintf("gatewayUrl is not set, using default value %s...", PUBLIC_GATEWAY_URL))
		h.akeyless.GatewayUrl = PUBLIC_GATEWAY_URL
	} else {
		_, err := url.ParseRequestURI(h.akeyless.GatewayUrl)
		if err != nil {
			return errors.New("invalid gateway URL '" + h.akeyless.GatewayUrl + "': " + err.Error())
		}
	}

	// Validate Access ID
	if h.akeyless.AccessId == "" {
		return errors.New("accessId is required")
	}

	h.logger.Info(fmt.Sprintf("initializing Akeyless handler '%s'...", h.akeyless.GatewayUrl))
	err := h.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("unable to authenticate with Akeyless: %w", err)
	}

	h.logger.Info("Akeyless handler initialized successfully")
	return nil
}

// Authenticate with Akeyless
func (h *AkeylessHandler) Authenticate(ctx context.Context) error {

	h.logger.Info(fmt.Sprintf("authenticating with Akeyless '%s' using Access ID '%s'...", h.akeyless.GatewayUrl, h.akeyless.AccessId))
	authRequest := akeyless_client.NewAuth()
	authRequest.SetAccessId(h.akeyless.AccessId)

	// Get the authentication method
	h.logger.Info("extracting access type from Access ID...")
	accessTypeChar, err := extractAccessTypeChar(h.akeyless.AccessId)
	if err != nil {
		return errors.New("unable to extract access type character from accessId, expected format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})")
	}

	h.logger.Info(fmt.Sprintf("getting access type display name for character '%s'...", accessTypeChar))
	accessType, err := getAccessTypeDisplayName(accessTypeChar)
	if err != nil {
		return errors.New("unable to get access type display name, expected format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})")
	}

	h.logger.Info(fmt.Sprintf("authenticating using access type '%s'...", accessType))

	switch accessType {
	case AUTH_ACCESS_KEY:
		accessKey := h.akeyless.AccessKey
		if accessKey == nil {
			return errors.New("access key is required for access type 'access_key'")
		}
		authRequest.SetAccessKey(*accessKey)
	case AUTH_AWS_IAM:
		authRequest.SetAccessType(AUTH_AWS_IAM)
		id, err := aws.GetCloudId()
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for AWS IAM: %w", err)
		}
		authRequest.SetCloudId(id)
	case AUTH_GCP:
		authRequest.SetAccessType(AUTH_GCP)
		// TODO add conf for audience
		id, err := gcp.GetCloudID("akeyless.io")
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for GCP: %w", err)
		}
		authRequest.SetCloudId(id)
	case AUTH_AZURE_AD:
		authRequest.SetAccessType(AUTH_AZURE_AD)
		// TODO add conf for object ID
		id, err := azure.GetCloudId("")
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for Azure AD: %w", err)
		}
		authRequest.SetCloudId(id)
	case AUTH_K8S:
		authRequest.SetAccessType(AUTH_K8S)

		if h.akeyless.K8sAuthConfigName == "" {
			return errors.New("k8sAuthConfigName is required for access type 'k8s'")
		}
		if h.akeyless.K8sGatewayUrl == "" {
			return errors.New("k8sGatewayUrl is required for access type 'k8s'")
		}

		// if k8s service account token is provided, try to read it from the file system
		if h.akeyless.K8sServiceAccountToken == "" {
			h.logger.Info(fmt.Sprintf("k8sServiceAccountToken is not provided, attempting to retrieve from file '%s'...", K8S_SERVICE_ACCOUNT_TOKEN_FILE))
			token, err := os.ReadFile(K8S_SERVICE_ACCOUNT_TOKEN_FILE)
			if err != nil {
				return fmt.Errorf("unable to read k8s service account token from file '%s': %w", K8S_SERVICE_ACCOUNT_TOKEN_FILE, err)
			}
			h.akeyless.K8sServiceAccountToken = string(token)
		}

		authRequest.SetK8sAuthConfigName(h.akeyless.K8sAuthConfigName)
		authRequest.SetK8sServiceAccountToken(h.akeyless.K8sServiceAccountToken)
		authRequest.SetGatewayUrl(h.akeyless.K8sGatewayUrl)

	default:
		return fmt.Errorf("unsupported access type: %s", accessType)
	}

	// Create Akeyless API client configuration
	// TODO add support for TLS
	h.logger.Info("creating Akeyless API client configuration...")
	config := akeyless.NewConfiguration()
	config.Servers = []akeyless.ServerConfiguration{
		{
			URL: h.akeyless.GatewayUrl,
		},
	}
	config.UserAgent = USER_AGENT
	config.AddDefaultHeader(CLIENT_SOURCE, USER_AGENT)

	h.client = akeyless.NewAPIClient(config).V2Api

	h.logger.Info("authenticating with Akeyless...")
	out, httpResponse, err := h.client.Auth(ctx).Body(*authRequest).Execute()
	if err != nil || httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate with Akeyless (HTTP status code: %d): %w", httpResponse.StatusCode, err)
	}

	h.logger.Info(fmt.Sprintf("authentication successful - token expires at %s", out.GetExpiration()))
	h.token = out.GetToken()
	return nil
}

func (h *AkeylessHandler) GetSecretsValue(ctx context.Context, secretResults map[string]string) (map[string]string, error) {
	h.logger.Info(fmt.Sprintf("getting secrets values for %d secrets...", len(h.akeyless.Secrets)))
	for _, secret := range h.akeyless.Secrets {
		var secretValue string
		// Get the secret type
		secretType, err := h.GetSecretType(ctx, secret.Path)
		h.logger.Info(fmt.Sprintf("getting secret type for '%s'...", secret.Path))
		// if error getting secret type, return error
		if err != nil {
			return nil, fmt.Errorf("failed to get secret type for '%s': %w", secret.Path, err)
		}
		h.logger.Info(fmt.Sprintf("secret type for '%s' is '%s'", secret.Path, secretType))
		// if secret type is not supported, return error
		if !slices.Contains(supportedSecretTypes, secretType) {
			return nil, fmt.Errorf("unsupported secret type '%s' for '%s': supported secret types are: %s", secretType, secret.Path, strings.Join(supportedSecretTypes, ", "))
		}
		h.logger.Info(fmt.Sprintf("secret type for '%s' is supported", secret.Path))
		// Get the secret value
		switch secretType {
		case STATIC_SECRET_RESPONSE:
			h.logger.Info(fmt.Sprintf("getting secret value for static secret '%s'...", secret.Path))
			getSecretValue := akeyless.NewGetSecretValue([]string{secret.Path})
			getSecretValue.SetToken(h.token)
			secretRespMap, _, apiErr := h.client.GetSecretValue(ctx).Body(*getSecretValue).Execute()
			if apiErr != nil {
				return nil, fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: %w", secret.Path, apiErr)
			}
			// check if secret key is in response
			value, ok := secretRespMap[secret.Path]
			if !ok {
				return nil, fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: %w", secret.Path, apiErr)
			}
			// single static secrets can be of type string, or map[string]string (e.g. key/value, username/password, JSON)
			// if it's a map[string]string, we use the provided key to get the value
			if strValue, ok := value.(string); ok {
				h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a string", secret.Path))
				secretValue = strValue
			} else if mapValue, ok := value.(map[string]string); ok {
				h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a map[string]string", secret.Path))
				// if the key is not provided, return stringified json
				if secret.Key == "" {
					h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a map[string]string and key is not provided, returning stringified JSON", secret.Path))
					jsonBytes, err := json.Marshal(mapValue)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal static secret '%s' value to JSON: %w", secret.Path, err)
					}
					secretValue = string(jsonBytes)
				} else {
					h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a map[string]string and key is provided, returning value for key '%s'", secret.Path, secret.Key))
					var found bool
					secretValue, found = mapValue[secret.Key]
					if !found {
						return nil, fmt.Errorf("failed to get secret '%s' value for static secret: key '%s' not found", secret.Path, secret.Key)
					}
				}
			} else {
				return nil, fmt.Errorf("failed to get secret '%s' value for static secret: unexpected value type: %T", secret.Path, value)
			}
		case DYNAMIC_SECRET_RESPONSE:
			h.logger.Info(fmt.Sprintf("getting dynamic secret value for '%s' from Akeyless API...", secret.Path))
			getDynamicSecretValue := akeyless.NewGetDynamicSecretValue(secret.Path)
			getDynamicSecretValue.SetToken(h.token)
			secretRespMap, httpResponse, apiErr := h.client.GetDynamicSecretValue(ctx).Body(*getDynamicSecretValue).Execute()
			if apiErr != nil {
				return nil, fmt.Errorf("failed to get dynamic secret '%s' value from Akeyless API: %w", secret.Path, apiErr)
			}
			if httpResponse.StatusCode != 200 {
				return nil, fmt.Errorf("failed to get dynamic secret '%s' value from Akeyless API (HTTP status code: %d): %w", secret.Path, httpResponse.StatusCode, errors.New(httpResponse.Status))
			}

			// Parse response to extract value and check for errors
			var dynamicSecretResp struct {
				Value string `json:"value"`
				Error string `json:"error"`
			}

			if err := mapstructure.Decode(secretRespMap, &dynamicSecretResp); err != nil {
				return nil, fmt.Errorf("dynamic secret '%s' response in unexpected format: %w", secret.Path, err)
			}

			if dynamicSecretResp.Value == "" {
				return nil, fmt.Errorf("dynamic secret '%s' response contains no value", secret.Path)
			}

			if dynamicSecretResp.Error != "" {
				return nil, fmt.Errorf("dynamic secret '%s' response contains an error: %s", secret.Path, dynamicSecretResp.Error)
			}

			// parse the value as a map[string]string
			var mapValue map[string]string
			if err := json.Unmarshal([]byte(dynamicSecretResp.Value), &mapValue); err != nil {
				return nil, fmt.Errorf("dynamic secret '%s' value in unexpected format: %w", secret.Path, err)
			}

			// if the key is not provided, return stringified json
			if secret.Key == "" {
				h.logger.Info(fmt.Sprintf("dynamic secret value for '%s' is a map[string]string but key is not provided, returning stringified JSON", secret.Path))
				jsonBytes, err := json.Marshal(mapValue)
				if err != nil {
					return nil, fmt.Errorf("dynamic secret '%s' value in unexpected format: %w", secret.Path, err)
				}
				secretValue = string(jsonBytes)
			} else {
				h.logger.Info(fmt.Sprintf("dynamic secret value for '%s' is a map[string]string and key is provided, returning value for key '%s'", secret.Path, secret.Key))
				var found bool
				secretValue, found = mapValue[secret.Key]
				if !found {
					return nil, fmt.Errorf("failed to get secret '%s' value for dynamic secret: key '%s' not found", secret.Path, secret.Key)
				}
			}
		case ROTATED_SECRET_RESPONSE:
			h.logger.Info(fmt.Sprintf("getting rotated secret value for '%s'...", secret.Path))
			getRotatedSecretValue := akeyless.NewGetRotatedSecretValue(secret.Path)
			getRotatedSecretValue.SetToken(h.token)
			secretRespMap, httpResponse, apiErr := h.client.GetRotatedSecretValue(ctx).Body(*getRotatedSecretValue).Execute()
			if apiErr != nil {
				return nil, fmt.Errorf("failed to get rotated secret '%s' value from Akeyless API: %w", secret.Path, apiErr)
			}
			if httpResponse.StatusCode != 200 {
				return nil, fmt.Errorf("failed to get rotated secret '%s' value from Akeyless API (HTTP status code: %d): %w", secret.Path, httpResponse.StatusCode, errors.New(httpResponse.Status))
			}

			var rotatedSecretResponse struct {
				Value map[string]string `json:"value"`
			}

			if err := mapstructure.Decode(secretRespMap, &rotatedSecretResponse); err != nil {
				return nil, fmt.Errorf("rotated secret '%s' response in unexpected format: %w", secret.Path, err)
			}

			if rotatedSecretResponse.Value == nil {
				return nil, fmt.Errorf("rotated secret '%s' response contains no value", secret.Path)
			}

			// if the key is not provided, return stringified json
			if secret.Key == "" {
				h.logger.Info(fmt.Sprintf("rotated secret value for '%s' is a map[string]string but key is not provided, returning stringified JSON", secret.Path))
				jsonBytes, err := json.Marshal(rotatedSecretResponse.Value)
				if err != nil {
					return nil, fmt.Errorf("rotated secret '%s' value in unexpected format: %w", secret.Path, err)
				}
				secretValue = string(jsonBytes)
			} else {
				var found bool
				secretValue, found = rotatedSecretResponse.Value[secret.Key]
				if !found {
					return nil, fmt.Errorf("failed to get secret '%s' value for rotated secret: key '%s' not found", secret.Path, secret.Key)
				}
			}
		}
		secretResults[secret.Parameter] = secretValue
	}
	h.logger.Info(fmt.Sprintf("returning %d secrets values", len(secretResults)))
	return secretResults, nil
}

func (h *AkeylessHandler) GetSecretType(ctx context.Context, secretName string) (string, error) {
	describeItem := akeyless.NewDescribeItem(secretName)
	describeItem.SetToken(h.token)
	describeItemResp, _, err := h.client.DescribeItem(ctx).Body(*describeItem).Execute()
	if err != nil {
		return "", fmt.Errorf("failed to describe item '%s': %w", secretName, err)
	}

	if describeItemResp.ItemType == nil {
		return "", errors.New("unable to retrieve secret type, missing type in describe item response")
	}

	return *describeItemResp.ItemType, nil
}

// AccessTypeCharMap maps single-character access types to their display names.
var accessTypeCharMap = map[string]string{
	"a": AUTH_ACCESS_KEY,
	"w": AUTH_AWS_IAM,
	// TODO add support for other access types
	// "k": AUTH_K8S,
	"g": AUTH_GCP,
	"z": AUTH_AZURE_AD,
}

// AccessIdRegex is the compiled regular expression for validating Akeyless Access IDs.
var accessIdRegex = regexp.MustCompile(`^p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})$`)

// isValidAccessIdFormat validates the format of an Akeyless Access ID.
// The format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12}).
// It returns true if the format is valid, and false otherwise.
func isValidAccessIdFormat(accessId string) bool {
	return accessIdRegex.MatchString(accessId)
}

// extractAccessTypeChar extracts the Akeyless Access Type character from a valid Access ID.
// The access type character is the second to last character of the ID part.
// It returns the single-character access type (e.g., 'a', 'o') or an empty string and an error if the format is invalid.
func extractAccessTypeChar(accessId string) (string, error) {
	if !isValidAccessIdFormat(accessId) {
		return "", errors.New("invalid access ID format")
	}
	parts := strings.Split(accessId, "-")
	idPart := parts[1] // Get the part after "p-"
	// The access type char is the second-to-last character
	return string(idPart[len(idPart)-2]), nil
}

// getAccessTypeDisplayName gets the full display name of the access type from the character.
// It returns the display name (e.g., 'api_key') or an error if the type character is unknown.
func getAccessTypeDisplayName(typeChar string) (string, error) {
	if typeChar == "" {
		return "", errors.New("unable to retrieve access type, missing type char")
	}
	displayName, ok := accessTypeCharMap[typeChar]
	if !ok {
		return "Unknown", errors.New("access type character not found in map")
	}
	return displayName, nil
}
