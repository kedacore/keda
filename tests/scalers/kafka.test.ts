import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test, { Assertions } from 'ava';
import { createNamespace, waitForDeploymentReplicaCount } from './helpers';

const defaultNamespace = 'kafka-test'
const defaultCluster = 'kafka-cluster'
const timeToWait = 300
const defaultTopic = 'kafka-topic'
const defaultTopic2 = 'kafka-topic-2'
const zeroInvalidOffsetTopic = 'kafka-topic-zero-invalid-offset'
const oneInvalidOffsetTopic = 'kafka-topic-one-invalid-offset'
const invalidOffsetGroup = 'invalidOffset'
const defaultKafkaClient = 'kafka-client'
const strimziOperatorVersion = '0.23.0'
const bootstrapServer = `${defaultCluster}-kafka-bootstrap.${defaultNamespace}:9092`

const strimziOperatorYamlFile = tmp.fileSync()
const kafkaClusterYamlFile = tmp.fileSync()
const kafkaTopicYamlFile = tmp.fileSync()
const kafkaClientYamlFile = tmp.fileSync()
const kafkaApplicationYamlFile = tmp.fileSync()
const scaledObjectYamlFile = tmp.fileSync()

function deployFromYaml(t: Assertions, filename: string, yaml: string, name: string) {
  sh.exec(`echo Deploying ${name}`)
  fs.writeFileSync(filename, yaml)
	t.is(0, sh.exec(`kubectl apply -f ${filename} --namespace ${defaultNamespace}`).code, `Deploying ${name} should work.`)
}

function waitForReady(t: Assertions, app: string, name: string, condition: string = 'Ready') {
  sh.exec(`echo Waiting for ${app} for ${timeToWait} seconds to be ${condition}`)
  t.is(
		0,
		sh.exec(`kubectl wait ${app} --for=condition=${condition} --timeout=${timeToWait}s --namespace ${defaultNamespace}`).code,
		`${name} should be ready within given time limit.`
    )
}

function commitPartition(topic: string, group: string) {
  sh.exec(`echo Committing partition for ${topic}:${group}`)
  return sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -c 'kafka-console-consumer --bootstrap-server "${bootstrapServer}" --topic ${topic} --group ${group} --from-beginning --consumer-property enable.auto.commit=true --timeout-ms 15000'`).code == 0
}

function publishMessage(topic: string) {
  sh.exec(`kubectl exec --namespace ${defaultNamespace} ${defaultKafkaClient} -- sh -exc 'echo "{\"text\": \"foo\"}" | kafka-console-producer --broker-list ${bootstrapServer} --topic ${topic}'`)
  sh.exec(`sleep 5s`)
}

function cleanup(t: Assertions) {
  t.is(
    0,
    sh.exec(`kubectl delete -f ${scaledObjectYamlFile.name} --namespace ${defaultNamespace}`).code,
    'Deleting Scaled Object should work.'
  )
  t.is(
      0,
      sh.exec(`kubectl delete -f ${kafkaApplicationYamlFile.name} --namespace ${defaultNamespace}`).code,
      'Deleting kafka application should work.'
  )
  sh.exec(`sleep 10s`)
}

test.before('Set up, create necessary resources.', async t => {
	createNamespace(defaultNamespace)

  sh.config.silent = true
  const strimziOperatorYaml = sh.exec(`curl -L https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimziOperatorVersion}/strimzi-cluster-operator-${strimziOperatorVersion}.yaml`).stdout
  sh.config.silent = false

  deployFromYaml(t, strimziOperatorYamlFile.name, strimziOperatorYaml.replace(/myproject/g, `${defaultNamespace}`), 'Strimzi operator')
  deployFromYaml(t, kafkaClusterYamlFile.name, kafkaClusterYaml, 'Kafka cluster')
  waitForReady(t, `kafka/${defaultCluster}`,'Kafka instance')

  var kafkaTopicsYaml =
    kafkaTopicsTemplateYaml.replace('{{TOPIC}}', defaultTopic).replace('{{PARTITIONS}}', '3') +
    kafkaTopicsTemplateYaml.replace('{{TOPIC}}', defaultTopic2).replace('{{PARTITIONS}}', '3') +
    kafkaTopicsTemplateYaml.replace('{{TOPIC}}', zeroInvalidOffsetTopic).replace('{{PARTITIONS}}', '1') +
    kafkaTopicsTemplateYaml.replace('{{TOPIC}}', oneInvalidOffsetTopic).replace('{{PARTITIONS}}', '1')
  deployFromYaml(t, kafkaTopicYamlFile.name, kafkaTopicsYaml, 'Kafka topic')
  waitForReady(t, `kafkatopic/${defaultTopic}`,defaultTopic)
  waitForReady(t, `kafkatopic/${defaultTopic2}`,defaultTopic2)
  waitForReady(t, `kafkatopic/${zeroInvalidOffsetTopic}`,zeroInvalidOffsetTopic)
  waitForReady(t, `kafkatopic/${oneInvalidOffsetTopic}`,oneInvalidOffsetTopic)

  deployFromYaml(t, kafkaClientYamlFile.name, kafkaClientYaml, 'Kafka client')
  waitForReady(t, `pod/${defaultKafkaClient}`,'Kafka client')
});

test.serial('Applying ScaledObject earliest policy should not scale up pods', async t => {

  deployFromYaml(t, kafkaApplicationYamlFile.name, kafkaApplicationEarliestYaml, 'Kafka application')
  deployFromYaml(t, scaledObjectYamlFile.name, scaledObjectEarliestYaml, 'Scaled Object')
  waitForReady(t, 'deployment/kafka-consumer','Kafka application', 'Available')

  t.true(await waitForDeploymentReplicaCount(0, 'kafka-consumer', defaultNamespace, 30, 2000), 'replica count should start out as 0')
});

test.serial('Scale application with kafka messages.', async t => {
  for (let r = 1; r <= 3; r++) {
    publishMessage(defaultTopic)
    t.true(await waitForDeploymentReplicaCount(r, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be ${r}.`)
  }
})

