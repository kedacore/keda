import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava';

const defaultNamespace = 'kafka-test'
const defaultCluster = 'kafka-cluster'
const timeToWait = 300
const defaultTopic = 'kafka-topic'
const defaultKafkaClient = 'kafka-client'

const kafkaClusterYamlFile = tmp.fileSync()
const kafkaTopicYamlFile = tmp.fileSync()
const kafkaClientYamlFile = tmp.fileSync()
const kafkaApplicationYamlFile = tmp.fileSync()

// test.before('Set up, create necessary resources.', t => {
// 	sh.config.silent = true
// 	sh.exec(`kubectl create namespace ${defaultNamespace}`)
// 	// 1. - nainstaluje strimzi.io operator
// 	// - oc apply -f 'https://strimzi.io/install/latest?namespace=myproject' -n myproject
// 	// ide
// 	t.is(
// 		0,
// 		sh.exec(`kubectl apply -f 'https://strimzi.io/install/latest?namespace=${defaultNamespace}' --namespace ${defaultNamespace}`).code,
// 		'Deploying Strimzi.io operator should work.'
// 	)
// 	// 2.1 - vytvorim apache kafka cluster
// 	//     - zkombinovat:
// 	//         - navod: https://github.com/kedacore/sample-azure-functions-on-ocp4#create-a-kafka-instance
// 	//         - strimzi example: https://strimzi.io/examples/latest/kafka/kafka-persistent-single.yaml
// 	// const kafkaClusterYamlFile = tmp.fileSync()
// 	fs.writeFileSync(kafkaClusterYamlFile.name, kafkaClusterYaml)
// 	t.is(
// 		0,
// 		sh.exec(`kubectl apply -f ${kafkaClusterYamlFile.name} --namespace ${defaultNamespace}`).code,
// 		'Deploying Kafka cluster instance should work.'
// 	)
// 	// 2.2 - pockat kym sa vsetko vytvori
// 	//     - oc wait kafka/my-cluster --for=condition=Ready --timeout=300s -n myproject
// 	t.is(
// 		0,
// 		sh.exec(`kubectl wait kafka/${defaultCluster} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
// 		'Kafka instacne should be ready within given time limit.'
// 	)
// 	// 3. - vytvorit topic - yaml z navodu
// 	// const kafkaTopicYamlFile = tmp.fileSync()
// 	fs.writeFileSync(kafkaTopicYamlFile.name, kafkaTopicYaml)
// 	t.is(
// 		0,
// 		sh.exec(`kubectl apply -f ${kafkaTopicYamlFile.name} --namespace ${defaultNamespace}`).code,
// 		'Deploying Kafka topic should work.'
// 	)
// 	// 4. - deploynut kafka client - yaml z navodu
// 	// const kafkaClientYamlFile = tmp.fileSync()
// 	fs.writeFileSync(kafkaClientYamlFile.name, kafkaClientYaml)
// 	t.is(
// 		0,
// 		sh.exec(`kubectl apply -f ${kafkaClientYamlFile.name} --namespace ${defaultNamespace}`).code,
// 		'Deploying Kafka client should work.'
// 	)
// 	// 5. - deploynut twitter-function (yaml z gitu z navodu)
//   // const kafkaApplicationYamlFile = tmp.fileSync()
// 	fs.writeFileSync(kafkaApplicationYamlFile.name, kafkaApplicationYaml)
// 	t.is(
// 		0,
// 		sh.exec(`kubectl apply -f ${kafkaApplicationYamlFile.name} --namespace ${defaultNamespace}`).code,
// 		'Deploying Kafka application should work.'
// 	)
// 	// 6. - poslat spravu do topicu (z navodu)
// 	// 7. - zkontrolovat ci pribudol pod
// });

