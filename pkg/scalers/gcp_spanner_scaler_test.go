package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// resolvedEnv simulates Secret/ConfigMap values available to the scaler.
var testSpannerResolvedEnv = map[string]string{
	"CREDS_ENV":      `{"type":"service_account","project_id":"my-project"}`,
	"CREDS_FILE_ENV": "/var/secrets/sa.json",
}

// requiredFields is a convenience baseline used across multiple test cases.
var spannerRequiredFields = map[string]string{
	"projectId":  "my-project",
	"instanceId": "my-instance",
	"databaseId": "my-db",
	"query":      "SELECT COUNT(*) FROM jobs WHERE status = 'pending'",
}

// withOverrides returns a copy of base with the given key/value pairs applied.
func withOverrides(base map[string]string, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// withoutKey returns a copy of base without the named key.
func withoutKey(base map[string]string, key string) map[string]string {
	out := make(map[string]string, len(base))
	for k, v := range base {
		if k != key {
			out[k] = v
		}
	}
	return out
}

type parseSpannerMetadataTestData struct {
	desc       string
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

var testSpannerMetadataCases = []parseSpannerMetadataTestData{
	// ── required fields ──────────────────────────────────────────────────────
	{
		desc:       "all required fields missing",
		authParams: map[string]string{},
		metadata:   map[string]string{},
		isError:    true,
	},
	{
		desc:       "missing projectId",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   withoutKey(spannerRequiredFields, "projectId"),
		isError:    true,
	},
	{
		desc:       "missing instanceId",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   withoutKey(spannerRequiredFields, "instanceId"),
		isError:    true,
	},
	{
		desc:       "missing databaseId",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   withoutKey(spannerRequiredFields, "databaseId"),
		isError:    true,
	},
	{
		desc:       "missing query",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   withoutKey(spannerRequiredFields, "query"),
		isError:    true,
	},

	// ── authentication methods ────────────────────────────────────────────────
	{
		desc:       "credentials inline via authParams",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   spannerRequiredFields,
		isError:    false,
	},
	{
		desc:       "credentials from env var (credentialsFromEnv)",
		authParams: map[string]string{},
		metadata:   withOverrides(spannerRequiredFields, map[string]string{"credentialsFromEnv": "CREDS_ENV"}),
		isError:    false,
	},
	{
		desc:       "credentials from env file path (credentialsFromEnvFile)",
		authParams: map[string]string{},
		metadata:   withOverrides(spannerRequiredFields, map[string]string{"credentialsFromEnvFile": "CREDS_FILE_ENV"}),
		isError:    false,
	},
	{
		desc:       "no credentials provided (no pod identity)",
		authParams: map[string]string{},
		metadata:   spannerRequiredFields,
		isError:    true,
	},

	// ── optional numeric parameters ───────────────────────────────────────────
	{
		desc:       "custom targetValue and activationValue",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata: withOverrides(spannerRequiredFields, map[string]string{
			"targetValue":     "20",
			"activationValue": "3",
		}),
		isError: false,
	},
	{
		desc:       "targetValue defaults to 5 when omitted",
		authParams: map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		metadata:   spannerRequiredFields,
		isError:    false,
	},
}

func TestSpannerParseMetadata(t *testing.T) {
	for _, tc := range testSpannerMetadataCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			_, err := parseSpannerMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      tc.authParams,
				TriggerMetadata: tc.metadata,
				ResolvedEnv:     testSpannerResolvedEnv,
			})
			if tc.isError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.isError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSpannerParseMetadata_Defaults verifies that optional fields receive the
// correct default values when omitted from the trigger metadata.
func TestSpannerParseMetadata_Defaults(t *testing.T) {
	meta, err := parseSpannerMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
		TriggerMetadata: spannerRequiredFields,
		ResolvedEnv:     testSpannerResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.TargetValue != 5 {
		t.Errorf("TargetValue default: got %v, want 5", meta.TargetValue)
	}
	if meta.ActivationValue != 0 {
		t.Errorf("ActivationValue default: got %v, want 0", meta.ActivationValue)
	}
}

// TestSpannerParseMetadata_CredentialsFromEnvFile verifies that the file-path
// credential method is stored correctly.
func TestSpannerParseMetadata_CredentialsFromEnvFile(t *testing.T) {
	meta, err := parseSpannerMetadata(&scalersconfig.ScalerConfig{
		AuthParams: map[string]string{},
		TriggerMetadata: withOverrides(spannerRequiredFields, map[string]string{
			"credentialsFromEnvFile": "CREDS_FILE_ENV",
		}),
		ResolvedEnv: testSpannerResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/var/secrets/sa.json"
	if meta.gcpAuthorization.GoogleApplicationCredentialsFile != want {
		t.Errorf("CredentialsFile: got %q, want %q",
			meta.gcpAuthorization.GoogleApplicationCredentialsFile, want)
	}
}

// ── metric name generation ────────────────────────────────────────────────────

type spannerMetricNameCase struct {
	desc         string
	triggerIndex int
	wantName     string
}

var spannerMetricNameCases = []spannerMetricNameCase{
	{desc: "trigger index 0", triggerIndex: 0, wantName: "s0-gcp-spanner-my-instance-my-db-my-project"},
	{desc: "trigger index 1", triggerIndex: 1, wantName: "s1-gcp-spanner-my-instance-my-db-my-project"},
	{desc: "trigger index 5", triggerIndex: 5, wantName: "s5-gcp-spanner-my-instance-my-db-my-project"},
}

func TestSpannerGetMetricSpecForScaling(t *testing.T) {
	for _, tc := range spannerMetricNameCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			meta, err := parseSpannerMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
				TriggerMetadata: spannerRequiredFields,
				ResolvedEnv:     testSpannerResolvedEnv,
				TriggerIndex:    tc.triggerIndex,
			})
			if err != nil {
				t.Fatalf("parseSpannerMetadata: %v", err)
			}

			s := &spannerScaler{
				metricType: v2.AverageValueMetricType,
				metadata:   meta,
				logger:     logr.Discard(),
			}

			specs := s.GetMetricSpecForScaling(context.Background())
			if len(specs) != 1 {
				t.Fatalf("expected 1 metric spec, got %d", len(specs))
			}
			got := specs[0].External.Metric.Name
			if got != tc.wantName {
				t.Errorf("metric name: got %q, want %q", got, tc.wantName)
			}
		})
	}
}
