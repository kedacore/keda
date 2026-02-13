//go:build e2e
// +build e2e

package elastic_forecast_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

var _ = godotenv.Load("../../.env")

const (
	forecastTestName = "elastic-forecast-test"
)

var (
	forecastTestNamespace    = fmt.Sprintf("%s-ns", forecastTestName)
	forecastDeploymentName   = fmt.Sprintf("%s-deployment", forecastTestName)
	forecastScaledObjectName = fmt.Sprintf("%s-so", forecastTestName)
	forecastSecretName       = fmt.Sprintf("%s-secret", forecastTestName)
	forecastPassword         = "passw0rd"
	forecastJobID            = "keda-forecast-job"
	forecastMaxReplicaCount  = 2

	forecastKubectlExecCmd = fmt.Sprintf(
		"kubectl exec -n %s elastic-forecast-0 -- curl -sS -f -H 'Content-Type: application/json' -u 'elastic:%s'",
		forecastTestNamespace, forecastPassword,
	)
)

type forecastTemplateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	SecretName            string
	ElasticPassword       string
	ElasticPasswordBase64 string
	ForecastJobID         string
}

const (
	forecastSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  password: {{.ElasticPasswordBase64}}
`

	forecastTriggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-elastic-forecast-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
`

	forecastDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	forecastElasticTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elastic-forecast
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      name: elastic-forecast
  template:
    metadata:
      labels:
        name: elastic-forecast
    spec:
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:7.17.0
        imagePullPolicy: IfNotPresent
        env:
          - name: ES_JAVA_OPTS
            value: -Xms512m -Xmx512m
          - name: cluster.name
            value: elastic-forecast-keda
          - name: discovery.type
            value: single-node
          - name: ELASTIC_PASSWORD
            value: "{{.ElasticPassword}}"
          - name: xpack.security.enabled
            value: "true"
          - name: xpack.ml.enabled
            value: "true"
          - name: node.ml
            value: "true"
          - name: node.store.allow_mmap
            value: "false"
        ports:
        - containerPort: 9200
          name: http
          protocol: TCP
        readinessProbe:
          exec:
            command:
              - /usr/bin/curl
              - -sS
              - -u
              - "elastic:{{.ElasticPassword}}"
              - http://127.0.0.1:9200
          failureThreshold: 3
          initialDelaySeconds: 15
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 5
  serviceName: {{.DeploymentName}}
`

	forecastServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 9200
    targetPort: 9200
    protocol: TCP
  selector:
    name: elastic-forecast
`

	forecastScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 3
  cooldownPeriod: 5
  triggers:
    - type: elastic-forecast
      metadata:
        addresses: "http://{{.DeploymentName}}.{{.TestNamespace}}.svc:9200"
        username: "elastic"
        jobID: "{{.ForecastJobID}}"
        index: "shared"
        lookAhead: "5m"
        targetValue: "1"
        activationTargetValue: "0"
      authenticationRef:
        name: keda-trigger-auth-elastic-forecast-secret
`
)

// Anomaly-detection job counting events per 1-minute bucket.
// 1m bucket_span with 2h of training data gives ~120 buckets enough for a reliable forecast.
const forecastMLJobPayload = `{
  "analysis_config": {
    "bucket_span": "1m",
    "detectors": [{"function": "count"}]
  },
  "data_description": {
    "time_field": "@timestamp",
    "time_format": "epoch_ms"
  }
}`

// Index mapping for the synthetic time-series that trains the ML model.
const forecastIndexMapping = `{
  "mappings": {
    "properties": {
      "@timestamp": {"type": "date"},
      "value":      {"type": "double"}
    }
  },
  "settings": {
    "number_of_replicas": 0,
    "number_of_shards":   1
  }
}`

func TestElasticForecastScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getForecastTemplateData()

	CreateKubernetesResources(t, kc, forecastTestNamespace, data, templates)
	defer DeleteKubernetesResources(t, forecastTestNamespace, data, templates)

	setupElasticForecast(t)

	KubectlApplyWithTemplate(t, data, "forecastScaledObjectTemplate", forecastScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "forecastScaledObjectTemplate", forecastScaledObjectTemplate)

	t.Log("--- [setup] waiting for forecast documents to be indexed ---")
	waitForForecastDocs(t)

	testElasticForecastScaler(t, kc)
}

func setupElasticForecast(t *testing.T) {
	t.Helper()
	kc := GetKubernetesClient(t)

	// Step 1: wait for the ES pod to be ready.
	t.Log("--- [setup] waiting for Elasticsearch pod to be ready ---")
	require.True(t,
		WaitForStatefulsetReplicaReadyCount(t, kc, "elastic-forecast", forecastTestNamespace, 1, 60, 5),
		"elastic-forecast StatefulSet should have 1 ready replica within 5 minutes",
	)

	// Step 2: activate trial licence so ML features are available.
	t.Log("--- [setup] activating trial licence ---")
	activateTrialAndVerify(t)

	// Step 3: create the index that will hold training data.
	t.Log("--- [setup] creating training-data index ---")
	execAndVerify(t, "create index keda-forecast-data",
		fmt.Sprintf("%s -XPUT http://127.0.0.1:9200/keda-forecast-data -d '%s'",
			forecastKubectlExecCmd, forecastIndexMapping),
		func(body string) error {
			if !strings.Contains(body, `"acknowledged":true`) {
				return fmt.Errorf("unexpected response: %s", body)
			}
			return nil
		},
	)

	// Step 4: ingest synthetic training data.
	t.Log("--- [setup] ingesting synthetic training data ---")
	ingestForecastTrainingData(t, 240, 2*time.Hour)

	// Step 5: create the ML anomaly-detection job.
	t.Log("--- [setup] creating ML job ---")
	execAndVerify(t, "create ML job",
		fmt.Sprintf("%s -XPUT http://127.0.0.1:9200/_ml/anomaly_detectors/%s -d '%s'",
			forecastKubectlExecCmd, forecastJobID, forecastMLJobPayload),
		func(body string) error {
			if !strings.Contains(body, fmt.Sprintf(`"job_id":"%s"`, forecastJobID)) {
				return fmt.Errorf("job_id not found in response: %s", body)
			}
			return nil
		},
	)

	// Step 6: create the datafeed that links the job to the index.
	t.Log("--- [setup] creating datafeed ---")
	datafeedPayload := fmt.Sprintf(
		`{"job_id":"%s","indices":["keda-forecast-data"],"query":{"match_all":{}},"query_delay":"0s"}`,
		forecastJobID,
	)
	execAndVerify(t, "create datafeed",
		fmt.Sprintf("%s -XPUT http://127.0.0.1:9200/_ml/datafeeds/datafeed-%s -d '%s'",
			forecastKubectlExecCmd, forecastJobID, datafeedPayload),
		func(body string) error {
			if !strings.Contains(body, fmt.Sprintf(`"datafeed_id":"datafeed-%s"`, forecastJobID)) {
				return fmt.Errorf("datafeed_id not found in response: %s", body)
			}
			return nil
		},
	)

	// Step 7: open the job (required before starting the datafeed).
	t.Log("--- [setup] opening ML job ---")
	execAndVerify(t, "open ML job",
		fmt.Sprintf("%s -XPOST http://127.0.0.1:9200/_ml/anomaly_detectors/%s/_open",
			forecastKubectlExecCmd, forecastJobID),
		func(body string) error {
			if !strings.Contains(body, `"opened":true`) {
				return fmt.Errorf("job did not open successfully: %s", body)
			}
			return nil
		},
	)

	// Step 8: start the datafeed over the full 2-hour historical window.
	t.Log("--- [setup] starting datafeed over historical window ---")
	startMs := time.Now().Add(-2 * time.Hour).UnixMilli()
	endMs := time.Now().Add(1 * time.Minute).UnixMilli()
	startPayload := fmt.Sprintf(`{"start":"%d","end":"%d"}`, startMs, endMs)
	execAndVerify(t, "start datafeed",
		fmt.Sprintf("%s -XPOST http://127.0.0.1:9200/_ml/datafeeds/datafeed-%s/_start -d '%s'",
			forecastKubectlExecCmd, forecastJobID, startPayload),
		func(body string) error {
			if !strings.Contains(body, `"started":true`) {
				return fmt.Errorf("datafeed did not start: %s", body)
			}
			return nil
		},
	)

	// Step 9: wait until the datafeed has finished processing the historical window.
	t.Log("--- [setup] waiting for datafeed to finish processing ---")
	requireDatafeedStopped(t, 3*time.Minute)

	// Step 10: verify the ML job has processed enough buckets to produce a meaningful forecast.
	t.Log("--- [setup] verifying ML job has processed data ---")
	requireJobHasBuckets(t)

	// Step 11: re-open the ML job.
	t.Log("--- [setup] re-opening ML job for forecast ---")
	execAndVerify(t, "re-open ML job",
		fmt.Sprintf("%s -XPOST http://127.0.0.1:9200/_ml/anomaly_detectors/%s/_open",
			forecastKubectlExecCmd, forecastJobID),
		func(body string) error {
			if !strings.Contains(body, `"opened":true`) {
				return fmt.Errorf("job did not re-open successfully: %s", body)
			}
			return nil
		},
	)
}

// execAndVerify runs a shell command, captures stdout, and calls validate.
func execAndVerify(t *testing.T, stepName, cmd string, validate func(body string) error) {
	t.Helper()
	out, err := ExecuteCommand(cmd)
	require.NoErrorf(t, err, "[%s] command execution failed: %v", stepName, err)
	body := string(out)
	require.NoErrorf(t, validate(body), "[%s] response validation failed", stepName)
	t.Logf("[%s] OK — response: %s", stepName, truncate(body, 200))
}

// activateTrialAndVerify starts the 30-day trial and confirms the licence type is "trial" afterwards.
func activateTrialAndVerify(t *testing.T) {
	t.Helper()

	// POST to activate, tolerate "already active" responses.
	out, _ := ExecuteCommand(fmt.Sprintf(
		"%s -XPOST 'http://127.0.0.1:9200/_license/start_trial?acknowledge=true'",
		forecastKubectlExecCmd,
	))
	t.Logf("[activate trial] response: %s", truncate(string(out), 300))

	// GET current licence and confirm it is "trial".
	execAndVerify(t, "verify trial licence",
		fmt.Sprintf("%s -XGET http://127.0.0.1:9200/_license", forecastKubectlExecCmd),
		func(body string) error {
			var resp struct {
				License struct {
					Type string `json:"type"`
				} `json:"license"`
			}
			if err := json.Unmarshal([]byte(body), &resp); err != nil {
				return fmt.Errorf("cannot parse licence response: %w (body: %s)", err, body)
			}
			if resp.License.Type != "trial" && resp.License.Type != "platinum" && resp.License.Type != "enterprise" {
				return fmt.Errorf("expected trial/platinum/enterprise licence, got %q", resp.License.Type)
			}
			return nil
		},
	)
}

// requireDatafeedStopped polls the datafeed status until it reports "stopped", which means the bounded historical run has completed.
func requireDatafeedStopped(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := ExecuteCommand(fmt.Sprintf(
			"%s -XGET http://127.0.0.1:9200/_ml/datafeeds/datafeed-%s/_stats",
			forecastKubectlExecCmd, forecastJobID,
		))
		if err == nil {
			body := string(out)
			if strings.Contains(body, `"state":"stopped"`) {
				t.Log("[datafeed] state is stopped — historical processing complete")
				return
			}
			t.Logf("[datafeed] not yet stopped, waiting… (body: %s)", truncate(body, 150))
		}
		time.Sleep(3 * time.Second)
	}
	require.Fail(t, fmt.Sprintf("datafeed-%s did not reach state 'stopped' within %s", forecastJobID, timeout))
}

// requireJobHasBuckets asserts that the ML job has processed enough result buckets for the model to produce a reliable forecast.
const minForecastBuckets = 48

func requireJobHasBuckets(t *testing.T) {
	t.Helper()
	execAndVerify(t, "verify ML job buckets",
		fmt.Sprintf("%s -XGET http://127.0.0.1:9200/_ml/anomaly_detectors/%s/results/buckets?size=1",
			forecastKubectlExecCmd, forecastJobID),
		func(body string) error {
			var resp struct {
				Count int `json:"count"`
			}
			if err := json.Unmarshal([]byte(body), &resp); err != nil {
				return fmt.Errorf("cannot parse buckets response: %w (body: %s)", err, body)
			}
			if resp.Count < minForecastBuckets {
				return fmt.Errorf(
					"ML job only has %d result buckets, need at least %d for a reliable forecast (body: %s)",
					resp.Count, minForecastBuckets, body,
				)
			}
			return nil
		},
	)
}

// ingestForecastTrainingData inserts `count` synthetic documents spread evenly over the past `window`.
func ingestForecastTrainingData(t *testing.T, count int, window time.Duration) {
	t.Helper()
	interval := window / time.Duration(count)
	for i := 0; i < count; i++ {
		ts := time.Now().Add(-window).Add(time.Duration(i) * interval).UnixMilli()
		doc := fmt.Sprintf(`{"@timestamp":%d,"value":%d}`, ts, i%5+1)
		_, err := ExecuteCommand(fmt.Sprintf(
			"%s -XPOST http://127.0.0.1:9200/keda-forecast-data/_doc -d '%s'",
			forecastKubectlExecCmd, doc,
		))
		require.NoErrorf(t, err, "failed to ingest training document %d", i)
	}
	t.Logf("[training data] ingested %d documents over the past %s", count, window)
}

// truncate shortens a string to max n runes for log output.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// waitForForecastDocs polls .ml-anomalies-shared until at least one model_forecast document exists.
func waitForForecastDocs(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		out, err := ExecuteCommand(fmt.Sprintf(
			`%s -XGET 'http://127.0.0.1:9200/.ml-anomalies-shared/_count' -d '{"query":{"bool":{"filter":[{"term":{"job_id":"%s"}},{"term":{"result_type":"model_forecast"}}]}}}'`,
			forecastKubectlExecCmd, forecastJobID,
		))
		if err == nil {
			body := string(out)
			var countResp struct {
				Count int64 `json:"count"`
			}
			if jsonErr := json.Unmarshal([]byte(body), &countResp); jsonErr == nil && countResp.Count > 0 {
				t.Logf("[forecast] %d forecast document(s) indexed — proceeding", countResp.Count)
				return
			}
			t.Logf("[forecast] 0 documents yet, waiting... (response: %s)", truncate(body, 100))
		}
		time.Sleep(5 * time.Second)
	}
	require.Fail(t, "forecast documents did not appear in .ml-anomalies-shared within 3 minutes")
}

func testElasticForecastScaler(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out (forecast value > targetValue) ---")
	assert.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, forecastDeploymentName, forecastTestNamespace,
			forecastMaxReplicaCount, 60, 3),
		"replica count should reach %d within 3 minutes", forecastMaxReplicaCount,
	)
}

func getForecastTemplateData() (forecastTemplateData, []Template) {
	data := forecastTemplateData{
		TestNamespace:         forecastTestNamespace,
		DeploymentName:        forecastDeploymentName,
		ScaledObjectName:      forecastScaledObjectName,
		SecretName:            forecastSecretName,
		ElasticPassword:       forecastPassword,
		ElasticPasswordBase64: base64.StdEncoding.EncodeToString([]byte(forecastPassword)),
		ForecastJobID:         forecastJobID,
	}
	templates := []Template{
		{Name: "forecastSecretTemplate", Config: forecastSecretTemplate},
		{Name: "forecastTriggerAuthTemplate", Config: forecastTriggerAuthTemplate},
		{Name: "forecastServiceTemplate", Config: forecastServiceTemplate},
		{Name: "forecastElasticTemplate", Config: forecastElasticTemplate},
		{Name: "forecastDeploymentTemplate", Config: forecastDeploymentTemplate},
	}
	return data, templates
}
