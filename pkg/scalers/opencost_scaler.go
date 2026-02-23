package scalers

// OpenCost Scaler
//
// The OpenCost scaler enables cost-based autoscaling for Kubernetes workloads by querying
// the OpenCost API for real-time cost metrics. It supports two scaling modes:
//
// 1. Inverse Scaling (Default - inverseScaling=true):
//    Scale DOWN when costs are HIGH to reduce expenses.
//    This is the intuitive behavior for cost optimization: when your workload costs exceed
//    the threshold, KEDA reduces replicas to bring costs back under control.
//
//    Use case: Cost containment
//    - Set a maximum cost budget (costThreshold)
//    - When costs exceed the budget, scale down to reduce spending
//    - Example: Limit namespace costs to $100/hour
//
// 2. Normal Scaling (inverseScaling=false):
//    Scale UP when costs are HIGH.
//    This mode is useful for time-based cost optimization scenarios where high costs
//    indicate high demand periods.
//
//    Use case: Time-based cost optimization
//    - During peak hours (high demand), costs naturally increase due to more traffic
//    - Scale UP to handle the increased load efficiently
//    - During off-peak hours (low demand), costs decrease naturally
//    - Scale DOWN as there's less work to do
//    - Example: E-commerce site scales up during business hours when costs/traffic are high
//
// The scaler queries OpenCost's allocation API to get cost metrics aggregated by various
// dimensions (namespace, pod, controller, etc.) and can filter by specific resources.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type openCostScaler struct {
	metricType v2.MetricTargetType
	metadata   *openCostScalerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type openCostScalerMetadata struct {
	// OpenCost server URL (e.g., http://opencost.opencost.svc.cluster.local:9003)
	ServerAddress string `keda:"name=serverAddress,order=triggerMetadata"`
	// Window for cost query (e.g., "1h", "24h", "7d")
	Window string `keda:"name=window,order=triggerMetadata,default=1h"`
	// Aggregate by: cluster, node, namespace, controllerKind, controller, service, pod, container
	Aggregate string `keda:"name=aggregate,order=triggerMetadata,default=namespace"`
	// Filter to apply (e.g., namespace name, pod name)
	Filter string `keda:"name=filter,order=triggerMetadata,optional"`
	// Cost threshold in dollars - with inverse scaling (default), scale down when cost exceeds this
	CostThreshold float64 `keda:"name=costThreshold,order=triggerMetadata"`
	// Activation cost threshold - scaler becomes active when cost exceeds this
	ActivationCostThreshold float64 `keda:"name=activationCostThreshold,order=triggerMetadata,default=0"`
	// Cost type: totalCost, cpuCost, gpuCost, ramCost, pvCost, networkCost
	CostType string `keda:"name=costType,order=triggerMetadata,default=totalCost"`
	// Whether to use unsafe SSL
	UnsafeSsl bool `keda:"name=unsafeSsl,order=triggerMetadata,default=false"`
	// Inverse scaling: when true (default), scale DOWN when costs are high to reduce expenses.
	// When false, scale UP when costs are high (useful for time-based cost optimization scenarios).
	InverseScaling bool `keda:"name=inverseScaling,order=triggerMetadata,default=true"`

	triggerIndex   int
	asMetricSource bool
}

func (m *openCostScalerMetadata) Validate() error {
	// Validate cost threshold
	if m.CostThreshold <= 0 && !m.asMetricSource {
		return fmt.Errorf("costThreshold must be a positive number")
	}

	// Validate cost type
	validCostTypes := map[string]bool{
		"totalCost":   true,
		"cpuCost":     true,
		"gpuCost":     true,
		"ramCost":     true,
		"pvCost":      true,
		"networkCost": true,
	}
	if !validCostTypes[m.CostType] {
		return fmt.Errorf("invalid costType: %s. Valid options: totalCost, cpuCost, gpuCost, ramCost, pvCost, networkCost", m.CostType)
	}

	// Validate aggregate
	validAggregates := map[string]bool{
		"cluster":        true,
		"node":           true,
		"namespace":      true,
		"controllerKind": true,
		"controller":     true,
		"service":        true,
		"pod":            true,
		"container":      true,
	}
	if !validAggregates[m.Aggregate] {
		return fmt.Errorf("invalid aggregate: %s. Valid options: cluster, node, namespace, controllerKind, controller, service, pod, container", m.Aggregate)
	}

	return nil
}

