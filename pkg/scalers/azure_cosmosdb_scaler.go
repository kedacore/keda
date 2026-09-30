package scalers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	cosmosDBMetricType                     = "External"
	cosmosDBRestAPIVersion                 = "2018-12-31"
	cosmosDBSubStatusHeader                = "x-ms-substatus"
	cosmosDBPartitionKeyRangeGoneSubStatus = "1002"
)

type azureCosmosDBScaler struct {
	metricType           v2.MetricTargetType
	metadata             *azureCosmosDBMetadata
	cosmosClient         *cosmosDBClient
	logger               logr.Logger
	scalerConfig         *scalersconfig.ScalerConfig
	diagnosticsMutex     sync.Mutex
	diagnosticsClosed    bool
	diagnosticMetricName string
}

type azureCosmosDBMetadata struct {
	DatabaseID                string `keda:"name=databaseId,              order=triggerMetadata"`
	ContainerID               string `keda:"name=containerId,             order=triggerMetadata"`
	LeaseDatabaseID           string `keda:"name=leaseDatabaseId,         order=triggerMetadata"`
	LeaseContainerID          string `keda:"name=leaseContainerId,        order=triggerMetadata"`
	ProcessorName             string `keda:"name=processorName,           order=triggerMetadata"`
	Endpoint                  string `keda:"name=endpoint,                order=authParams;triggerMetadata, optional"`
	Connection                string `keda:"name=connection,              order=authParams;resolvedEnv;triggerMetadata, optional"`
	LeaseEndpoint             string `keda:"name=leaseEndpoint,           order=authParams;triggerMetadata, optional"`
	LeaseConnection           string `keda:"name=leaseConnection,         order=authParams;resolvedEnv;triggerMetadata, optional"`
	CosmosDBKey               string `keda:"name=cosmosDBKey,             order=authParams;resolvedEnv, optional"`
	LeaseCosmosDBKey          string `keda:"name=leaseCosmosDBKey,        order=authParams;resolvedEnv, optional"`
	TenantID                  string `keda:"name=tenantId,                order=authParams, optional"`
	ClientID                  string `keda:"name=clientId,                order=authParams, optional"`
	ClientSecret              string `keda:"name=clientSecret,            order=authParams, optional"`
	Threshold                 int64  `keda:"name=changeFeedLagThreshold,            order=triggerMetadata, default=100"`
	ActivationThreshold       int64  `keda:"name=activationChangeFeedLagThreshold,  order=triggerMetadata, default=0"`
	MaxActiveLeasesPerReplica int64  `keda:"name=maxActiveLeasesPerReplica, order=triggerMetadata, optional"`
	EnableHighAvailability    bool   `keda:"name=enableHighAvailability, order=triggerMetadata, default=false"`
	TriggerIndex              int
}

func (m *azureCosmosDBMetadata) Validate() error {
	if m.MaxActiveLeasesPerReplica < 0 {
		return fmt.Errorf("maxActiveLeasesPerReplica must be greater than zero when configured")
	}
	if m.EnableHighAvailability && m.MaxActiveLeasesPerReplica == 0 {
		return fmt.Errorf("enableHighAvailability requires maxActiveLeasesPerReplica")
	}
	if m.Threshold <= 0 {
		return fmt.Errorf("changeFeedLagThreshold must be greater than zero")
	}
	if m.ActivationThreshold < 0 {
		return fmt.Errorf("activationChangeFeedLagThreshold must be greater than or equal to zero")
	}
	if m.ActivationThreshold >= m.Threshold {
		return fmt.Errorf("activationChangeFeedLagThreshold must be less than changeFeedLagThreshold")
	}

	if m.LeaseConnection == "" && m.LeaseEndpoint == "" {
		if m.Connection != "" {
			m.LeaseConnection = m.Connection
		} else {
			m.LeaseEndpoint = m.Endpoint
			m.LeaseCosmosDBKey = m.CosmosDBKey
		}
	}
	return nil
}

// cosmosDBClient provides low-level access to Cosmos DB via the REST API
// for querying lease documents and reading the change feed.
type cosmosDBClient struct {
	httpClient          *http.Client
	dataEndpoint        string
	dataKey             string
	leaseEndpoint       string
	leaseKey            string
	leaseDatabaseID     string
	leaseContainerID    string
	databaseID          string
	containerID         string
	processorName       string
	cosmosDBResourceURL string
	credential          azcore.TokenCredential
	logger              logr.Logger
}

type leaseDocument struct {
	ID                string          `json:"id"`
	LeaseToken        string          `json:"LeaseToken"`
	ContinuationToken string          `json:"ContinuationToken"`
	Owner             string          `json:"Owner,omitempty"`
	Version           int             `json:"version"`
	FeedRange         json.RawMessage `json:"FeedRange"`
	epkRange          *cosmosDBEPKRange
}

type cosmosDBEPKRange struct {
	Min string
	Max string
}

type cosmosDBPartitionKeyRange struct {
	ID    string
	Range cosmosDBEPKRange
}

