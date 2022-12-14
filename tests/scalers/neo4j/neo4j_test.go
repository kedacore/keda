//go:build e2e
// +build e2e

package neo4j_test

import (
	// "bytes"
	"context"
	// "encoding/base64"
	"fmt"
	// "io"
	// "strings"
	"testing"
	// "time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "neo4j-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-sobj", testName)
	neo4jNamespace = "neo4j-ns"
	neo4jUser      = "neo4j"
	neo4jHelmRepo  = "https://helm.neo4j.com/neo4j"
	minReplicaCount  = 1
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace                string
	DeploymentName               string
	HostName                     string
	Port                         string
	Username                     string
	Password                     string
	Neo4jNamespace               string
	SecretName                   string
	TriggerAuthName              string
	ScaledObjectName             string
	Connection, Base64Connection string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 1
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
        image: nginx:alpine
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  username: {{.Username}}
  password: {{.Password}}
`
	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
  - parameter: username
    name: {{.SecretName}}
    key: username
`
	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 1
  maxReplicaCount: 2
  triggers:
  - type: couchdb
    metadata:
      host: {{.HostName}}
      port: "7687"
      queryValue: "1"
      query: >-
		CREATE (ac1:Person { name: "Danish", from: "Colombo", popularfor: "singer" }),
		(ac2:Person { name: "Saanvi", from: "Delhi", popularfor: "actress" }),
		(ac3:Person { name: "Saurav", from: "Mumbai", popularfor: "Badminton" }),
		(ac4:Person { name: "Robert", from: "California", popularfor: "Swimming" }),
		(ac5:Person { name: "Sezzi", from: "Florida", profession: "Doctor" }),
		(ac6:Person { name: "Sally", from: "Texas", profession: "Software Engineer" }),
		(ac7:Person { name: "Sohali", from: "Gandhinagar", popularfor: "Tiktok" }),
		(ac8:Person { name: "Ranauk", from: "Bangalore", profession: "Entrepreneur" }),
		(ac9:Person { name: "Soha", from: "Bhopal", profession: "Charter Accountant" }),
		(ac10:Person { name: "Ananya", from: "Paris", popularfor: "Lawn Tennis" }),
		(ac11:Person { name: "Anna", from: "Moscow", profession: "Software Engineer" }),
		(ac1)-[:FOLLOWS]->(ac2),(ac2)<-[:FOLLOWS]-(ac1),(ac5)-[:FOLLOWS]->(ac1),
		(ac6)-[:FOLLOWS]->(ac1),(ac8)-[:FOLLOWS]->(ac1),(ac9)-[:FOLLOWS]->(ac1),
		(ac11)-[:FOLLOWS]->(ac1),(ac5)-[:FOLLOWS]->(ac2),(ac6)-[:FOLLOWS]->(ac2),
		(ac8)-[:FOLLOWS]->(ac2),(ac9)-[:FOLLOWS]->(ac2),(ac11)-[:FOLLOWS]->(ac2),
		(ac5)-[:FOLLOWS]->(ac3),(ac6)-[:FOLLOWS]->(ac3),(ac8)-[:FOLLOWS]->(ac3),
		(ac9)-[:FOLLOWS]->(ac3),(ac11)-[:FOLLOWS]->(ac3),(ac5)-[:FOLLOWS]->(ac4),
		(ac6)-[:FOLLOWS]->(ac4),(ac8)-[:FOLLOWS]->(ac4),(ac9)-[:FOLLOWS]->(ac4),
		(ac11)-[:FOLLOWS]->(ac4),(ac5)-[:FOLLOWS]->(ac7),(ac6)-[:FOLLOWS]->(ac7),
		(ac5)-[:FOLLOWS]->(ac10),(ac6)-[:FOLLOWS]->(ac10),(ac8)-[:FOLLOWS]->(ac10),
		(ac5)<-[:FOLLOWS]-(ac6),(ac5)<-[:FOLLOWS]-(ac8),(ac5)<-[:FOLLOWS]-(ac11),
		(ac6)<-[:FOLLOWS]-(ac5),(ac6)<-[:FOLLOWS]-(ac8),(ac6)<-[:FOLLOWS]-(ac11),
		(ac6)<-[:FOLLOWS]-(ac9),(ac11)<-[:FOLLOWS]-(ac9),(ac11)<-[:FOLLOWS]-(ac6),
		(ac8)<-[:FOLLOWS]-(ac11),(ac8)<-[:FOLLOWS]-(ac9),(ac9)<-[:FOLLOWS]-(ac5),
		(ac10)<-[:FOLLOWS]-(ac7),(ac4)<-[:FOLLOWS]-(ac7),(ac3)<-[:FOLLOWS]-(ac7),
		(ac3)<-[:FOLLOWS]-(ac10),(ac1)<-[:FOLLOWS]-(ac7),(ac2)<-[:FOLLOWS]-(ac7),
		(ac3)<-[:FOLLOWS]-(ac7),(ac3)<-[:FOLLOWS]-(ac4),(ac3)<-[:FOLLOWS]-(ac1),
		(ac10)<-[:FOLLOWS]-(ac3),(ac10)<-[:FOLLOWS]-(ac2),(ac10)<-[:FOLLOWS]-(ac1)
		return ac1,ac2,ac3,ac4,ac5,ac6,ac7,ac8,ac9,ac10,ac11;MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1
      activationQueryValue: "1"
      metricName: "global-metric"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestNeo4jScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	// setup neo4j
	CreateNamespace(t, kc, testNamespace)
	installNeo4j(t)

	// data, templates := getTemplateData(kc)
	// CreateKubernetesResourcesCouchDB(t, kc, testNamespace, data, templates)

	// assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
	// 	"replica count should be %d after 3 minutes", minReplicaCount)

	// // test scaling
	// testActivation(t, kc)
	// testScaleUp(t, kc)
	// testScaleDown(t, kc)

	// cleanup
	// DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func installNeo4j(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add neo4j %s", neo4jHelmRepo))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl create namespace %s", testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("helm install my-neo4j-release neo4j/neo4j --namespace %s -f https://raw.githubusercontent.com/26tanishabanik/manifests/main/values.yaml", testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl --namespace %s rollout status --watch --timeout=600s statefulset/my-neo4j-release", testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getPassword(kc *kubernetes.Clientset, namespace string) string {
	secret, _ := kc.CoreV1().Secrets(namespace).Get(context.Background(), "test-release-neo4j", metav1.GetOptions{})
	encodedPassword := secret.Data["adminPassword"]
	password := string(encodedPassword)
	return password
}