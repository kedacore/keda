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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	cosmosDBMetricType     = "External"
	cosmosDBRestAPIVersion = "2018-12-31"
)

type azureCosmosDBScaler struct {
	metricType   v2.MetricTargetType
	metadata     *azureCosmosDBMetadata
	cosmosClient *cosmosDBClient
	logger       logr.Logger
}

type azureCosmosDBMetadata struct {
	DatabaseID          string `keda:"name=databaseId,              order=triggerMetadata"`
	ContainerID         string `keda:"name=containerId,             order=triggerMetadata"`
	LeaseDatabaseID     string `keda:"name=leaseDatabaseId,         order=triggerMetadata"`
	LeaseContainerID    string `keda:"name=leaseContainerId,        order=triggerMetadata"`
	ProcessorName       string `keda:"name=processorName,           order=triggerMetadata"`
	Endpoint            string `keda:"name=endpoint,                order=authParams;triggerMetadata, optional"`
	Connection          string `keda:"name=connection,              order=authParams;resolvedEnv;triggerMetadata, optional"`
	LeaseEndpoint       string `keda:"name=leaseEndpoint,           order=authParams;triggerMetadata, optional"`
	LeaseConnection     string `keda:"name=leaseConnection,         order=authParams;resolvedEnv;triggerMetadata, optional"`
	CosmosDBKey         string `keda:"name=cosmosDBKey,             order=authParams;resolvedEnv, optional"`
	LeaseCosmosDBKey    string `keda:"name=leaseCosmosDBKey,        order=authParams;resolvedEnv, optional"`
	Threshold           int64  `keda:"name=changeFeedLagThreshold,            order=triggerMetadata, default=100"`
	ActivationThreshold int64  `keda:"name=activationChangeFeedLagThreshold,  order=triggerMetadata, default=0"`
	TriggerIndex        int
}

// cosmosDBClient provides low-level access to Cosmos DB via the REST API
// for querying lease documents and reading the change feed.
type cosmosDBClient struct {
	httpClient       *http.Client
	dataEndpoint     string
	dataKey          string
	leaseEndpoint    string
	leaseKey         string
	leaseDatabaseID  string
	leaseContainerID string
	databaseID       string
	containerID      string
	credential       azcore.TokenCredential
}

type leaseDocument struct {
	ID                string `json:"id"`
	LeaseToken        string `json:"LeaseToken"`
	ContinuationToken string `json:"ContinuationToken"`
	Owner             string `json:"Owner,omitempty"`
}

type changeFeedResponse struct {
	StatusCode   int
	Items        []json.RawMessage
	SessionToken string
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

	cosmosClient, err := newCosmosDBClient(meta, config.PodIdentity, logger, config.GlobalHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("error creating cosmos db client: %w", err)
	}

	return &azureCosmosDBScaler{
		metricType:   metricType,
		metadata:     meta,
		cosmosClient: cosmosClient,
		logger:       logger,
	}, nil
}

