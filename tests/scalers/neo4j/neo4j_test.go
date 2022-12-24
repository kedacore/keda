//go:build e2e
// +build e2e

package neo4j_test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "neo4j-test"
)

var (
	scalerName       = "test-release"
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	protocol         = "neo4j"
	neo4jPassword    = "mypassword"
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-sobj", testName)
	neo4jNamespace   = "neo4j-ns"
	neo4jUser        = "neo4j"
	neo4jHelmRepo    = "https://helm.neo4j.com/neo4j"
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace                string
	PasswordArg                  string
	UsernameArg                  string
	ScalerName                   string
	Protocol                     string
	DeploymentName               string
	HostName                     string
	Port                         string
	Username                     string
	Password                     string
	Neo4jNamespace               string
	SecretName                   string
	TriggerAuthName              string
	ScaledObjectName             string
	MinReplicaCount				 int
	MaxReplicaCount              int
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
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: neo4j
    metadata:
      host: {{.HostName}}
      protocol: {{.Protocol}}
      port: "7687"
      queryValue: "9"
      query: 'MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1'
      activationQueryValue: "9"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func CreateKubernetesResourcesNeo4j(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, templates []Template) {
	KubectlApplyMultipleWithTemplate(t, data, templates)
}

func TestNeo4jScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	// setup neo4j
	CreateNamespace(t, kc, testNamespace)
	installNeo4j(t)

	data, templates := getTemplateData()
	CreateKubernetesResourcesNeo4j(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func installNeo4j(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add neo4j %s", neo4jHelmRepo))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("helm install --wait %s neo4j/neo4j --namespace %s --set neo4j.name=%s --set neo4j.password=%s --set volumes.data.mode=defaultStorageClass", scalerName, testNamespace, neo4jUser, neo4jPassword))
}

func deployPodActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	const activationPodTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: neo4j-demo-activation
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: tanishabanik/neo4j-demo:0.0.8
    name: neo4j-demo-activation
    args: ["neo4j://{{.ScalerName}}.{{.TestNamespace}}.svc.cluster.local:7687",
	      'CREATE (ac1:Person { name: "Danish", from: "Colombo", popularfor: "singer" }),
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
	       (ac3)<-[:FOLLOWS]-(ac7),(ac3)<-[:FOLLOWS]-(ac1),
	       (ac10)<-[:FOLLOWS]-(ac3),(ac10)<-[:FOLLOWS]-(ac2),(ac10)<-[:FOLLOWS]-(ac1)
	       return ac1,ac2,ac3,ac4,ac5,ac6,ac7,ac8,ac9,ac10,ac11',{{.UsernameArg}},{{.PasswordArg}}]
  restartPolicy: OnFailure
`
	KubectlApplyWithTemplate(t, data, "activationPodTemplate", activationPodTemplate)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	deployPodActivation(t, kc, data)
	time.Sleep(time.Second * 60)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minReplicaCount)
}

func deployPodUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	const scaleUpPodTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: neo4j-demo-up
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: tanishabanik/neo4j-demo:0.0.8
    name: neo4j-demo-up
    args: ["neo4j://{{.ScalerName}}.{{.TestNamespace}}.svc.cluster.local:7687",
	      "match(e:Person{name:'Robert'}),(d:Person{name:'Saurav'}) create (e)-[m:FOLLOWS]->(d) return e,m,d",{{.UsernameArg}},{{.PasswordArg}}]
  restartPolicy: OnFailure
`
	KubectlApplyWithTemplate(t, data, "scaleUpPodTemplate", scaleUpPodTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	deployPodUp(t, kc, data)
	time.Sleep(time.Second * 60)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", maxReplicaCount)
}

func deployPodDown(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	const scaleDownPodTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: neo4j-demo-down
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: tanishabanik/neo4j-demo:0.0.8
    name: neo4j-demo-down
    args: ["neo4j://{{.ScalerName}}.{{.TestNamespace}}.svc.cluster.local:7687",
	      'match(n:Person) detach delete n',{{.UsernameArg}},{{.PasswordArg}}]
  restartPolicy: OnFailure
`
	KubectlApplyWithTemplate(t, data, "scaleDownPodTemplate", scaleDownPodTemplate)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	deployPodDown(t, kc, data)
	time.Sleep(time.Second * 900)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 4),
		"replica count should be %d after 1 minute", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	passwordEncoded := base64.StdEncoding.EncodeToString([]byte(neo4jPassword))
	hostName := fmt.Sprintf("%s.%s.svc.cluster.local", scalerName, testNamespace)
	return templateData{
			TestNamespace:    testNamespace,
			PasswordArg:      neo4jPassword,
			UsernameArg:      neo4jUser,
			ScalerName:       scalerName,
			Protocol:         protocol,
			DeploymentName:   deploymentName,
			HostName:         hostName,
			Port:             "7687",
			Username:         base64.StdEncoding.EncodeToString([]byte(neo4jUser)),
			Password:         passwordEncoded,
			TriggerAuthName:  triggerAuthName,
			SecretName:       secretName,
			ScaledObjectName: scaledObjectName,
			Neo4jNamespace:   neo4jNamespace,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
