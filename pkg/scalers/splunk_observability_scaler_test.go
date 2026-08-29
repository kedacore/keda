package scalers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalfx-go/idtool"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseSplunkObservabilityMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type SplunkObservabilityMetricIdentifier struct {
	metadataTestData *parseSplunkObservabilityMetadataTestData
	triggerIndex     int
	metricName       string
}

var validSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "my-super-secret-access-token",
	"realm":       "my-realm",
}

var invalidSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "",
	"realm":       "my-realm",
}

var validSplunkObservabilityMetadata = map[string]string{
	"query":                 "data('demo.trans.latency').max().publish()",
	"duration":              "10",
	"targetValue":           "200.0",
	"queryAggregator":       "avg",
	"activationTargetValue": "1.1",
}

var testSplunkObservabilityMetadata = []parseSplunkObservabilityMetadataTestData{
	// Valid metadata and valid auth params, pass.
	{validSplunkObservabilityMetadata, validSplunkObservabilityAuthParams, false},
	// no params at all, fail
	{map[string]string{}, map[string]string{}, true},
	// No metadada but valid auth, fail.
	{map[string]string{}, validSplunkObservabilityAuthParams, true},
	// Valid metadada but no auth params, fail.
	{validSplunkObservabilityMetadata, map[string]string{}, true},
	// Missing 'query' field, fail
	{map[string]string{"duration": "10", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'duration' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'targetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'queryAggregator' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'activationTargetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, validSplunkObservabilityAuthParams, true},
	// Empty 'accessToken' field
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, invalidSplunkObservabilityAuthParams, true},
}

var SplunkObservabilityMetricIdentifiers = []SplunkObservabilityMetricIdentifier{
	{&testSplunkObservabilityMetadata[0], 0, "s0-signalfx"},
	{&testSplunkObservabilityMetadata[0], 1, "s1-signalfx"},
}

func TestSplunkObservabilityParseMetadata(t *testing.T) {
	for _, testData := range testSplunkObservabilityMetadata {
		_, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSplunkObservabilityGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range SplunkObservabilityMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSplunkObservabilityAuthParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse Splunk Observability metadata:", err)
		}
		mockSplunkObservabilityScaler := splunkObservabilityScaler{
			metadata: meta,
		}

		metricSpec := mockSplunkObservabilityScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.metricName {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

// newFakeSplunkO11yScaler wires a scaler to a fake backend that streams indefinitely without closing.
func newFakeSplunkO11yScaler(t *testing.T, program string, duration int) (*splunkObservabilityScaler, func()) {
	t.Helper()

	fake := signalflow.NewRunningFakeBackend()
	client, err := fake.Client()
	if err != nil {
		fake.Stop()
		t.Fatal("could not create fake backend client:", err)
	}

	tsid := idtool.ID(1)
	fake.AddProgramTSIDs(program, []idtool.ID{tsid})
	fake.SetTSIDFloatData(tsid, 42.0)

	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query:           program,
			Duration:        duration,
			QueryAggregator: "max",
		},
		apiClient: client,
		logger:    logr.Discard(),
	}

	return scaler, fake.Stop
}

// Regression guard: a stuck stream must not block getQueryResult past the parent context deadline.
func TestSplunkObservabilityGetQueryResultReturnsOnParentContextCancel(t *testing.T) {
	const program = "data('demo.trans.latency').max().publish()"
	// Large duration so the stopTimer never fires; the parent deadline must bound the call.
	scaler, stop := newFakeSplunkO11yScaler(t, program, 3600)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		_, _ = scaler.getQueryResult(ctx)
	}()

	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("getQueryResult returned after %v, far longer than the context deadline", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("getQueryResult did not return after parent context was cancelled; it is hanging")
	}
}

func newFakeSplunkO11yScalerWithAggregator(t *testing.T, program string, duration int, aggregator string, tsidVals map[idtool.ID]float64) (*splunkObservabilityScaler, func()) {
	t.Helper()
	fake := signalflow.NewRunningFakeBackend()
	client, err := fake.Client()
	if err != nil {
		fake.Stop()
		t.Fatal("could not create fake backend client:", err)
	}
	tsids := make([]idtool.ID, 0, len(tsidVals))
	for tsid, val := range tsidVals {
		tsids = append(tsids, tsid)
		fake.SetTSIDFloatData(tsid, val)
	}
	fake.AddProgramTSIDs(program, tsids)
	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query:           program,
			Duration:        duration,
			QueryAggregator: aggregator,
		},
		apiClient: client,
		logger:    logr.Discard(),
	}
	return scaler, fake.Stop
}

