import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava';

const defaultNamespace = 'kafka-test'
const defaultCluster = 'kafka-cluster'
const timeToWait = 300
const defaultTopic = 'kafka-topic'
const defaultKafkaClient = 'kafka-client'
const strimziOperatorVersion = '0.18.0'
const commandToCheckReplicas = `kubectl get deployments/kafka-consumer --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`

const strimziOperatroYamlFile = tmp.fileSync()
const kafkaClusterYamlFile = tmp.fileSync()
const kafkaTopicYamlFile = tmp.fileSync()
const kafkaClientYamlFile = tmp.fileSync()
const kafkaApplicationLatestYamlFile = tmp.fileSync()
const kafkaApplicationEarliestYamlFile = tmp.fileSync()
const scaledObjectEarliestYamlFile = tmp.fileSync()
const scaledObjectLatestYamlFile = tmp.fileSync()

test.before('Set up, create necessary resources.', t => {
	sh.config.silent = true
	sh.exec(`kubectl create namespace ${defaultNamespace}`)

  const strimziOperatorYaml = sh.exec(`curl -L https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimziOperatorVersion}/strimzi-cluster-operator-${strimziOperatorVersion}.yaml`).stdout
  fs.writeFileSync(strimziOperatroYamlFile.name, strimziOperatorYaml.replace(/myproject/g, `${defaultNamespace}`))
	t.is(
		0,
		sh.exec(`kubectl apply -f ${strimziOperatroYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Strimzi operator should work.'
	)

	fs.writeFileSync(kafkaClusterYamlFile.name, kafkaClusterYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaClusterYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka cluster instance should work.'
	)
	t.is(
		0,
		sh.exec(`kubectl wait kafka/${defaultCluster} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		'Kafka instance should be ready within given time limit.'
  )

	fs.writeFileSync(kafkaTopicYamlFile.name, kafkaTopicYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaTopicYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka topic should work.'
	)
	t.is(
		0,
		sh.exec(`kubectl wait kafkatopic/${defaultTopic} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		'Kafka topic should be ready within given time limit.'
  )

	fs.writeFileSync(kafkaClientYamlFile.name, kafkaClientYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaClientYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka client should work.'
	)
	t.is(
		0,
		sh.exec(`kubectl wait pod/${defaultKafkaClient} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		'Kafka client should be ready within given time limit.'
  )

  fs.writeFileSync(kafkaApplicationEarliestYamlFile.name, kafkaApplicationEarliestYaml)

	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaApplicationEarliestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka application should work.'
  )
  fs.writeFileSync(scaledObjectEarliestYamlFile.name, scaledObjectEarliestYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${scaledObjectEarliestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Scaled Object should work.'
	)
	t.is(
		0,
		sh.exec(`kubectl wait deployment/kafka-consumer --for=condition=Available --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		'Kafka application should be ready within given time limit.'
	)
  waitForReplicaCount(0, commandToCheckReplicas)
  t.is('0', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 0.')
});

function waitForReplicaCount(desiredReplicaCount: number, commandToCheck: string) {
  let replicaCount = undefined
  let changed = undefined
  for (let i = 0; i < 10; i++) {
    changed = false
    // checks the replica count 3 times, it tends to fluctuate from the beginning
    for (let j = 0; j < 3; j++) {
      replicaCount = sh.exec(commandToCheck).stdout
      if (replicaCount === desiredReplicaCount.toString()) {
        sh.exec('sleep 2s')
      } else {
        changed = true
        break
      }
    }
    if (changed === false) {
      return
    } else {
      sh.exec('sleep 3s')
    }
  }
}

test.serial('Scale application with kafka messages.', t => {
  for (let r = 1; r <= 3; r++) {

    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`)
    sh.exec(`sleep 5s`)

    waitForReplicaCount(r, commandToCheckReplicas)

    t.is(r.toString(), sh.exec(commandToCheckReplicas).stdout, `Replica count should be ${r}.`)
  }
})

test.serial('Scale application beyond partition max.', t => {
  sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`)
  sh.exec(`sleep 5s`)

  waitForReplicaCount(3, commandToCheckReplicas)

  t.is('3', sh.exec(commandToCheckReplicas).stdout, `Replica count should be 3.`)
})

test.serial('cleanup after earliest policy test', t=> {
  t.is(
		0,
		sh.exec(`kubectl delete -f ${scaledObjectEarliestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deleting Scaled Object should work.'
  )
  t.is(
		0,
		sh.exec(`kubectl delete -f ${kafkaApplicationEarliestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deleting kafka application should work.'
  )

  sh.exec(`sleep 30s`)
})

test.serial('Applying ScaledObject latest policy should not scale up pods', t => {

  //Make the consumer commit the first offset for each partition.
  sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'kafka-console-consumer --bootstrap-server ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic} --group latest --from-beginning --consumer-property enable.auto.commit=true --timeout-ms 15000'`)

  fs.writeFileSync(kafkaApplicationLatestYamlFile.name, kafkaApplicationLatestYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaApplicationLatestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka application should work.'
  )
  sh.exec(`sleep 10s`)
  fs.writeFileSync(scaledObjectLatestYamlFile.name, scaledObjectLatestYaml)
  t.is(
		0,
		sh.exec(`kubectl apply -f ${scaledObjectLatestYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Scaled Object should work.'
  )
  sh.exec(`sleep 5s`)
  waitForReplicaCount(1, commandToCheckReplicas)
  t.is('0', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 0.')

})


test.serial('Latest Scale object should scale with new messages', t => {

  for (let r = 1; r <= 3; r++) {

    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`)
    sh.exec(`sleep 5s`)

    waitForReplicaCount(r, commandToCheckReplicas)

    t.is(r.toString(), sh.exec(commandToCheckReplicas).stdout, `Replica count should be ${r}.`)
  }
})

test.after.always('Clean up, delete created resources.', t => {
  const resources = [
    `${scaledObjectEarliestYamlFile.name}`,
    `${scaledObjectLatestYamlFile.name}`,
    `${kafkaApplicationLatestYamlFile.name}`,
    `${kafkaClientYamlFile.name}`,
    `${kafkaTopicYamlFile.name}`,
    `${kafkaClusterYamlFile.name}`,
    `${strimziOperatroYamlFile}`
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
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
  partitions: 3
  replicas: 1
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

const kafkaApplicationLatestYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kafka-consumer
  namespace: ${defaultNamespace}
  labels:
    app: kafka-consumer
spec:
  selector:
    matchLabels:
      app: kafka-consumer
  template:
    metadata:
      labels:
        app: kafka-consumer
    spec:
      containers:
      - name: kafka-consumer
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic} --group latest --consumer-property enable.auto.commit=false"`


const kafkaApplicationEarliestYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kafka-consumer
  namespace: ${defaultNamespace}
  labels:
    app: kafka-consumer
spec:
  selector:
    matchLabels:
      app: kafka-consumer
  template:
    metadata:
      labels:
        app: kafka-consumer
    spec:
      containers:
      - name: kafka-consumer
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic} --group earliest --from-beginning --consumer-property enable.auto.commit=false"`

const scaledObjectEarliestYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-consumer-earliest
  namespace: ${defaultNamespace}
spec:
  scaleTargetRef:
    name: kafka-consumer
  triggers:
  - type: kafka
    metadata:
      topic: ${defaultTopic}
      bootstrapServers: ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092
      consumerGroup: earliest
      lagThreshold: '1'
      offsetResetPolicy: 'earliest'`

const scaledObjectLatestYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-consumer-latest
  namespace: ${defaultNamespace}
spec:
  scaleTargetRef:
    name: kafka-consumer
  triggers:
  - type: kafka
    metadata:
      topic: ${defaultTopic}
      bootstrapServers: ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092
      consumerGroup: latest
      lagThreshold: '1'
      offsetResetPolicy: 'latest'`