func parseAzureCosmosDBMetadata(config *scalersconfig.ScalerConfig) (*azureCosmosDBMetadata, error) {
	meta := &azureCosmosDBMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if meta.Connection == "" && (meta.Endpoint == "" || meta.CosmosDBKey == "") {
			return nil, fmt.Errorf("connection string or endpoint+cosmosDBKey is required when not using pod identity")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		if meta.Endpoint == "" && meta.Connection == "" {
			return nil, fmt.Errorf("endpoint or connection string is required when using workload identity")
		}
	default:
		return nil, fmt.Errorf("pod identity %s not supported for azure cosmos db", config.PodIdentity.Provider)
	}

	// Default lease settings to data settings if not specified
	if meta.LeaseConnection == "" {
		meta.LeaseConnection = meta.Connection
	}
	if meta.LeaseEndpoint == "" {
		meta.LeaseEndpoint = meta.Endpoint
	}
	if meta.LeaseCosmosDBKey == "" {
		meta.LeaseCosmosDBKey = meta.CosmosDBKey
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func newCosmosDBClient(meta *azureCosmosDBMetadata, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger, httpTimeout time.Duration) (*cosmosDBClient, error) {
	if httpTimeout == 0 {
		httpTimeout = 30 * time.Second
	}

	client := &cosmosDBClient{
		httpClient:       kedautil.CreateHTTPClient(httpTimeout, false),
		leaseDatabaseID:  meta.LeaseDatabaseID,
		leaseContainerID: meta.LeaseContainerID,
		databaseID:       meta.DatabaseID,
		containerID:      meta.ContainerID,
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

	// Set up workload identity credential for bearer token auth
	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload && client.dataKey == "" {
		cred, err := azure.NewChainedCredential(logger, podIdentity)
		if err != nil {
			return nil, fmt.Errorf("error creating azure credential for workload identity: %w", err)
		}
		client.credential = cred
	}

	return client, nil
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

// setAuthHeader sets the Authorization header using either master key HMAC-SHA256 or bearer token.
func (c *cosmosDBClient) setAuthHeader(req *http.Request, verb, resourceType, resourceLink, date, key string) error {
	if key != "" {
		token := generateCosmosDBAuthToken(verb, resourceType, resourceLink, date, key)
		req.Header.Set("Authorization", token)
		return nil
	}

	if c.credential != nil {
		tk, err := c.credential.GetToken(req.Context(), policy.TokenRequestOptions{
			Scopes: []string{azure.PublicCloud.ResourceIdentifiers.CosmosDB + "/.default"},
		})
		if err != nil {
			return fmt.Errorf("error acquiring bearer token: %w", err)
		}
		req.Header.Set("Authorization", "type=aad&ver=1.0&sig="+tk.Token)
		return nil
	}

	return fmt.Errorf("no authentication method available: provide a key or configure workload identity")
}

// generateCosmosDBAuthToken generates an HMAC-SHA256 auth token for Cosmos DB REST API.
// Format: type=master&ver=1.0&sig={hashsignature}
// Signature input: {verb}\n{resourceType}\n{resourceLink}\n{date}\n\n
func generateCosmosDBAuthToken(verb, resourceType, resourceLink, date, key string) string {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return ""
	}

	text := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		strings.ToLower(resourceType),
		resourceLink,
		strings.ToLower(date))

	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(text))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", signature))
}

func (c *cosmosDBClient) queryLeases(ctx context.Context) ([]leaseDocument, error) {
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", c.leaseDatabaseID, c.leaseContainerID)
	reqURL := fmt.Sprintf("%s/%s/docs", strings.TrimRight(c.leaseEndpoint, "/"), resourceLink)

	body := `{"query":"SELECT * FROM c"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", cosmosDBRestAPIVersion)
	req.Header.Set("Content-Type", "application/query+json")
	req.Header.Set("x-ms-documentdb-isquery", "true")
	req.Header.Set("x-ms-documentdb-query-enablecrosspartition", "true")

	if err := c.setAuthHeader(req, http.MethodPost, "docs", resourceLink, now, c.leaseKey); err != nil {
		return nil, fmt.Errorf("error setting auth header: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Documents []json.RawMessage `json:"Documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Parse and filter out metadata documents (those without LeaseToken or ContinuationToken)
	var leases []leaseDocument
	for _, raw := range result.Documents {
		var doc leaseDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		if doc.LeaseToken != "" && doc.ContinuationToken != "" {
			leases = append(leases, doc)
		}
	}

	return leases, nil
}

func (c *cosmosDBClient) readChangeFeed(ctx context.Context, partitionKeyRangeID, continuationToken string) (*changeFeedResponse, error) {
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
		StatusCode:   resp.StatusCode,
		SessionToken: resp.Header.Get("x-ms-session-token"),
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

// estimateLag estimates the total change feed lag across all partitions and
// returns both the total lag and partition count.
// If a partition split (410 Gone) is detected, it retries once to get fresh lease data.
func (c *cosmosDBClient) estimateLag(ctx context.Context) (totalLag int64, partitionCount int64, err error) {
	totalLag, partitionCount, splitDetected, err := c.estimateOnce(ctx)
	if err != nil {
		return 0, 0, err
	}
	if splitDetected {
		totalLag, partitionCount, _, err = c.estimateOnce(ctx)
		if err != nil {
			return 0, 0, err
		}
	}
	return totalLag, partitionCount, nil
}

func (c *cosmosDBClient) estimateOnce(ctx context.Context) (int64, int64, bool, error) {
	leases, err := c.queryLeases(ctx)
	if err != nil {
		return 0, 0, false, fmt.Errorf("error querying leases: %w", err)
	}

	if len(leases) == 0 {
		return 0, 0, false, nil
	}

	totalLag := int64(0)
	splitDetected := false

	for _, lease := range leases {
		lag, isSplit, err := c.estimatePartitionLag(ctx, lease)
		if err != nil {
			return 0, 0, false, fmt.Errorf("error estimating lag for partition %s: %w", lease.LeaseToken, err)
		}
		if isSplit {
			splitDetected = true
			continue
		}
		if lag > 0 {
			totalLag += lag
		}
	}

	// Cap to prevent int64 overflow from summing across many partitions
	if totalLag < 0 {
		totalLag = math.MaxInt64
	}

	return totalLag, int64(len(leases)), splitDetected, nil
}

// estimatePartitionLag calculates the lag for a single partition.
// Algorithm (matching .NET/Java SDKs):
//  1. Read change feed with maxItemCount=1 starting from the lease's continuation token
//  2. Extract latest LSN from session token
//  3. If items present: lag = sessionLSN - firstItem._lsn + 1
//  4. If no items (304): lag = 0 (caught up)
//  5. If 410 Gone: report lag = -1 (split/merge)
func (c *cosmosDBClient) estimatePartitionLag(ctx context.Context, lease leaseDocument) (int64, bool, error) {
	cfResp, err := c.readChangeFeed(ctx, lease.LeaseToken, lease.ContinuationToken)
	if err != nil {
		return 0, false, err
	}

	// 410 Gone indicates partition split or merge
	if cfResp.StatusCode == http.StatusGone {
		return -1, true, nil
	}

	// 304 Not Modified or empty results means processor is caught up
	if cfResp.StatusCode == http.StatusNotModified || len(cfResp.Items) == 0 {
		return 0, false, nil
	}

	// Calculate lag: sessionLSN - firstItemLSN + 1
	sessionLSN, err := parseLSNFromSessionToken(cfResp.SessionToken)
	if err != nil || sessionLSN < 0 {
		return 0, false, nil
	}

	firstItemLSN, err := extractItemLSN(cfResp.Items[0])
	if err != nil || firstItemLSN < 0 {
		return 0, false, nil
	}

	lag := sessionLSN - firstItemLSN + 1
	if lag < 0 {
		return 0, false, nil
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
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cosmosDBMetricType}
	return []v2.MetricSpec{metricSpec}
}

// getChangeFeedTotalLagRelatedToPartitionAmount caps the total lag to prevent scaling beyond
// the number of partitions. This matches the EventHub scaler's approach.
func getChangeFeedTotalLagRelatedToPartitionAmount(totalLag int64, partitionCount int64, threshold int64) int64 {
	if threshold > 0 && (totalLag/threshold) > partitionCount {
		return partitionCount * threshold
	}
	return totalLag
}

// GetMetricsAndActivity returns the metric value and activity status.
func (s *azureCosmosDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalLag, partitionCount, err := s.cosmosClient.estimateLag(ctx)
	if err != nil {
		s.logger.Error(err, "error getting cosmos db change feed lag")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	// Don't scale out beyond the number of partitions
	lagRelatedToPartitionCount := getChangeFeedTotalLagRelatedToPartitionAmount(totalLag, partitionCount, s.metadata.Threshold)

	s.logger.V(1).Info(fmt.Sprintf("Cosmos DB change feed total lag: %d, scaling for a lag of %d related to %d partitions",
		totalLag, lagRelatedToPartitionCount, partitionCount))

	metric := GenerateMetricInMili(metricName, float64(lagRelatedToPartitionCount))
	return []external_metrics.ExternalMetricValue{metric}, totalLag > s.metadata.ActivationThreshold, nil
}

// Close cleans up the scaler resources.
func (s *azureCosmosDBScaler) Close(context.Context) error {
	if s.cosmosClient != nil && s.cosmosClient.httpClient != nil {
		s.cosmosClient.httpClient.CloseIdleConnections()
	}
	return nil
}
