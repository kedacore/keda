package scalers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// kubernetesNodesScaler scales workloads with the schedulable Ready node
// count: one replica per node for workloads that follow capacity (plain
// StatefulSets with parallel join/leave, operator CRs where the operator
// owns membership). Cordoned, draining and NotReady nodes never flap the
// metric by themselves, matching cluster-proportional-autoscaler.
type kubernetesNodesScaler struct {
	metricType v2.MetricTargetType
	metadata   kubernetesNodesMetadata
	kubeClient client.Client
	logger     logr.Logger
}

const (
	kubernetesNodesMetricType = "External"
)

type kubernetesNodesMetadata struct {
	NodeSelector    string  `keda:"name=nodeSelector, order=triggerMetadata, optional"`
	Value           float64 `keda:"name=value, order=triggerMetadata, default=1"`
	ActivationValue float64 `keda:"name=activationValue, order=triggerMetadata, default=0"`
	// Levels maps the node-following replica count onto quorum steps,
	// e.g. "1,3,5" emits 1,1,3,3,5,5 for 0..6 nodes (0 stays 0 for
	// scale-to-zero; below the first level floors to it so extra
	// replicas pend by design as node pressure). Empty keeps linear.
	// Mutually exclusive with OddOnly: use Levels for explicit caps,
	// OddOnly for unbounded odd-quorum scaling.
	Levels string `keda:"name=levels, order=triggerMetadata, optional"`
	// OddOnly rounds the replica count down to the nearest odd number
	// (0,1,1,3,3,5,5,7,7...), so unbounded scaling always keeps an odd
	// quorum. Empty levels required alongside it.
	OddOnly bool `keda:"name=oddOnly, order=triggerMetadata, default=false"`

	triggerIndex int
	nodeSelector labels.Selector
	levels       []int
}

func (m *kubernetesNodesMetadata) Validate() error {
	if err := validateNodesFloats(m.Value, m.ActivationValue); err != nil {
		return err
	}
	levels, err := parseNodesLevels(m.Levels)
	if err != nil {
		return err
	}
	if m.OddOnly && len(levels) > 0 {
		return fmt.Errorf("levels and oddOnly are mutually exclusive")
	}

	return nil
}

// validateNodesFloats rejects non-positive or non-finite values: NaN slips
// past plain comparisons (NaN <= 0 is false), int64(NaN) is undefined, and
// infinite values collapse the replica math and the activity check.
func validateNodesFloats(value, activationValue float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return fmt.Errorf("value must be a finite float greater than 0")
	}
	if math.IsNaN(activationValue) || math.IsInf(activationValue, 0) {
		return fmt.Errorf("activationValue must be a finite number")
	}
	return nil
}

// parseNodesLevels parses a comma-separated ascending list of positive
// replica steps, e.g. "1,3,5". Empty means linear (no stepping).
func parseNodesLevels(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	levels := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("levels must be comma-separated positive integers, got %q", s)
		}
		if len(levels) > 0 && n <= levels[len(levels)-1] {
			return nil, fmt.Errorf("levels must be strictly ascending, got %q", s)
		}
		levels = append(levels, n)
	}
	return levels, nil
}

// applyNodesLevels maps a linear replica count onto the stepped levels.
// Zero stays zero so scale-to-zero keeps working; a nonzero count below
// the first level floors to it (pend-by-design node pressure).
func applyNodesLevels(desired int64, levels []int) int64 {
	if len(levels) == 0 || desired == 0 {
		return desired
	}
	stepped := int64(levels[0])
	for _, l := range levels {
		if int64(l) <= desired {
			stepped = int64(l)
		} else {
			break
		}
	}
	return stepped
}

// applyNodesOddOnly rounds a linear replica count down to the nearest odd
// number (0,1,1,3,3,5,5,7,7...), so unbounded scaling always keeps an odd
// quorum. Zero stays zero so scale-to-zero keeps working.
func applyNodesOddOnly(desired int64) int64 {
	if desired <= 0 || desired%2 == 1 {
		return desired
	}
	return desired - 1
}

// NewKubernetesNodesScaler creates a new kubernetesNodesScaler
func NewKubernetesNodesScaler(kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseKubernetesNodesMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes nodes metadata: %w", err)
	}

	return &kubernetesNodesScaler{
		metricType: metricType,
		metadata:   meta,
		kubeClient: kubeClient,
		logger:     InitializeLogger(config, "kubernetes_nodes_scaler"),
	}, nil
}

func parseKubernetesNodesMetadata(config *scalersconfig.ScalerConfig) (kubernetesNodesMetadata, error) {
	meta := kubernetesNodesMetadata{}
	meta.triggerIndex = config.TriggerIndex

	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing kubernetes nodes metadata: %w", err)
	}

	selector, err := labels.Parse(meta.NodeSelector)
	if err != nil {
		return meta, fmt.Errorf("error parsing node selector: %w", err)
	}
	meta.nodeSelector = selector

	levels, err := parseNodesLevels(meta.Levels)
	if err != nil {
		return meta, err
	}
	meta.levels = levels

	if meta.OddOnly && len(levels) > 0 {
		return meta, fmt.Errorf("levels and oddOnly are mutually exclusive")
	}

	if err := validateNodesFloats(meta.Value, meta.ActivationValue); err != nil {
		return meta, err
	}

	return meta, nil
}

// Close closes the scaler (nothing to close for node listing).
func (s *kubernetesNodesScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *kubernetesNodesScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	name := "all"
	if s.metadata.NodeSelector != "" {
		name = s.metadata.NodeSelector
	}
	metricName := kedautil.NormalizeString(fmt.Sprintf("nodes-%s", name))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kubernetesNodesMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric
func (s *kubernetesNodesScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	count, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting kubernetes nodes: %w", err)
	}

	// HPA aims for ceil(metric/value) replicas, so the metric carries the
	// stepped replica count times value (levels apply after the value
	// division, keeping value=1 flows identical to before). The epsilon
	// absorbs binary float error that would otherwise push exact quotients
	// (e.g. 21/0.7) one step above their integer and over-scale by one;
	// it sits six orders below HPA milli precision, so genuine fractions
	// still round up correctly.
	raw := int64(math.Ceil(float64(count)/s.metadata.Value - 1e-9))
	stepped := applyNodesLevels(raw, s.metadata.levels)
	if s.metadata.OddOnly {
		stepped = applyNodesOddOnly(raw)
	}
	metric := GenerateMetricInMili(metricName, float64(stepped)*s.metadata.Value)

	return []external_metrics.ExternalMetricValue{metric}, float64(stepped) > s.metadata.ActivationValue, nil
}

func (s *kubernetesNodesScaler) getMetricValue(ctx context.Context) (int64, error) {
	nodeList := &corev1.NodeList{}
	err := s.kubeClient.List(ctx, nodeList, &client.ListOptions{
		LabelSelector: s.metadata.nodeSelector,
	})
	if err != nil {
		return 0, err
	}

	var count int64
	for i := range nodeList.Items {
		if isNodeSchedulableReady(&nodeList.Items[i]) {
			count++
		}
	}

	return count, nil
}

// isNodeSchedulableReady counts only schedulable Ready=True nodes:
// cordoned/draining (spec.unschedulable) and NotReady nodes are invisible
// to the metric so capacity churn never flaps the replica count by itself.
func isNodeSchedulableReady(node *corev1.Node) bool {
	if node.Spec.Unschedulable {
		return false
	}
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
