import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava';

const defaultNamespace = 'kafka-test'
const defaultCluster = 'kafka-cluster'
const timeToWait = 300
const defaultTopic = 'kafka-topic'
const defaultTopic2 = 'kafka-topic-2'
const defaultKafkaClient = 'kafka-client'
const strimziOperatorVersion = '0.18.0'
const commandToCheckReplicas = `kubectl get deployments/kafka-consumer --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`

const strimziOperatorYamlFile = tmp.fileSync()
const kafkaClusterYamlFile = tmp.fileSync()
const kafkaTopicYamlFile = tmp.fileSync()
const kafkaClientYamlFile = tmp.fileSync()
const kafkaApplicationLatestYamlFile = tmp.fileSync()
const kafkaApplicationEarliestYamlFile = tmp.fileSync()
const kafkaApplicationMultipleTopicsYamlFile = tmp.fileSync()
const scaledObjectEarliestYamlFile = tmp.fileSync()
const scaledObjectLatestYamlFile = tmp.fileSync()
const scaledObjectMultipleTopicsYamlFile = tmp.fileSync()

test.before('Set up, create necessary resources.', t => {
    sh.config.silent = true
	sh.exec(`kubectl create namespace ${defaultNamespace}`)

  const strimziOperatorYaml = sh.exec(`curl -L https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimziOperatorVersion}/strimzi-cluster-operator-${strimziOperatorVersion}.yaml`).stdout
  fs.writeFileSync(strimziOperatorYamlFile.name, strimziOperatorYaml.replace(/myproject/g, `${defaultNamespace}`))
	t.is(
		0,
		sh.exec(`kubectl apply -f ${strimziOperatorYamlFile.name} --namespace ${defaultNamespace}`).code,
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

    fs.writeFileSync(kafkaTopicYamlFile.name, kafkaTopicsYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${kafkaTopicYamlFile.name} --namespace ${defaultNamespace}`).code,
		'Deploying Kafka topic should work.'
	)
	t.is(
		0,
		sh.exec(`kubectl wait kafkatopic/${defaultTopic} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		'Kafka topic should be ready withlanguage-mattersin given time limit.'
    )
    t.is(
        0,
        sh.exec(`kubectl wait kafkatopic/${defaultTopic2} --for=condition=Ready --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
        'Kafka topic2 should be ready within given time limit.'
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

test.serial('Cleanup after latest policy test', t=> {
    t.is(
        0,
        sh.exec(`kubectl delete -f ${scaledObjectLatestYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deleting Scaled Object should work.'
    )
    t.is(
        0,
        sh.exec(`kubectl delete -f ${kafkaApplicationLatestYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deleting kafka application should work.'
    )
    sh.exec(`sleep 10s`)
})

test.serial('Applying ScaledObject with multiple topics should scale up pods', t => {
    // Make the consumer commit the all offsets for all topics in the group
    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'kafka-console-consumer --bootstrap-server "${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092" --topic ${defaultTopic}  --group multiTopic --from-beginning --consumer-property enable.auto.commit=true --timeout-ms 15000'`)
    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'kafka-console-consumer --bootstrap-server "${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092" --topic ${defaultTopic2} --group multiTopic --from-beginning --consumer-property enable.auto.commit=true --timeout-ms 15000'`)

    fs.writeFileSync(kafkaApplicationMultipleTopicsYamlFile.name, kafkaApplicationMultipleTopicsYaml)
    t.is(
        0,
        sh.exec(`kubectl apply -f ${kafkaApplicationMultipleTopicsYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deploying Kafka application should work.'
    )
    sh.exec(`sleep 5s`)
    fs.writeFileSync(scaledObjectMultipleTopicsYamlFile.name, scaledObjectMultipleTopicsYaml)

    t.is(
        0,
        sh.exec(`kubectl apply -f ${scaledObjectMultipleTopicsYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deploying Scaled Object should work.'
    )
    sh.exec(`sleep 5s`)

    // when lag is 0, scaled object is not active, replica = 0
    waitForReplicaCount(0, commandToCheckReplicas)
    t.is('0', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 0.')

    // produce a single msg to the default topic
    // should turn scale object active, replica = 1
    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -exc 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`)
    sh.exec(`sleep 5s`)
    waitForReplicaCount(1, commandToCheckReplicas)
    t.is('1', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 1.')

    // produce one more msg to the different topic within the same group
    // will turn total consumer group lag to 2.
    // with lagThreshold as 1 -> making hpa AverageValue to 1
    // this should turn nb of replicas to 2
    // as desiredReplicaCount = totalLag / avgThreshold
    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -exc 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic2}'`)
    sh.exec(`sleep 5s`)
    waitForReplicaCount(2, commandToCheckReplicas)
    t.is('2', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 2.')

    // make it 3 cause why not?
    sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -exc 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic ${defaultTopic}'`)
    sh.exec(`sleep 5s`)
    waitForReplicaCount(3, commandToCheckReplicas)
    t.is('3', sh.exec(commandToCheckReplicas).stdout, 'Replica count should be 3.')
})

test.serial('Cleanup after multiple topics test', t=> {
    t.is(
        0,
        sh.exec(`kubectl delete -f ${scaledObjectMultipleTopicsYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deleting Scaled Object should work.'
    )
    t.is(
        0,
        sh.exec(`kubectl delete -f ${kafkaApplicationMultipleTopicsYamlFile.name} --namespace ${defaultNamespace}`).code,
        'Deleting kafka application should work.'
    )
})


test.after.always('Clean up, delete created resources.', t => {
  const resources = [
    `${scaledObjectEarliestYamlFile.name}`,
    `${scaledObjectLatestYamlFile.name}`,
    `${scaledObjectMultipleTopicsYamlFile.name}`,

    `${kafkaApplicationEarliestYamlFile.name}`,
    `${kafkaApplicationLatestYamlFile.name}`,
    `${kafkaApplicationMultipleTopicsYamlFile.name}`,

    `${kafkaClientYamlFile.name}`,
    `${kafkaTopicYamlFile.name}`,
    `${kafkaClusterYamlFile.name}`,
    `${strimziOperatorYamlFile}`
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
    replicas: 3
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

const kafkaTopicsYaml = `apiVersion: kafka.strimzi.io/v1beta1
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
    segment.bytes: 1073741824
---
apiVersion: kafka.strimzi.io/v1beta1
kind: KafkaTopic
metadata:
  name: ${defaultTopic2}
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

const kafkaApplicationMultipleTopicsYaml = `
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
      # only recent version of kafka-console-consumer support flag "include"
      # old version's equiv flag will violate language-matters commit hook
      # work around -> create two consumer container joining the same group
      - name: kafka-consumer
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic '${defaultTopic}'  --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"
      - name: kafka-consumer-2
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092 --topic '${defaultTopic2}' --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"`

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

const scaledObjectMultipleTopicsYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-consumer-multi-topic
  namespace: ${defaultNamespace}
spec:
  scaleTargetRef:
    name: kafka-consumer
  triggers:
  - type: kafka
    metadata:
      bootstrapServers: ${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092
      consumerGroup: multiTopic
      lagThreshold: '1'
      offsetResetPolicy: 'latest'`
