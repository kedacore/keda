import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

export class ArtemisHelper {
    static installArtemis(t, artemisNamespace: string) {
        const tmpFile = tmp.fileSync()
        // deploy artemis
        fs.writeFileSync(tmpFile.name, artemisYaml)
        sh.exec('kubectl create namespace artemis')
        t.is(
            0,
            sh.exec(`kubectl -n ${artemisNamespace} apply -f ${tmpFile.name}`).code, 'creating artemis deployment should work.'
        )
        t.is(
            0,
            sh.exec(`kubectl -n ${artemisNamespace} wait --for=condition=available --timeout=600s deployment/artemis-activemq`).code, 'Artemis should be available.'
        )

    }

    static installArtemisSecret(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, artemisSecretYaml)
        sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating secrets should work.'

    }

    static publishMessages(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, producerYaml)
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating artemis producer deployment should work.'
        )
    }

    static installConsumer(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, consumerYaml)
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating artemis consumer deployment should work.'
        )
    }

    static uninstallArtemis(t, artemisNamespace: string){
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, artemisYaml)
        sh.exec(`kubectl -n ${artemisNamespace} delete -f ${tmpFile.name}`)

    }

    static uninstallWorkloads(t, testNamespace: string){
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, consumerYaml)
        sh.exec(`kubectl -n ${testNamespace} delete -f ${tmpFile.name}`)
        fs.writeFileSync(tmpFile.name, producerYaml)
        sh.exec(`kubectl -n ${testNamespace} delete -f ${tmpFile.name}`)
    }
}


const artemisYaml = `---
apiVersion: v1
kind: Secret
metadata:
  name: artemis-activemq
  labels:
    app: activemq-artemis
type: Opaque
data:
  artemis-password:  "YXJ0ZW1pcw=="

---
apiVersion: v1
kind: Service
metadata:
  name: artemis-activemq
  labels:
    app: activemq-artemis
spec:
  ports:
    - name: http
      port: 8161
      targetPort: http
    - name: core
      port: 61616
      targetPort: core
    - name: amqp
      port: 5672
      targetPort: amqp
    - name: jmx
      port: 9494
      targetPort: jmxexporter

  selector:
    app: activemq-artemis

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: artemis-activemq-cm
data:
  broker-00.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>

    <configuration xmlns="urn:activemq" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:activemq /schema/artemis-configuration.xsd">

      <core xmlns="urn:activemq:core" xsi:schemaLocation="urn:activemq:core ">
         <name>artemis-activemq</name>
         <addresses>
           <address name="test">
           <anycast>
             <queue name="test"/>
           </anycast>
         </address>
        </addresses>
      </core>
    </configuration>

  configure-cluster.sh: |

    set -e
    echo Copying common configuration
    cp /data/etc-override/*.xml /var/lib/artemis/etc-override/broker-10.xml

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: artemis-activemq
  labels:
    app: activemq-artemis
spec:
  replicas: 1
  revisionHistoryLimit: 1
  selector:
    matchLabels:
      app: activemq-artemis
  template:
    metadata:
      name: artemis-activemq-artemis
      labels:
        app: activemq-artemis
    spec:
      initContainers:
        - name: configure-cluster
          image: docker.io/vromero/activemq-artemis:2.6.2
          command: ["/bin/sh", "/data/etc-override/configure-cluster.sh"]
          volumeMounts:
            - name: config-override
              mountPath: /var/lib/artemis/etc-override
            - name: configmap-override
              mountPath: /data/etc-override/
      containers:
        - name: artemis-activemq-artemis
          image: docker.io/vromero/activemq-artemis:2.6.2
          imagePullPolicy:
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
          env:
            - name: ARTEMIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: artemis-activemq
                  key: artemis-password
            - name: ARTEMIS_USERNAME
              value: "artemis"
            - name: ARTEMIS_PERF_JOURNAL
              value: "AUTO"
            - name: ENABLE_JMX_EXPORTER
              value: "true"
          ports:
            - name: http
              containerPort: 8161
            - name: core
              containerPort: 61616
            - name: amqp
              containerPort: 5672
            - name: jmxexporter
              containerPort: 9404
          livenessProbe:
            tcpSocket:
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            tcpSocket:
              port: core
            initialDelaySeconds: 10
            periodSeconds: 10
          volumeMounts:
            - name: data
              mountPath: /var/lib/artemis/data
            - name: config-override
              mountPath: /var/lib/artemis/etc-override
      volumes:
        - name: data
          emptyDir: {}
        - name: config-override
          emptyDir: {}
        - name: configmap-override
          configMap:
            name:  artemis-activemq-cm`

const artemisSecretYaml = `apiVersion: v1
kind: Secret
metadata:
  name: kedartemis
  labels:
    app: kedartemis
type: Opaque
data:
  artemis-password: "YXJ0ZW1pcw=="
  artemis-username: "YXJ0ZW1pcw=="
`

const consumerYaml = `
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kedartemis-consumer
spec:
  selector:
    matchLabels:
      app: kedartemis-consumer
  replicas: 0
  template:
    metadata:
      labels:
        app: kedartemis-consumer
    spec:
      containers:
        - name: kedartemis-consumer
          image: balchu/kedartemis-consumer
          imagePullPolicy: Always
          env:
            - name: ARTEMIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: kedartemis
                  key: artemis-password
            - name: ARTEMIS_USERNAME
              value: "artemis"
            - name: ARTEMIS_HOST
              value: "artemis-activemq.artemis"
            - name: ARTEMIS_PORT
              value: "61616"
`

const producerYaml = `
apiVersion: batch/v1
kind: Job
metadata:
  name: artemis-producer
spec:
  ttlSecondsAfterFinished: 10
  template:
    spec:
      containers:
        - name: artemis-producer
          image: balchu/artemis-producer:0.0.1
          env:
            - name: ARTEMIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: kedartemis
                  key: artemis-password
            - name: ARTEMIS_USERNAME
              value: "artemis"
            - name: ARTEMIS_SERVER_HOST
              value: "artemis-activemq.artemis"
            - name: ARTEMIS_SERVER_PORT
              value: "61616"
      restartPolicy: Never
  backoffLimit: 4
`
