package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/akeylesslabs/akeyless-go/v5"
	akeyless_client "github.com/akeylesslabs/akeyless-go/v5"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	AUTH_ACCESS_KEY = "access_key"

	// TODO uncomment when supported by the API
	// AUTH_AWS_IAM            = "aws_iam"
	// AUTH_K8S                = "k8s"
	// AUTH_GCP                = "gcp"
	// AUTH_AZURE_AD           = "azure_ad"
	PUBLIC_GATEWAY_URL      = "https://api.akeyless.io"
	USER_AGENT              = "keda.sh"
	STATIC_SECRET_RESPONSE  = "STATIC_SECRET"
	DYNAMIC_SECRET_RESPONSE = "DYNAMIC_SECRET"
	ROTATED_SECRET_RESPONSE = "ROTATED_SECRET"
	ALL_SECRET_TYPES        = "all"
	CLIENT_SOURCE           = "akeylessclienttype"
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
		return errors.New("unable to authenticate with Akeyless")
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

	// TODO add support for other access types
	switch accessType {
	case AUTH_ACCESS_KEY:
		accessKey := h.akeyless.AccessKey
		if accessKey == nil {
			return errors.New("access key is required for access type 'access_key'")
		}
		authRequest.SetAccessKey(*accessKey)
	default:
		return errors.New("unsupported access type: " + accessType)
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
		return fmt.Errorf("failed to authenticate with Akeyless (HTTP status code: %d): %w", httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	h.logger.Info(fmt.Sprintf("authentication successful - token expires at %s", out.GetExpiration()))
	h.token = out.GetToken()
	return nil
}

func (h *AkeylessHandler) GetSecretsValue(ctx context.Context, secretResults map[string]string) (map[string]string, error) {
	for _, secret := range h.akeyless.Secrets {
		// Get the secret type
		secretType, err := h.GetSecretType(ctx, secret.Path)

		// if error getting secret type, return error
		if err != nil {
			return nil, fmt.Errorf("failed to get secret type for '%s': %w", secret.Path, err)
		}

		// if secret type is not supported, return error
		if !slices.Contains(supportedSecretTypes, secretType) {
			return nil, fmt.Errorf("unsupported secret type '%s' for '%s': supported secret types are: %s", secretType, secret.Path, strings.Join(supportedSecretTypes, ", "))
		}

		// Get the secret value
		switch secretType {
		case STATIC_SECRET_RESPONSE:
			getSecretValue := akeyless.NewGetSecretValue([]string{secret.Path})
			getSecretValue.SetToken(h.token)
			secretRespMap, _, apiErr := h.client.GetSecretValue(ctx).Body(*getSecretValue).Execute()
			if apiErr != nil {
				err = fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: %w", secret.Path, apiErr)
				break
			}

			// check if secret key is in response
			value, ok := secretRespMap[secret.Path]
			if !ok {
				err = fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: key not found", secret.Path)
				break
			}

			// single static secrets can be of type string, or map[string]string (e.g. key/value, username/password, JSON)
			// if it's a map[string]string, we use the provided key to get the value
			var secretValue string
			if strValue, ok := value.(string); ok {
				secretValue = strValue
			} else if mapValue, ok := value.(map[string]string); ok {
				if secret.Key != "" {
					err = fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: key not found", secret.Path)
					break
				}
				var found bool
				secretValue, found = mapValue[secret.Key]
				if !found {
					err = fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: key not found", secret.Path)
					break
				}
			} else {
				err = fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: value is not a string or map[string]string", secret.Path)
				break
			}

			secretResults[secret.Parameter] = secretValue

		case DYNAMIC_SECRET_RESPONSE:
			getDynamicSecretValue := akeyless.NewGetDynamicSecretValue(secret.Path)
			getDynamicSecretValue.SetToken(h.token)
			secretRespMap, _, apiErr := h.client.GetDynamicSecretValue(ctx).Body(*getDynamicSecretValue).Execute()
			if apiErr != nil {
				err = fmt.Errorf("failed to get dynamic secret '%s' value from Akeyless API: %w", secret.Path, apiErr)
				break
			}

			// Parse response to extract value and check for errors
			var dynamicSecretResp struct {
				Value string `json:"value"`
				Error string `json:"error"`
			}
			jsonBytes, marshalErr := json.Marshal(secretRespMap)
			if marshalErr != nil {
				err = fmt.Errorf("failed to marshal secret response to JSON: %w", marshalErr)
				break
			}
			if unmarshalErr := json.Unmarshal(jsonBytes, &dynamicSecretResp); unmarshalErr != nil {
				err = fmt.Errorf("failed to unmarshal secret response: %w", unmarshalErr)
				break
			}

			// Check if the response contains an error
			if dynamicSecretResp.Error != "" {
				err = fmt.Errorf("dynamic secret retrieval error: %s", dynamicSecretResp.Error)
				break
			}

			// Return the value field directly (already a JSON string with credentials)
			secretResults[secret.Parameter] = dynamicSecretResp.Value

		case ROTATED_SECRET_RESPONSE:
			getRotatedSecretValue := akeyless.NewGetRotatedSecretValue(secret.Path)
			getRotatedSecretValue.SetToken(h.token)
			secretRespMap, _, apiErr := h.client.GetRotatedSecretValue(ctx).Body(*getRotatedSecretValue).Execute()
			if apiErr != nil {
				err = fmt.Errorf("failed to get rotated secret '%s' value from Akeyless API: %w", secret.Path, apiErr)
				break
			}

			// Marshal the entire response value object
			jsonBytes, marshalErr := json.Marshal(secretRespMap)
			if marshalErr != nil {
				err = fmt.Errorf("failed to marshal rotated secret response to JSON: %w", marshalErr)
				break
			}
			secretResults[secret.Parameter] = string(jsonBytes)
		}
	}
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

// Utils

// Define constants for the access types. These are equivalent to the TypeScript consts.

// AccessTypeCharMap maps single-character access types to their display names.
var accessTypeCharMap = map[string]string{
	"a": AUTH_ACCESS_KEY,
	// TODO add support for other access types
	// "w": AUTH_IAM,
	// "k": AUTH_K8S,
	// "g": AUTH_GCP,
	// "z": AUTH_AZURE_AD,
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