func parseCosmosDBEPKRange(raw json.RawMessage) (*cosmosDBEPKRange, error) {
	var bounds struct {
		Min            *string `json:"min"`
		Max            *string `json:"max"`
		IsMinInclusive *bool   `json:"isMinInclusive"`
		IsMaxInclusive *bool   `json:"isMaxInclusive"`
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		return nil, fmt.Errorf("invalid EPK range: %w", err)
	}
	if (bounds.IsMinInclusive != nil && !*bounds.IsMinInclusive) || (bounds.IsMaxInclusive != nil && *bounds.IsMaxInclusive) {
		return nil, fmt.Errorf("EPK range must be min-inclusive and max-exclusive")
	}
	return newCosmosDBEPKRange(bounds.Min, bounds.Max)
}

func newCosmosDBEPKRange(minInclusive, maxExclusive *string) (*cosmosDBEPKRange, error) {
	if minInclusive == nil || maxExclusive == nil {
		return nil, fmt.Errorf("EPK range requires min and max bounds")
	}
	minValue, maxValue := *minInclusive, *maxExclusive
	if strings.Trim(minValue+maxValue, "0123456789ABCDEF") != "" || minValue >= maxValue || maxValue > "FF" {
		return nil, fmt.Errorf("EPK range must have ordered uppercase hexadecimal bounds within [\"\", \"FF\")")
	}
	return &cosmosDBEPKRange{Min: minValue, Max: maxValue}, nil
}

func (l *leaseDocument) parseFeedRange() error {
	switch l.Version {
	case 0:
		return nil
	case 1:
		var feedRange struct {
			Range json.RawMessage `json:"Range"`
		}
		if err := json.Unmarshal(l.FeedRange, &feedRange); err != nil {
			return fmt.Errorf("invalid FeedRange for lease %q: %w", l.ID, err)
		}
		epkRange, err := parseCosmosDBEPKRange(feedRange.Range)
		if err != nil {
			return fmt.Errorf("invalid FeedRange for lease %q: %w", l.ID, err)
		}
		l.epkRange = epkRange
		return nil
	default:
		return fmt.Errorf("unsupported lease version %d for lease %q", l.Version, l.ID)
	}
}

type changeFeedResponse struct {
	StatusCode    int
	SubStatusCode string
	Items         []json.RawMessage
	SessionToken  string
}

// NewAzureCosmosDBScaler creates a new Azure Cosmos DB change feed scaler.
func NewAzureCosmosDBScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_cosmosdb_scaler")

	meta, err := parseAzureCosmosDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure cosmos db metadata: %w", err)
	}
	if meta.MaxActiveLeasesPerReplica > 0 && metricType != v2.AverageValueMetricType {
		return nil, fmt.Errorf("lease-capacity scaling requires metricType AverageValue")
	}

	cosmosClient, err := newCosmosDBClient(meta, config.TriggerMetadata, config.PodIdentity, logger, config.GlobalHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("error creating cosmos db client: %w", err)
	}

	return &azureCosmosDBScaler{
		metricType:   metricType,
		metadata:     meta,
		cosmosClient: cosmosClient,
		logger:       logger,
		scalerConfig: config,
	}, nil
}

