//go:build e2e
// +build e2e

package arangodb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	Namespace  string
	Database   string
	Collection string
}

const (
	arangoDeploymentTemplate = `apiVersion: "database.arangodb.com/v1"
kind: "ArangoDeployment"
metadata:
  name: "example-arangodb-cluster"
  namespace: {{.Namespace}}
spec:
  architectures:
    - arm64
    - amd64
  mode: Single
  image: "arangodb/arangodb:3.10.1"
`

	createDatabaseTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: create-db
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
        - image: ghcr.io/nginx/nginx-unprivileged:1.26
          name: alpine
          command: ["/bin/sh"]
          args: ["-c", "curl -H 'Authorization: Basic cm9vdDo=' --location --request POST 'https://example-arangodb-cluster-ea.{{.Namespace}}.svc.cluster.local:8529/_api/database' --data-raw '{\"name\": \"{{.Database}}\"}' -k"]
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            seccompProfile:
              type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`

	createCollectionTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: create-arangodb-collection
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
        - image: ghcr.io/nginx/nginx-unprivileged:1.26
          name: alpine
          command: ["/bin/sh"]
          args: ["-c", "curl -H 'Authorization: Basic cm9vdDo=' --location --request POST 'https://example-arangodb-cluster-ea.{{.Namespace}}.svc.cluster.local:8529/_db/{{.Database}}/_api/collection' --data-raw '{\"name\": \"{{.Collection}}\"}' -k"]
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            seccompProfile:
              type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

func InstallArangoDB(t *testing.T, kc *kubernetes.Clientset, testNamespace string) {
	t.Log("installing arangodb crds")
	_, err := helper.ExecuteCommand(fmt.Sprintf("helm install arangodb-crds https://github.com/arangodb/kube-arangodb/releases/download/1.2.20/kube-arangodb-crd-1.2.20.tgz --namespace=%s --wait", testNamespace))
	require.NoErrorf(t, err, "cannot install crds - %s", err)

	t.Log("installing arangodb operator")
	_, err = helper.ExecuteCommand(fmt.Sprintf("helm install arangodb https://github.com/arangodb/kube-arangodb/releases/download/1.2.20/kube-arangodb-1.2.20.tgz --set 'operator.architectures={arm64,amd64}' --set 'operator.resources.requests.cpu=1m' --set 'operator.resources.requests.memory=1Mi' --namespace=%s --wait", testNamespace))
	require.NoErrorf(t, err, "cannot create operator deployment - %s", err)

	t.Log("creating arangodeployment resource")
	helper.KubectlApplyWithTemplate(t, templateData{Namespace: testNamespace}, "arangoDeploymentTemplate", arangoDeploymentTemplate)
	require.True(t, helper.WaitForPodCountInNamespace(t, kc, testNamespace, 3, 5, 20), "pod count should be 3")
	require.True(t, helper.WaitForAllPodRunningInNamespace(t, kc, testNamespace, 5, 20), "all pods should be running")
}

func SetupArangoDB(t *testing.T, kc *kubernetes.Clientset, testNamespace, arangoDBName, arangoDBCollection string) {
	helper.KubectlApplyWithTemplate(t, templateData{Namespace: testNamespace, Database: arangoDBName}, "createDatabaseTemplate", createDatabaseTemplate)
	require.True(t, helper.WaitForJobSuccess(t, kc, "create-db", testNamespace, 5, 10), "create database job failed")

	helper.KubectlApplyWithTemplate(t, templateData{Namespace: testNamespace, Database: arangoDBName, Collection: arangoDBCollection}, "createCollectionTemplate", createCollectionTemplate)
	require.True(t, helper.WaitForJobSuccess(t, kc, "create-arangodb-collection", testNamespace, 5, 10), "create collection job failed")
}

func UninstallArangoDB(t *testing.T, namespace string) {
	helper.KubectlDeleteMultipleWithTemplate(t, templateData{Namespace: namespace}, []helper.Template{{Name: "arangoDeploymentTemplate", Config: arangoDeploymentTemplate}})

	_, err := helper.ExecuteCommand(fmt.Sprintf("helm uninstall arangodb --namespace=%s --wait", namespace))
	assert.NoErrorf(t, err, "cannot uninstall arangodb operator - %s", err)

	_, err = helper.ExecuteCommand(fmt.Sprintf("helm uninstall arangodb-crds --namespace=%s --wait", namespace))
	assert.NoErrorf(t, err, "cannot uninstall arangodb crds - %s", err)
}