test.serial('Scale application with kafka message.', t => {

  let replicaCount = '0'
  // for (let i = 0; i < 10 && replicaCount !== '0'; i++) {
  //   replicaCount = sh.exec(
  //     `kubectl get deployments/twitter-function --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
  //   ).stdout
  //   if (replicaCount !== '0') {
  //     sh.exec('sleep 1s')
  //   }
  // }
  // t.is(replicaCount, '0', 'Replica count should be 0.')

  t.is(
		0,
		sh.exec(`kubectl exec ${defaultKafkaClient} -- sh -c 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`).code,
		'Sending a fake message should work.'
	)

  for (let i = 0; i < 10 && replicaCount !== '1'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployments/twitter-function --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '1') {
      sh.exec('sleep 1s')
    }
  }
  t.is('1', replicaCount, 'Replica count should be 1 after 10 seconds.')
})

test.after.always('Clean up, delete created resources.', t => {
  const resources = [
    `${kafkaClusterYamlFile.name}`,
    `${kafkaTopicYamlFile.name}`,
    `${kafkaClientYamlFile.name}`,
    `${kafkaApplicationYamlFile.name}`,
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`kubectl delete -f 'https://strimzi.io/install/latest?namespace=${defaultNamespace}' --namespace ${defaultNamespace}`)
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)
})

const kafkaClusterYaml = `apiVersion: kafka.strimzi.io/v1beta1
kind: Kafka
metadata:
  name: ${defaultCluster}
  namespace: ${defaultNamespace}
spec:
  kafka:
    version: 2.5.0
    replicas: 1
    listeners:
      plain: {}
      tls: {}
    config:
      offsets.topic.replication.factor: 1
      transaction.state.log.replication.factor: 1
      transaction.state.log.min.isr: 1
      log.message.format.version: "2.5"
    storage:
      type: ephemeral
  zookeeper:
    replicas: 1
    storage:
      type: ephemeral
  entityOperator:
    topicOperator: {}
    userOperator: {}`

const kafkaTopicYaml = `apiVersion: kafka.strimzi.io/v1beta1
kind: KafkaTopic
metadata:
  name: ${defaultTopic}
  labels:
    strimzi.io/cluster: ${defaultCluster}
  namespace: ${defaultNamespace}
spec:
  partitions: 10
  replicas: 3
  config:
    retention.ms: 604800000
    segment.bytes: 1073741824`

const kafkaClientYaml = `apiVersion: v1
kind: Pod
metadata:
  name: ${defaultKafkaClient}
  namespace: ${defaultNamespace}
spec:
  containers:
  - name: ${defaultKafkaClient}
    image: confluentinc/cp-kafka:5.2.1
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`

const kafkaApplicationYaml = `data:
  FUNCTIONS_WORKER_RUNTIME: bm9kZQ==
  AzureWebJobsStorage: Tm9uZQ==
  POWER_BI_URL: aHR0cHM6Ly9hcGkucG93ZXJiaS5jb20vYmV0YS83MmY5ODhiZi04NmYxLTQxYWYtOTFhYi0yZDdjZDAxMWRiNDcvZGF0YXNldHMvMjVkYmRkMjAtZWM5OS00NDI2LTgyY2ItOGI3YTFlYmU2YTdlL3Jvd3M/a2V5PWRXNnpLeURYRWZMQTVycCUyRjdtNzFyaE55RU1hYVMwZUdOUm1ZOWlEMTdyUkxPbjIzOHBPWDFGWmo5M0sxWWszbzRnbW9wVmZFRng1NnNrc0tsb010clElM0QlM0Q=
apiVersion: v1
kind: Secret
metadata:
  name: twitter-function
  namespace: ${defaultNamespace}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: twitter-function
  namespace: ${defaultNamespace}
  labels:
    app: twitter-function
spec:
  selector:
    matchLabels:
      app: twitter-function
  template:
    metadata:
      labels:
        app: twitter-function
    spec:
      containers:
      - name: twitter-function
        image: jeffhollan/twitter-function
        env:
        - name: AzureFunctionsJobHost__functions__0
          value: KafkaTwitterTrigger
        envFrom:
        - secretRef:
            name: twitter-function
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: twitter-function
  namespace: ${defaultNamespace}
  labels:
    deploymentName: twitter-function
spec:
  scaleTargetRef:
    deploymentName: twitter-function
  triggers:
  - type: kafka
    metadata:
      type: kafkaTrigger
      direction: in
      name: event
      topic: ${defaultTopic}
      brokerList: ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092
      consumerGroup: functions
      dataType: binary
      lagThreshold: '5'`