func parseAzureCosmosDBMetadata(config *scalersconfig.ScalerConfig) (*azureCosmosDBMetadata, error) {
	meta := &azureCosmosDBMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}
	if _, configured := config.TriggerMetadata["maxActiveLeasesPerReplica"]; configured && meta.MaxActiveLeasesPerReplica <= 0 {
		return nil, fmt.Errorf("maxActiveLeasesPerReplica must be greater than zero when configured")
	}

	if meta.Endpoint == "" && meta.Connection == "" {
		return nil, fmt.Errorf("connection string or endpoint is required for the data account")
	}
	if meta.LeaseEndpoint == "" && meta.LeaseConnection == "" {
		return nil, fmt.Errorf("connection string or endpoint is required for the lease account")
	}

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		servicePrincipalParameterCount := 0
		for _, parameter := range []string{meta.TenantID, meta.ClientID, meta.ClientSecret} {
			if parameter != "" {
				servicePrincipalParameterCount++
			}
		}
		dataHasKey := meta.Connection != "" || meta.CosmosDBKey != ""
		leaseHasKey := meta.LeaseConnection != "" || meta.LeaseCosmosDBKey != ""
		if !dataHasKey || !leaseHasKey {
			if servicePrincipalParameterCount != 0 && servicePrincipalParameterCount != 3 {
				return nil, fmt.Errorf("tenantId, clientId and clientSecret must all be provided for service-principal authentication")
			}
			if servicePrincipalParameterCount != 3 {
				return nil, fmt.Errorf("a connection string, account key, or complete service-principal credentials are required for both data and lease accounts")
			}
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
	default:
		return nil, fmt.Errorf("pod identity %s not supported for azure cosmos db", config.PodIdentity.Provider)
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func newCosmosDBClient(meta *azureCosmosDBMetadata, triggerMetadata map[string]string, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger, httpTimeout time.Duration) (*cosmosDBClient, error) {
	if httpTimeout == 0 {
		httpTimeout = 30 * time.Second
	}

	client := &cosmosDBClient{
		httpClient:       kedautil.CreateHTTPClient(httpTimeout, false),
		leaseDatabaseID:  meta.LeaseDatabaseID,
		leaseContainerID: meta.LeaseContainerID,
		databaseID:       meta.DatabaseID,
		containerID:      meta.ContainerID,
		processorName:    meta.ProcessorName,
		logger:           logger,
	}

	// Resolve data endpoint and key
	if meta.Connection != "" {
		endpoint, key, err := parseCosmosDBConnectionString(meta.Connection)
		if err != nil {
			return nil, fmt.Errorf("error parsing connection string: %w", err)
		}
		client.dataEndpoint = endpoint
		client.dataKey = key
	} else if meta.Endpoint != "" {
		client.dataEndpoint = meta.Endpoint
		client.dataKey = meta.CosmosDBKey
	}

	// Resolve lease endpoint and key
	if meta.LeaseConnection != "" {
		endpoint, key, err := parseCosmosDBConnectionString(meta.LeaseConnection)
		if err != nil {
			return nil, fmt.Errorf("error parsing lease connection string: %w", err)
		}
		client.leaseEndpoint = endpoint
		client.leaseKey = key
	} else if meta.LeaseEndpoint != "" {
		client.leaseEndpoint = meta.LeaseEndpoint
		client.leaseKey = meta.LeaseCosmosDBKey
	}

	if client.dataEndpoint == "" || client.leaseEndpoint == "" {
		return nil, fmt.Errorf("failed to determine cosmos db endpoints")
	}

	if client.dataKey == "" || client.leaseKey == "" {
		credential, resourceURL, err := newCosmosDBTokenCredential(meta, triggerMetadata, podIdentity, logger, client.httpClient)
		if err != nil {
			return nil, err
		}
		client.credential = credential
		client.cosmosDBResourceURL = resourceURL
	}

	return client, nil
}

func newCosmosDBTokenCredential(meta *azureCosmosDBMetadata, triggerMetadata map[string]string, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger, transport policy.Transporter) (azcore.TokenCredential, string, error) {
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		cosmosDBResourceURL, credentialCloud, disableInstanceDiscovery, err := resolveCosmosDBServicePrincipalCloud(triggerMetadata)
		if err != nil {
			return nil, "", err
		}
		credential, err := azidentity.NewClientSecretCredential(meta.TenantID, meta.ClientID, meta.ClientSecret, &azidentity.ClientSecretCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud:     credentialCloud,
				Transport: transport,
			},
			DisableInstanceDiscovery: disableInstanceDiscovery,
		})
		if err != nil {
			return nil, "", fmt.Errorf("error creating service-principal credential: %w", err)
		}
		return credential, cosmosDBResourceURL, nil
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		cosmosDBResourceURL, err := resolveCosmosDBResourceURL(triggerMetadata)
		if err != nil {
			return nil, "", fmt.Errorf("error resolving cosmos db resource URL: %w", err)
		}
		credential, err := azure.NewChainedCredential(logger, podIdentity)
		if err != nil {
			return nil, "", fmt.Errorf("error creating azure credential for workload identity: %w", err)
		}
		return credential, cosmosDBResourceURL, nil
	default:
		return nil, "", fmt.Errorf("pod identity %s not supported for azure cosmos db", podIdentity.Provider)
	}
}

func resolveCosmosDBServicePrincipalCloud(triggerMetadata map[string]string) (string, azcloud.Configuration, bool, error) {
	cosmosDBResourceURL, err := resolveCosmosDBResourceURL(triggerMetadata)
	if err != nil {
		return "", azcloud.Configuration{}, false, fmt.Errorf("error resolving cosmos db resource URL: %w", err)
	}
	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(triggerMetadata)
	if err != nil {
		return "", azcloud.Configuration{}, false, fmt.Errorf("error resolving active directory endpoint: %w", err)
	}
	return cosmosDBResourceURL,
		azcloud.Configuration{ActiveDirectoryAuthorityHost: activeDirectoryEndpoint},
		strings.EqualFold(triggerMetadata["cloud"], azure.PrivateCloud),
		nil
}

func resolveCosmosDBResourceURL(triggerMetadata map[string]string) (string, error) {
	return azure.ParseEnvironmentProperty(triggerMetadata, "cosmosDBResourceURL", func(env azure.AzEnvironment) (string, error) {
		return env.ResourceIdentifiers.CosmosDB, nil
	})
}

func parseCosmosDBConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")
	var endpoint, key string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "AccountEndpoint=") {
			endpoint = strings.TrimPrefix(part, "AccountEndpoint=")
		} else if strings.HasPrefix(part, "AccountKey=") {
			key = strings.TrimPrefix(part, "AccountKey=")
		}
	}

	if endpoint == "" || key == "" {
		return "", "", fmt.Errorf("invalid connection string: missing AccountEndpoint or AccountKey")
	}

	return endpoint, key, nil
}

// cosmosDBAccountShortName extracts the short account name from a Cosmos DB endpoint,
// e.g. "https://myaccount.documents.azure.com:443/" -> "myaccount". Every Cosmos DB account
// hostname has the form "{account}.<domain-suffix>", so this is stable across clouds
// (public, sovereign, or private) without needing to know the exact domain suffix.
func cosmosDBAccountShortName(endpoint string) string {
	host := endpoint
	if u, err := url.Parse(endpoint); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	if idx := strings.IndexByte(host, '.'); idx >= 0 {
		return host[:idx]
	}
	return host
}