test.serial('Scale application beyond partition max.', async t => {
  publishMessage(defaultTopic)
  t.true(await waitForDeploymentReplicaCount(3, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 3.`)
})

test.serial('cleanup after earliest policy test', t => cleanup(t))

test.serial('Applying ScaledObject latest policy should not scale up pods', async t => {

  //Make the consumer commit the first offset for each partition.
  t.true(commitPartition(defaultTopic, 'latest'), 'Commit partition should work')

  deployFromYaml(t, kafkaApplicationYamlFile.name, kafkaApplicationLatestYaml, 'Kafka application')
  sh.exec(`sleep 10s`)
  deployFromYaml(t, scaledObjectYamlFile.name, scaledObjectLatestYaml, 'Scaled Object')
  sh.exec(`sleep 5s`)
  t.true(await waitForDeploymentReplicaCount(0, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 0.`)
})

test.serial('Latest Scale object should scale with new messages', async t => {

  for (let r = 1; r <= 3; r++) {
    publishMessage(defaultTopic)
    t.true(await waitForDeploymentReplicaCount(r, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be ${r}.`)
  }
})

test.serial('Cleanup after latest policy test', t => cleanup(t))

test.serial('Applying ScaledObject with multiple topics should scale up pods', async t => {
    // Make the consumer commit the all offsets for all topics in the group
    t.true(commitPartition(defaultTopic, 'multiTopic'), 'Commit partition should work')
    t.true(commitPartition(defaultTopic2, 'multiTopic'), 'Commit partition should work')

    deployFromYaml(t, kafkaApplicationYamlFile.name, kafkaApplicationMultipleTopicsYaml, 'Kafka application')
    sh.exec(`sleep 5s`)
    deployFromYaml(t, scaledObjectYamlFile.name, scaledObjectMultipleTopicsYaml, 'Scaled Object')
    sh.exec(`sleep 5s`)

    // when lag is 0, scaled object is not active, replica = 0
    t.true(await waitForDeploymentReplicaCount(0, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 0.`)

    // produce a single msg to the default topic
    // should turn scale object active, replica = 1
    publishMessage(defaultTopic)
    t.true(await waitForDeploymentReplicaCount(1, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 1.`)

    // produce one more msg to the different topic within the same group
    // will turn total consumer group lag to 2.
    // with lagThreshold as 1 -> making hpa AverageValue to 1
    // this should turn nb of replicas to 2
    // as desiredReplicaCount = totalLag / avgThreshold
    publishMessage(defaultTopic2)
    t.true(await waitForDeploymentReplicaCount(2, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 2.`)

    // make it 3 cause why not?
    publishMessage(defaultTopic)
    t.true(await waitForDeploymentReplicaCount(3, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 3.`)
})

test.serial('Cleanup after multiple topics test', t => cleanup(t))

test.serial('Applying ScaledObject zeroOnInvalidOffset policy should not scale up pods', async t => {

  deployFromYaml(t, kafkaApplicationYamlFile.name, kafkaApplicationZeroOnInvalidYaml, 'Kafka application')
  deployFromYaml(t, scaledObjectYamlFile.name, scaledObjectZeroOnInvalidOffsetYaml, 'Scaled Object')
  waitForReady(t, 'deployment/kafka-consumer','Kafka application', 'Available')
  sh.exec(`sleep 30s`)
  t.true(await waitForDeploymentReplicaCount(0, 'kafka-consumer', defaultNamespace, 30, 3000), `Replica count should be 0.`)
})

test.serial('cleanup after zeroOnInvalidOffset policy test', t => cleanup(t))

test.serial('Applying ScaledObject oneOnInvalidOffset policy should scale to one pod', async t => {
  deployFromYaml(t, kafkaApplicationYamlFile.name, kafkaApplicationOneOnInvalidYaml, 'Kafka application')
  deployFromYaml(t, scaledObjectYamlFile.name, scaledObjectOneOnInvalidOffsetYaml, 'Scaled Object')
  waitForReady(t, 'deployment/kafka-consumer','Kafka application', 'Available')
  sh.exec(`sleep 30s`)
  t.true(await waitForDeploymentReplicaCount(1, 'kafka-consumer', defaultNamespace, 30, 2000), `Replica count should be 1.`)
})

test.serial('oneOnInvalidOffset Scale object should scale to zero when offset is set', async t => {
  t.true(commitPartition(oneInvalidOffsetTopic, invalidOffsetGroup), 'Commit partition should work')
  publishMessage(oneInvalidOffsetTopic)
  t.true(await waitForDeploymentReplicaCount(0, 'kafka-consumer', defaultNamespace, 60, 10000), `Replica count should scale down to 0.`)
})

test.serial('cleanup after oneOnInvalidOffset policy test', t => cleanup(t))


test.after.always('Clean up, delete created resources.', t => {
  const resources = [
    `${kafkaClientYamlFile.name}`,
    `${kafkaTopicYamlFile.name}`,
    `${kafkaClusterYamlFile.name}`,
    `${strimziOperatorYamlFile.name}`
  ]

  for (const resource of resources) {
    sh.exec(`echo Deleting resource from file ${resource}`)
    sh.exec(`kubectl delete -f ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`echo Deleting namespace ${defaultNamespace}`)
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)
})

const kafkaClusterYaml = `apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: ${defaultCluster}
  namespace: ${defaultNamespace}
spec:
  kafka:
    version: "2.6.0"
    replicas: 1
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: tls
        port: 9093
        type: internal
        tls: true
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

const kafkaTopicsTemplateYaml = `apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: {{TOPIC}}
  labels:
    strimzi.io/cluster: ${defaultCluster}
  namespace: ${defaultNamespace}
spec:
  partitions: {{PARTITIONS}}
  replicas: 1
  config:
    retention.ms: 604800000
    segment.bytes: 1073741824
---
`

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

const kafkaApplicationTemplateYaml = `
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
          - "kafka-console-consumer --bootstrap-server ${bootstrapServer} PARAMS --consumer-property enable.auto.commit=COMMIT"`

const kafkaApplicationEarliestYaml = kafkaApplicationTemplateYaml.replace(/PARAMS/g, `--topic ${defaultTopic} --group earliest --from-beginning`).replace(/COMMIT/g, 'false')
const kafkaApplicationLatestYaml = kafkaApplicationTemplateYaml.replace(/PARAMS/g, `--topic ${defaultTopic} --group latest`).replace(/COMMIT/g, 'false')
const kafkaApplicationZeroOnInvalidYaml = kafkaApplicationTemplateYaml.replace(/PARAMS/g, `--topic ${zeroInvalidOffsetTopic} --group ${invalidOffsetGroup}`).replace(/COMMIT/g, 'true')
const kafkaApplicationOneOnInvalidYaml = kafkaApplicationTemplateYaml.replace(/PARAMS/g, `--topic ${oneInvalidOffsetTopic} --group ${invalidOffsetGroup} --from-beginning`).replace(/COMMIT/g, 'true')

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
          - "kafka-console-consumer --bootstrap-server ${bootstrapServer} --topic '${defaultTopic}'  --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"
      - name: kafka-consumer-2
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server ${bootstrapServer} --topic '${defaultTopic2}' --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"`

const scaledObjectTemplateYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-consumer-GROUP
  namespace: ${defaultNamespace}
spec:
  scaleTargetRef:
    name: kafka-consumer
  triggers:
  - type: kafka
    metadata:
      topic: ${defaultTopic}
      bootstrapServers: ${bootstrapServer}
      consumerGroup: GROUP
      lagThreshold: '1'
      offsetResetPolicy: 'GROUP'`

const scaledObjectEarliestYaml = scaledObjectTemplateYaml.replace(/GROUP/g, 'earliest')
const scaledObjectLatestYaml = scaledObjectTemplateYaml.replace(/GROUP/g, 'latest')

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
      bootstrapServers: ${bootstrapServer}
      consumerGroup: multiTopic
      lagThreshold: '1'
      offsetResetPolicy: 'latest'`

const scaledObjectInvalidOffsetTemplateYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-consumer-on-invalid
  namespace: ${defaultNamespace}
spec:
  scaleTargetRef:
    name: kafka-consumer
  triggers:
  - type: kafka
    metadata:
      topic: TOPIC
      bootstrapServers: ${bootstrapServer}
      consumerGroup: ${invalidOffsetGroup}
      lagThreshold: '1'
      scaleToZeroOnInvalidOffset: 'VALUE'
      offsetResetPolicy: 'latest'`

const scaledObjectZeroOnInvalidOffsetYaml = scaledObjectInvalidOffsetTemplateYaml.replace(/TOPIC/g, zeroInvalidOffsetTopic).replace(/VALUE/g, 'true')
const scaledObjectOneOnInvalidOffsetYaml = scaledObjectInvalidOffsetTemplateYaml.replace(/TOPIC/g, oneInvalidOffsetTopic).replace(/VALUE/g, 'false')
