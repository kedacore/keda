//go:build e2e
// +build e2e

package accurate_scaling_strategy_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

var _ = godotenv.Load("../../.env") // For loading env variables from .env

const (
	testName = "accurate-scaling-strategy-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	scaledJobName    = fmt.Sprintf("%s-sj", testName)
	connectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	queueName        = fmt.Sprintf("queue-%d", GetRandomNumber())
	secretName       = fmt.Sprintf("%s-secret", testName)
)

// YAML templates for your Kubernetes resources
const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AzureWebJobsStorage: {{.Connection}}
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScaledJobName}}
spec:
  jobTargetRef:
    template:
      spec:
        initContainers:
          - name: drainer
            image: python:3.12-alpine
            imagePullPolicy: IfNotPresent
            env:
            - name: QUEUE_NAME
              value: {{.QueueName}}
            - name: AZURE_STORAGE_CONNECTION_STRING
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: AzureWebJobsStorage
            command:
            - python3
            - -c
            - |
              import base64, hashlib, hmac, os, time, urllib.request, urllib.parse
              import xml.etree.ElementTree as ET
              from datetime import datetime, timezone
              V = "2021-08-06"
              d = dict(p.split("=", 1) for p in os.environ["AZURE_STORAGE_CONNECTION_STRING"].split(";") if "=" in p)
              acct, key, sfx = d["AccountName"], base64.b64decode(d["AccountKey"]), d.get("EndpointSuffix", "core.windows.net")
              q = os.environ["QUEUE_NAME"]
              def call(method, path, query):
                  now = datetime.now(timezone.utc).strftime("%a, %d %b %Y %H:%M:%S GMT")
                  h = {"x-ms-date": now, "x-ms-version": V}
                  ch = "".join("%s:%s\n" % (k, h[k]) for k in sorted(h))
                  cr = "/%s%s" % (acct, path) + "".join("\n%s:%s" % (k, query[k]) for k in sorted(query))
                  sts = "\n".join([method, "", "", "", "", "", "", "", "", "", "", ""]) + "\n" + ch + cr
                  h["Authorization"] = "SharedKey %s:%s" % (acct, base64.b64encode(hmac.new(key, sts.encode(), hashlib.sha256).digest()).decode())
                  url = "https://%s.queue.%s%s" % (acct, sfx, path)
                  if query:
                      url += "?" + urllib.parse.urlencode(query)
                  with urllib.request.urlopen(urllib.request.Request(url, method=method, headers=h)) as r:
                      return r.read()
              for i in range(60):
                  m = ET.fromstring(call("GET", "/%s/messages" % q, {"numofmessages": "1", "visibilitytimeout": "60"})).find("QueueMessage")
                  if m is not None:
                      call("DELETE", "/%s/messages/%s" % (q, m.findtext("MessageId")), {"popreceipt": m.findtext("PopReceipt")})
                      print("drained", m.findtext("MessageId"), flush=True)
                      break
                  print("no message yet, retry", i, flush=True)
                  time.sleep(2)
              else:
                  print("no message drained, proceeding", flush=True)
        containers:
          - name: sleeper
            image: python:3.12-alpine
            command:
            - sleep
            - "900"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 10
  scalingStrategy:
    strategy: "accurate"
  triggers:
    - type: azure-queue
      metadata:
        queueName: {{.QueueName}}
        connectionFromEnv: AzureWebJobsStorage
        queueLength: '1'
`
)

type templateData struct {
	ScaledJobName string
	TestNamespace string
	QueueName     string
	SecretName    string
	Connection    string
}

func TestScalingStrategy(t *testing.T) {
	// Setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure queue test")

	queueClient, err := azqueue.NewQueueClientFromConnectionString(connectionString, queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue client - %s", err)
	_, err = queueClient.Create(ctx, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		_, err := queueClient.Delete(ctx, nil)
		assert.NoErrorf(t, err, "cannot delete the queue - %s", err)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	testAccurateScaling(ctx, t, kc, queueClient)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			// Populate fields required in YAML templates
			ScaledJobName: scaledJobName,
			TestNamespace: testNamespace,
			QueueName:     queueName,
			Connection:    base64.StdEncoding.EncodeToString([]byte(connectionString)),
			SecretName:    secretName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func testAccurateScaling(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, client *azqueue.QueueClient) {
	// job count appears within a couple of operator polls (pollingInterval 5s)
	jobCountIterations := 60
	// running-pod waits must take into account image pull + drain on a cold node
	runningPodIterations := 240

	// Phase 1 - non-cap branch, no in-flight consumers.
	// Enqueue 4 (< maxReplicaCount). maxScale=4, running=0, pending=0 -> create maxScale-pending = 4 jobs.
	// Each job drains one message; the queue empties and the 4 pods become Running. No overshoot:
	// while draining the pods are Pending, so pending=4 holds maxScale-pending at 0.
	enqueueMessages(ctx, t, client, 4)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 4, jobCountIterations, 1),
		"job count should be %d after %d iterations", 4, jobCountIterations)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 4, runningPodIterations, 1),
		"running pod count should be %d after %d iterations", 4, runningPodIterations)

	// Phase 2 - still non-cap, now with 4 running (unfinished, sleeping) jobs.
	// Enqueue 4. maxScale=4, running=4, pending=0 -> 4+4 <= maxReplicaCount, so create maxScale-pending = 4.
	enqueueMessages(ctx, t, client, 4)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 8, jobCountIterations, 1),
		"job count should be %d after %d iterations", 8, jobCountIterations)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 8, runningPodIterations, 1),
		"running pod count should be %d after %d iterations", 8, runningPodIterations)

	// Phase 3 - cap branch. Enqueue 4 with 8 running.
	// maxScale=4, running=8 -> maxScale+running=12 > maxReplicaCount(10), so the cap branch creates
	// maxReplicaCount-running = 2 jobs, pinning the total at 10. The 2 new jobs drain 2 messages
	// (2 remain queued, cleaned up on teardown) and the running count never exceeds maxReplicaCount.
	enqueueMessages(ctx, t, client, 4)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 10, jobCountIterations, 1),
		"job count should be capped at %d after %d iterations", 10, jobCountIterations)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 10, runningPodIterations, 1),
		"running pod count should be %d after %d iterations", 10, runningPodIterations)
}

func enqueueMessages(ctx context.Context, t *testing.T, client *azqueue.QueueClient, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := client.EnqueueMessage(ctx, msg, nil)
		assert.NoErrorf(t, err, "cannot enqueue message - %s", err)
		t.Logf("Message queued")
	}
}