// setAuthHeader sets the Authorization header using either master key HMAC-SHA256 or bearer token.
func (c *cosmosDBClient) setAuthHeader(req *http.Request, verb, resourceType, resourceLink, date, key string) error {
	if key != "" {
		token, err := generateCosmosDBAuthToken(verb, resourceType, resourceLink, date, key)
		if err != nil {
			return fmt.Errorf("error generating auth token: %w", err)
		}
		req.Header.Set("Authorization", token)
		return nil
	}

	if c.credential != nil {
		tk, err := c.credential.GetToken(req.Context(), policy.TokenRequestOptions{
			Scopes: []string{strings.TrimSuffix(c.cosmosDBResourceURL, "/") + "/.default"},
		})
		if err != nil {
			return fmt.Errorf("error acquiring bearer token: %w", err)
		}
		req.Header.Set("Authorization", url.QueryEscape(fmt.Sprintf("type=aad&ver=1.0&sig=%s", tk.Token)))
		return nil
	}

	return fmt.Errorf("no authentication method available: provide a key, service-principal credentials, or workload identity")
}

// generateCosmosDBAuthToken generates an HMAC-SHA256 auth token for Cosmos DB REST API.
// Format: type=master&ver=1.0&sig={hashsignature}
// Signature input: {verb}\n{resourceType}\n{resourceLink}\n{date}\n\n
func generateCosmosDBAuthToken(verb, resourceType, resourceLink, date, key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("error decoding cosmos db key: %w", err)
	}

	text := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		strings.ToLower(resourceType),
		resourceLink,
		strings.ToLower(date))

	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(text))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", signature)), nil
}

func (c *cosmosDBClient) queryLeases(ctx context.Context) ([]leaseDocument, error) {
	var leases []leaseDocument
	continuation := ""
	for {
		page, next, err := c.queryLeasesPage(ctx, continuation)
		if err != nil {
			return nil, err
		}
		leases = append(leases, page...)
		if next == "" {
			return leases, nil
		}
		continuation = next
	}
}

func (c *cosmosDBClient) queryLeasesPage(ctx context.Context, continuation string) ([]leaseDocument, string, error) {
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", c.leaseDatabaseID, c.leaseContainerID)
	reqURL := fmt.Sprintf("%s/%s/docs", strings.TrimRight(c.leaseEndpoint, "/"), resourceLink)

	// The .NET and Java Change Feed Processor SDKs build each lease document's id as
	// {processorName}{monitoredAccountHost}_{rid}..{partitionId} - the processor name is
	// directly concatenated with the monitored (data) account's hostname, with no separator
	// (see the .NET SDK's CosmosContainerExtensions.GetLeasePrefix and the equivalent
	// getLeasePrefix in the Java SDK). Matching on processorName alone would let a processor
	// named "app" also match leases from "app-extended"; appending the monitored account's
	// short name (e.g. "myaccount" from "myaccount.documents.azure.com") closes that gap.
	// Since every Cosmos DB account hostname has the form "{account}.<domain-suffix>", the
	// "." immediately following the short name in the real id lets us match on
	// processorName + accountShortName + "." without needing to know the domain suffix for
	// the current cloud.
	prefix := c.processorName + cosmosDBAccountShortName(c.dataEndpoint) + "."
	prefixJSON, err := json.Marshal(prefix)
	if err != nil {
		return nil, "", fmt.Errorf("error marshaling processor name prefix: %w", err)
	}
	body := fmt.Sprintf(`{"query":"SELECT * FROM c WHERE STARTSWITH(c.id, @prefix)","parameters":[{"name":"@prefix","value":%s}]}`, string(prefixJSON))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("error creating request: %w", err)
	}

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", cosmosDBRestAPIVersion)
	req.Header.Set("Content-Type", "application/query+json")
	req.Header.Set("x-ms-documentdb-isquery", "true")
	req.Header.Set("x-ms-documentdb-query-enablecrosspartition", "true")
	if continuation != "" {
		req.Header.Set("x-ms-continuation", continuation)
	}

	if err := c.setAuthHeader(req, http.MethodPost, "docs", resourceLink, now, c.leaseKey); err != nil {
		return nil, "", fmt.Errorf("error setting auth header: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Documents []json.RawMessage `json:"Documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("error decoding response: %w", err)
	}

	// Parse and filter out non-lease metadata documents (the SDKs also create ".info"/".lock"
	// bookkeeping documents in the lease container, which never have a LeaseToken field).
	// A lease's ContinuationToken is intentionally NOT required here: a freshly created lease
	// that has never checkpointed yet (CreateLeaseIfNotExistAsync's initial state) legitimately
	// has an empty ContinuationToken - readChangeFeed/estimatePartitionLag already handle that
	// correctly by reading from the beginning of the feed, which is exactly what's needed to
	// detect a real customer's very first backlog. Requiring ContinuationToken here would
	// permanently hide that backlog, since nothing else ever populates it while the deployment
	// stays at 0 replicas.
	var leases []leaseDocument
	for _, raw := range result.Documents {
		var doc leaseDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, "", fmt.Errorf("error decoding lease document: %w", err)
		}
		if doc.LeaseToken != "" {
			if err := doc.parseFeedRange(); err != nil {
				return nil, "", err
			}
			leases = append(leases, doc)
		}
	}

	return leases, resp.Header.Get("x-ms-continuation"), nil
}