// OpenCost API response structures
type openCostAllocationResponse struct {
	Code    int                       `json:"code"`
	Status  string                    `json:"status"`
	Data    []map[string]openCostItem `json:"data"`
	Message string                    `json:"message,omitempty"`
}

type openCostItem struct {
	Name        string         `json:"name"`
	Properties  openCostProps  `json:"properties"`
	Window      openCostWindow `json:"window"`
	Start       string         `json:"start"`
	End         string         `json:"end"`
	CPUCost     float64        `json:"cpuCost"`
	GPUCost     float64        `json:"gpuCost"`
	RAMCost     float64        `json:"ramCost"`
	PVCost      float64        `json:"pvCost"`
	NetworkCost float64        `json:"networkCost"`
	TotalCost   float64        `json:"totalCost"`
}

type openCostProps struct {
	Cluster    string `json:"cluster"`
	Node       string `json:"node"`
	Namespace  string `json:"namespace"`
	Controller string `json:"controller"`
	Pod        string `json:"pod"`
	Container  string `json:"container"`
}

type openCostWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// NewOpenCostScaler creates a new OpenCost scaler
func NewOpenCostScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseOpenCostMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing OpenCost metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	return &openCostScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "opencost_scaler"),
	}, nil
}

func parseOpenCostMetadata(config *scalersconfig.ScalerConfig) (*openCostScalerMetadata, error) {
	meta := &openCostScalerMetadata{
		asMetricSource: config.AsMetricSource,
	}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing OpenCost metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

// GetMetricsAndActivity returns the current cost metric and activity status
func (s *openCostScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	cost, err := s.getCost(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting cost from OpenCost: %w", err)
	}

	// When inverseScaling is true (default), invert the metric so HPA scales DOWN when costs are high.
	// This is the intuitive behavior for cost optimization: reduce replicas to reduce costs.
	metricValue := cost
	if s.metadata.InverseScaling {
		// Invert: report (threshold * 2 - cost) so when cost > threshold, metric < threshold
		metricValue = (s.metadata.CostThreshold * 2) - cost
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	isActive := cost > s.metadata.ActivationCostThreshold

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the HPA
func (s *openCostScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("opencost-%s-%s", s.metadata.Aggregate, s.metadata.CostType))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.CostThreshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     v2.ExternalMetricSourceType,
	}
	return []v2.MetricSpec{metricSpec}
}

// Close closes the HTTP client connections
func (s *openCostScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

// getCost queries the OpenCost API and returns the cost value
func (s *openCostScaler) getCost(ctx context.Context) (float64, error) {
	// Build the OpenCost allocation API URL
	apiURL, err := url.Parse(s.metadata.ServerAddress)
	if err != nil {
		return 0, fmt.Errorf("invalid server address: %w", err)
	}
	apiURL.Path = "/allocation"

	// Add query parameters
	query := apiURL.Query()
	query.Set("window", s.metadata.Window)
	query.Set("aggregate", s.metadata.Aggregate)
	if s.metadata.Filter != "" {
		query.Set("filter", s.metadata.Filter)
	}
	apiURL.RawQuery = query.Encode()

	s.logger.V(1).Info("Querying OpenCost API", "url", apiURL.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request to OpenCost: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("OpenCost API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var response openCostAllocationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("error parsing OpenCost response: %w", err)
	}

	if response.Code != 200 {
		return 0, fmt.Errorf("OpenCost API error: %s", response.Message)
	}

	// Calculate total cost from all items in the response
	totalCost := 0.0
	for _, dataSet := range response.Data {
		for _, item := range dataSet {
			cost := s.extractCost(item)
			totalCost += cost
		}
	}

	s.logger.V(1).Info("Got cost from OpenCost", "costType", s.metadata.CostType, "cost", totalCost)

	return totalCost, nil
}

// extractCost extracts the appropriate cost value based on costType
func (s *openCostScaler) extractCost(item openCostItem) float64 {
	switch s.metadata.CostType {
	case "cpuCost":
		return item.CPUCost
	case "gpuCost":
		return item.GPUCost
	case "ramCost":
		return item.RAMCost
	case "pvCost":
		return item.PVCost
	case "networkCost":
		return item.NetworkCost
	case "totalCost":
		fallthrough
	default:
		return item.TotalCost
	}
}

// Helper function to parse float from string