func TestSplunkObservabilityAggregators(t *testing.T) {
	const program = "data('demo.trans.latency').publish()"

	t.Run("single series", func(t *testing.T) {
		for _, agg := range []string{"max", "min", "avg", "sum", "count", "latest"} {
			agg := agg
			t.Run(agg, func(t *testing.T) {
				t.Parallel()
				tsid := idtool.ID(1)
				scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, agg, map[idtool.ID]float64{tsid: 42.0})
				defer stop()
				ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
				defer cancel()
				result, err := scaler.getQueryResult(ctx)
				if err != nil {
					t.Fatalf("aggregator %s error: %v", agg, err)
				}
				switch agg {
				case "max", "min", "avg", "latest":
					if result != 42.0 {
						t.Errorf("aggregator %s expected 42 got %v", agg, result)
					}
				case "count":
					if result < 1 {
						t.Errorf("count expected >=1 got %v", result)
					}
				case "sum":
					if result < 42.0 || int(result)%42 != 0 {
						t.Errorf("sum expected multiple of 42 got %v", result)
					}
				}
			})
		}
	})

	t.Run("multi series", func(t *testing.T) {
		for _, agg := range []string{"max", "min", "avg", "latest"} {
			agg := agg
			t.Run(agg, func(t *testing.T) {
				t.Parallel()
				tsids := map[idtool.ID]float64{
					idtool.ID(1): 10.0,
					idtool.ID(2): 20.0,
					idtool.ID(3): 30.0,
				}
				scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, agg, tsids)
				defer stop()
				ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
				defer cancel()
				result, err := scaler.getQueryResult(ctx)
				if err != nil {
					t.Fatalf("agg %s error: %v", agg, err)
				}
				var expected float64
				switch agg {
				case "max":
					expected = 30.0
				case "min":
					expected = 10.0
				case "avg":
					expected = 20.0
				case "latest":
					expected = 30.0
				}
				if result != expected {
					t.Errorf("agg %s expected %v got %v", agg, expected, result)
				}
			})
		}
		// count and sum are proportional to message count, check divisibility
		t.Run("count", func(t *testing.T) {
			t.Parallel()
			tsids := map[idtool.ID]float64{
				idtool.ID(1): 10.0,
				idtool.ID(2): 20.0,
				idtool.ID(3): 30.0,
			}
			scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, "count", tsids)
			defer stop()
			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()
			result, err := scaler.getQueryResult(ctx)
			if err != nil {
				t.Fatalf("count error: %v", err)
			}
			if int(result)%3 != 0 || result < 3 {
				t.Errorf("count expected multiple of 3 got %v", result)
			}
		})
		t.Run("sum", func(t *testing.T) {
			t.Parallel()
			tsids := map[idtool.ID]float64{
				idtool.ID(1): 10.0,
				idtool.ID(2): 20.0,
				idtool.ID(3): 30.0,
			}
			scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, "sum", tsids)
			defer stop()
			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()
			result, err := scaler.getQueryResult(ctx)
			if err != nil {
				t.Fatalf("sum error: %v", err)
			}
			// each message sum =60, so result should be multiple of 10 and >=60
			if result < 60 || int(result)%10 != 0 {
				t.Errorf("sum expected >=60 multiple of 10 got %v", result)
			}
		})
	})

	t.Run("invalid aggregator", func(t *testing.T) {
		t.Parallel()
		tsid := idtool.ID(1)
		scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, "invalid", map[idtool.ID]float64{tsid: 42.0})
		defer stop()
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		_, err := scaler.getQueryResult(ctx)
		if err == nil {
			t.Error("expected error for invalid aggregator")
		}
	})

	t.Run("no aggregator multi-series should error", func(t *testing.T) {
		t.Parallel()
		tsids := map[idtool.ID]float64{
			idtool.ID(1): 10.0,
			idtool.ID(2): 20.0,
		}
		scaler, stop := newFakeSplunkO11yScalerWithAggregator(t, program, 2, "", tsids)
		defer stop()
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		_, err := scaler.getQueryResult(ctx)
		if err == nil {
			t.Error("expected error for multi-series without aggregator")
		}
	})
}