func (c *cosmosDBClient) readPartitionKeyRanges(ctx context.Context) ([]cosmosDBPartitionKeyRange, error) {
	var ranges []cosmosDBPartitionKeyRange
	continuation := ""
	for {
		page, next, err := c.readPartitionKeyRangesPage(ctx, continuation)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, page...)
		if next == "" {
			return ranges, nil
		}
		continuation = next
	}
}

func (c *cosmosDBClient) readPartitionKeyRangesPage(ctx context.Context, continuation string) ([]cosmosDBPartitionKeyRange, string, error) {
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", c.databaseID, c.containerID)
	reqURL := fmt.Sprintf("%s/%s/pkranges", strings.TrimRight(c.dataEndpoint, "/"), resourceLink)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error creating partition key ranges request: %w", err)
	}
	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", cosmosDBRestAPIVersion)
	if continuation != "" {
		req.Header.Set("x-ms-continuation", continuation)
	}
	if err := c.setAuthHeader(req, http.MethodGet, "pkranges", resourceLink, now, c.dataKey); err != nil {
		return nil, "", fmt.Errorf("error setting partition key ranges auth header: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("error reading partition key ranges: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("read partition key ranges failed with status %d", resp.StatusCode)
	}
	var result struct {
		Ranges []struct {
			ID           string  `json:"id"`
			MinInclusive *string `json:"minInclusive"`
			MaxExclusive *string `json:"maxExclusive"`
		} `json:"PartitionKeyRanges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("error decoding partition key ranges: %w", err)
	}
	ranges := make([]cosmosDBPartitionKeyRange, 0, len(result.Ranges))
	for _, physicalRange := range result.Ranges {
		if physicalRange.ID == "" {
			return nil, "", fmt.Errorf("partition key range is missing its ID")
		}
		epkRange, err := newCosmosDBEPKRange(physicalRange.MinInclusive, physicalRange.MaxExclusive)
		if err != nil {
			return nil, "", fmt.Errorf("invalid partition key range %q: %w", physicalRange.ID, err)
		}
		ranges = append(ranges, cosmosDBPartitionKeyRange{ID: physicalRange.ID, Range: *epkRange})
	}
	return ranges, resp.Header.Get("x-ms-continuation"), nil
}

func resolveCosmosDBEPKRange(epkRange cosmosDBEPKRange, ranges []cosmosDBPartitionKeyRange) (*cosmosDBPartitionKeyRange, bool, error) {
	var match *cosmosDBPartitionKeyRange
	for i := range ranges {
		physicalRange := &ranges[i]
		if epkRange.Min < physicalRange.Range.Max && epkRange.Max > physicalRange.Range.Min {
			if match != nil {
				// Match the SDK's synthetic 410/1002 when a lease spans split children.
				return nil, true, nil
			}
			match = physicalRange
		}
	}
	if match == nil || epkRange.Min < match.Range.Min || epkRange.Max > match.Range.Max {
		return nil, false, fmt.Errorf("no physical partition contains EPK range [%s, %s)", epkRange.Min, epkRange.Max)
	}
	return match, false, nil
}

func decodeCosmosDBContinuation(token string, epkRange cosmosDBEPKRange) (string, error) {
	// .NET persists the server ETag directly; Java persists Base64 ChangeFeedStateV1.
	if token == "" || strings.HasPrefix(token, `"`) {
		return token, nil
	}
	var data []byte
	var err error
	for _, encoding := range []*base64.Encoding{base64.URLEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.RawStdEncoding} {
		data, err = encoding.DecodeString(token)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", fmt.Errorf("invalid Java change feed continuation encoding: %w", err)
	}
	var state struct {
		Version      int    `json:"V"`
		Mode         string `json:"Mode"`
		Continuation struct {
			Version int `json:"V"`
			Tokens  []struct {
				Token *string         `json:"token"`
				Range json.RawMessage `json:"range"`
			} `json:"Continuation"`
		} `json:"Continuation"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("invalid Java change feed continuation state: %w", err)
	}
	if state.Version != 1 || state.Mode != "INCREMENTAL" || state.Continuation.Version != 1 || len(state.Continuation.Tokens) != 1 {
		return "", fmt.Errorf("java ChangeFeedStateV1 requires INCREMENTAL mode and one continuation range")
	}
	current := state.Continuation.Tokens[0]
	tokenRange, err := parseCosmosDBEPKRange(current.Range)
	if err != nil {
		return "", fmt.Errorf("invalid Java continuation range: %w", err)
	}
	if tokenRange.Min > epkRange.Min || tokenRange.Max < epkRange.Max || current.Token == nil || *current.Token == "" {
		return "", fmt.Errorf("java continuation must contain a server ETag covering the lease FeedRange")
	}
	return *current.Token, nil
}

func (c *cosmosDBClient) readLeaseChangeFeed(ctx context.Context, lease leaseDocument, ranges []cosmosDBPartitionKeyRange) (*changeFeedResponse, error) {
	if lease.Version == 0 {
		return c.readChangeFeed(ctx, lease.LeaseToken, lease.ContinuationToken, nil)
	}
	physicalRange, split, err := resolveCosmosDBEPKRange(*lease.epkRange, ranges)
	if err != nil {
		return nil, err
	}
	if split {
		return &changeFeedResponse{StatusCode: http.StatusGone, SubStatusCode: cosmosDBPartitionKeyRangeGoneSubStatus}, nil
	}
	continuation, err := decodeCosmosDBContinuation(lease.ContinuationToken, *lease.epkRange)
	if err != nil {
		return nil, err
	}
	var filter *cosmosDBEPKRange
	if physicalRange.Range != *lease.epkRange {
		filter = lease.epkRange
	}
	return c.readChangeFeed(ctx, physicalRange.ID, continuation, filter)
}

func (c *cosmosDBClient) readChangeFeed(ctx context.Context, partitionKeyRangeID, continuationToken string, epkRange *cosmosDBEPKRange) (*changeFeedResponse, error) {
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", c.databaseID, c.containerID)
	reqURL := fmt.Sprintf("%s/%s/docs", strings.TrimRight(c.dataEndpoint, "/"), resourceLink)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", cosmosDBRestAPIVersion)
	req.Header.Set("x-ms-documentdb-partitionkeyrangeid", partitionKeyRangeID)
	req.Header.Set("A-IM", "Incremental feed")
	req.Header.Set("x-ms-max-item-count", "1")
	if epkRange != nil {
		req.Header.Set("x-ms-read-key-type", "EffectivePartitionKeyRange")
		req.Header.Set("x-ms-start-epk", epkRange.Min)
		req.Header.Set("x-ms-end-epk", epkRange.Max)
	}

	if continuationToken != "" {
		req.Header.Set("If-None-Match", continuationToken)
	}

	if err := c.setAuthHeader(req, http.MethodGet, "docs", resourceLink, now, c.dataKey); err != nil {
		return nil, fmt.Errorf("error setting auth header: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	cfResp := &changeFeedResponse{
		StatusCode:    resp.StatusCode,
		SubStatusCode: resp.Header.Get(cosmosDBSubStatusHeader),
		SessionToken:  resp.Header.Get("x-ms-session-token"),
	}

	if resp.StatusCode == http.StatusNotModified || resp.StatusCode == http.StatusGone {
		return cfResp, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("read change feed failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Documents []json.RawMessage `json:"Documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	cfResp.Items = result.Documents
	return cfResp, nil
}

type cosmosDBLeaseState struct {
	totalLag               int64
	activeLeases           int64
	totalLeases            int64
	legacyActivePartitions int64
	splitRecoveryRequired  bool
}

// estimateLag refreshes leases and routing once if a partition split is detected.
func (c *cosmosDBClient) estimateLag(ctx context.Context) (cosmosDBLeaseState, error) {
	state, err := c.estimateOnce(ctx)
	if err != nil {
		return cosmosDBLeaseState{}, err
	}
	if state.splitRecoveryRequired {
		c.logger.Info("Warning: partition split detected, re-reading leases")
		return c.estimateOnce(ctx)
	}
	return state, nil
}

func (c *cosmosDBClient) estimateOnce(ctx context.Context) (cosmosDBLeaseState, error) {
	leases, err := c.queryLeases(ctx)
	if err != nil {
		return cosmosDBLeaseState{}, fmt.Errorf("error querying leases: %w", err)
	}

	if len(leases) == 0 {
		c.logger.V(1).Info("no lease documents found in lease container")
		return cosmosDBLeaseState{}, nil
	}

	c.logger.V(1).Info(fmt.Sprintf("found %d lease documents", len(leases)))

	state := cosmosDBLeaseState{totalLeases: int64(len(leases))}
	neverCheckpointedActiveCount := int64(0)

	var ranges []cosmosDBPartitionKeyRange
	for _, lease := range leases {
		if lease.Version == 1 && ranges == nil {
			ranges, err = c.readPartitionKeyRanges(ctx)
			if err != nil {
				return cosmosDBLeaseState{}, fmt.Errorf("error resolving lease FeedRange: %w", err)
			}
		}
		lag, isSplit, err := c.estimatePartitionLag(ctx, lease, ranges)
		if err != nil {
			return cosmosDBLeaseState{}, fmt.Errorf("error estimating lag: %w", err)
		}
		if isSplit {
			c.logger.Info(fmt.Sprintf("Warning: partition %s returned 410 Gone (split/merge detected)", lease.LeaseToken))
			state.splitRecoveryRequired = true
			continue
		}
		c.logger.V(1).Info(fmt.Sprintf("partition %s: estimated lag = %d, owner = %s", lease.LeaseToken, lag, lease.Owner))
		if lag > 0 {
			if lag > math.MaxInt64-state.totalLag {
				state.totalLag = math.MaxInt64
			} else {
				state.totalLag += lag
			}
			state.activeLeases++
			if lease.ContinuationToken == "" {
				neverCheckpointedActiveCount++
			}
		}
	}

	// Preserve the legacy bootstrap cap without hiding real lease counts from capacity scaling.
	state.legacyActivePartitions = state.activeLeases
	if neverCheckpointedActiveCount > 1 {
		state.legacyActivePartitions -= neverCheckpointedActiveCount - 1
	}

	return state, nil
}

// estimatePartitionLag calculates the lag for a single partition.
// Algorithm (matching .NET/Java SDKs):
//  1. Read change feed with maxItemCount=1 starting from the lease's continuation token
//  2. Extract latest LSN from session token
//  3. If items present: lag = sessionLSN - firstItem._lsn + 1
//  4. If no items (304): lag = 0 (caught up)
//  5. If 410/1002: flag a stale parent lease so the lease store can be refreshed
func (c *cosmosDBClient) estimatePartitionLag(ctx context.Context, lease leaseDocument, ranges []cosmosDBPartitionKeyRange) (int64, bool, error) {
	cfResp, err := c.readLeaseChangeFeed(ctx, lease, ranges)
	if err != nil {
		return 0, false, fmt.Errorf("error reading change feed for partition %s: %w", lease.LeaseToken, err)
	}

	if cfResp.StatusCode == http.StatusGone {
		if cfResp.SubStatusCode == cosmosDBPartitionKeyRangeGoneSubStatus {
			return 0, true, nil
		}
		return 0, false, fmt.Errorf("partition %s: read change feed failed with status %d and substatus %q", lease.LeaseToken, cfResp.StatusCode, cfResp.SubStatusCode)
	}

	// 304 Not Modified or empty results means processor is caught up
	if cfResp.StatusCode == http.StatusNotModified || len(cfResp.Items) == 0 {
		return 0, false, nil
	}

	// Calculate lag: sessionLSN - firstItemLSN + 1
	sessionLSN, err := parseLSNFromSessionToken(cfResp.SessionToken)
	if err != nil {
		return 0, false, fmt.Errorf("partition %s: could not parse session token LSN %q: %w", lease.LeaseToken, cfResp.SessionToken, err)
	}

	firstItemLSN, err := extractItemLSN(cfResp.Items[0])
	if err != nil {
		return 0, false, fmt.Errorf("partition %s: could not extract _lsn from first item: %w", lease.LeaseToken, err)
	}

	lag := sessionLSN - firstItemLSN + 1
	if lag < 0 {
		return 0, false, fmt.Errorf("partition %s: negative lag from session LSN %d and first item LSN %d", lease.LeaseToken, sessionLSN, firstItemLSN)
	}

	return lag, false, nil
}

// extractLSNFromSessionToken extracts the LSN from a Cosmos DB session token.
// Session token formats:
//   - Simple: "{pkRangeId}:{lsn}"
//   - Compound: "{pkRangeId}:{localLsn}#{globalLsn}"
//
// This matches the logic in both the .NET SDK (ChangeFeedEstimatorIterator.ExtractLsnFromSessionToken)
// and Java SDK (IncrementalChangeFeedProcessorImpl).
func extractLSNFromSessionToken(sessionToken string) string {
	if sessionToken == "" {
		return ""
	}

	colonIdx := strings.IndexByte(sessionToken, ':')
	if colonIdx < 0 {
		return sessionToken
	}
	parsed := sessionToken[colonIdx+1:]

	segments := strings.Split(parsed, "#")
	if len(segments) >= 2 {
		return segments[1] // Global LSN
	}
	return segments[0]
}

func parseLSNFromSessionToken(sessionToken string) (int64, error) {
	lsnStr := extractLSNFromSessionToken(sessionToken)
	if lsnStr == "" {
		return -1, fmt.Errorf("empty session token")
	}
	return strconv.ParseInt(lsnStr, 10, 64)
}

// extractItemLSN extracts the _lsn value from a Cosmos DB change feed document.
func extractItemLSN(item json.RawMessage) (int64, error) {
	var doc struct {
		LSN json.Number `json:"_lsn"`
	}
	if err := json.Unmarshal(item, &doc); err != nil {
		return -1, fmt.Errorf("parsing item: %w", err)
	}
	return doc.LSN.Int64()
}

// GetMetricSpecForScaling returns the metric spec for scaling.
func (s *azureCosmosDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("azure-cosmosdb-%s-%s",
		s.metadata.LeaseContainerID, s.metadata.ProcessorName))
	target := s.metadata.Threshold
	if s.metadata.MaxActiveLeasesPerReplica > 0 {
		target = 1
	}
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, target),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cosmosDBMetricType}
	return []v2.MetricSpec{metricSpec}
}

// getChangeFeedTotalLagRelatedToPartitionAmount caps the total lag to prevent scaling beyond
// the number of partitions that have lag.
func getChangeFeedTotalLagRelatedToPartitionAmount(totalLag int64, activePartitionCount int64, threshold int64) int64 {
	if threshold <= 0 || activePartitionCount <= 0 || activePartitionCount > math.MaxInt64/threshold {
		return totalLag
	}

	maxLag := activePartitionCount * threshold
	if totalLag > maxLag {
		return maxLag
	}
	return totalLag
}

// GetMetricsAndActivity returns the metric value and activity status.
func (s *azureCosmosDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	state, err := s.cosmosClient.estimateLag(ctx)
	if err != nil {
		s.recordLeaseDiagnostics(metricName, cosmosDBLeaseState{}, true)
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting cosmos db change feed lag: %w", err)
	}
	s.recordLeaseDiagnostics(metricName, state, false)
	if s.metadata.MaxActiveLeasesPerReplica > 0 {
		lagReplicas, capacityReplicas, eligibleLeases := s.leaseReplicaCounts(state)
		desired := min(max(lagReplicas, capacityReplicas), eligibleLeases)
		active := state.totalLag > s.metadata.ActivationThreshold || (s.metadata.EnableHighAvailability && state.totalLeases > 0)
		if state.splitRecoveryRequired {
			desired = max(desired, 1)
			active = true
		}
		return []external_metrics.ExternalMetricValue{GenerateMetricInMili(metricName, float64(desired))}, active, nil
	}

	lagForScaling := state.totalLag
	effectiveActivePartitionCount := state.legacyActivePartitions
	if state.splitRecoveryRequired {
		if lagForScaling <= s.metadata.ActivationThreshold {
			lagForScaling = s.metadata.ActivationThreshold + 1
		}
		if effectiveActivePartitionCount == 0 {
			effectiveActivePartitionCount = 1
		}
	}

	// Don't scale out beyond the number of partitions that have lag.
	lagRelatedToPartitionCount := getChangeFeedTotalLagRelatedToPartitionAmount(lagForScaling, effectiveActivePartitionCount, s.metadata.Threshold)

	s.logger.V(1).Info(fmt.Sprintf("Cosmos DB change feed total lag: %d, scaling for a lag of %d related to %d active partitions, split recovery required: %t",
		state.totalLag, lagRelatedToPartitionCount, effectiveActivePartitionCount, state.splitRecoveryRequired))

	metric := GenerateMetricInMili(metricName, float64(lagRelatedToPartitionCount))
	return []external_metrics.ExternalMetricValue{metric}, lagForScaling > s.metadata.ActivationThreshold, nil
}

func cosmosDBCeilDivide(value, divisor int64) int64 {
	result := value / divisor
	if value%divisor != 0 {
		result++
	}
	return result
}

func (s *azureCosmosDBScaler) leaseReplicaCounts(state cosmosDBLeaseState) (int64, int64, int64) {
	eligibleLeases := state.activeLeases
	if s.metadata.EnableHighAvailability {
		eligibleLeases = state.totalLeases
	}
	capacityReplicas := int64(0)
	if s.metadata.MaxActiveLeasesPerReplica > 0 {
		capacityReplicas = cosmosDBCeilDivide(eligibleLeases, s.metadata.MaxActiveLeasesPerReplica)
	}
	return cosmosDBCeilDivide(state.totalLag, s.metadata.Threshold), capacityReplicas, eligibleLeases
}

func (s *azureCosmosDBScaler) recordLeaseDiagnostics(metricName string, state cosmosDBLeaseState, failed bool) {
	s.diagnosticsMutex.Lock()
	defer s.diagnosticsMutex.Unlock()
	if s.diagnosticsClosed {
		return
	}
	if s.scalerConfig == nil {
		return
	}
	s.diagnosticMetricName = metricName
	for suffix, value := range s.leaseDiagnosticValues(state) {
		measurement := float64(value)
		if failed {
			measurement = math.NaN()
		}
		metricscollector.RecordScalerMetric(s.scalerConfig.ScalableObjectNamespace, s.scalerConfig.ScalableObjectName,
			"azure-cosmosdb", s.metadata.TriggerIndex, metricName+"_"+suffix, s.scalerConfig.ScalableObjectType == "ScaledObject", measurement)
	}
}

func (s *azureCosmosDBScaler) leaseDiagnosticValues(state cosmosDBLeaseState) map[string]int64 {
	lagReplicas, capacityReplicas, _ := s.leaseReplicaCounts(state)
	return map[string]int64{
		"total_lag":                       state.totalLag,
		"active_leases":                   state.activeLeases,
		"total_leases":                    state.totalLeases,
		"lag_desired_replicas":            lagReplicas,
		"lease_capacity_desired_replicas": capacityReplicas,
	}
}

// Close cleans up the scaler resources.
func (s *azureCosmosDBScaler) Close(context.Context) error {
	s.diagnosticsMutex.Lock()
	defer s.diagnosticsMutex.Unlock()
	if !s.diagnosticsClosed && s.scalerConfig != nil && s.diagnosticMetricName != "" {
		for suffix := range s.leaseDiagnosticValues(cosmosDBLeaseState{}) {
			metricscollector.DeleteScalerMetric(s.scalerConfig.ScalableObjectNamespace, s.scalerConfig.ScalableObjectName,
				"azure-cosmosdb", s.metadata.TriggerIndex, s.diagnosticMetricName+"_"+suffix, s.scalerConfig.ScalableObjectType == "ScaledObject")
		}
	}
	s.diagnosticsClosed = true
	if s.cosmosClient != nil && s.cosmosClient.httpClient != nil {
		s.cosmosClient.httpClient.CloseIdleConnections()
	}
	return nil
}
