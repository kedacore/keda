import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import {createNamespace, waitForRollout} from "./helpers";

export class RabbitMQHelper {

    static installRabbit(t, username: string, password: string,
        vhost: string, rabbitmqNamespace: string) {
        const rabbitMqTmpFile = tmp.fileSync()
        fs.writeFileSync(rabbitMqTmpFile.name, rabbitmqDeployYaml.replace('{{USERNAME}}', username)
            .replace('{{PASSWORD}}', password)
            .replace('{{VHOST}}', vhost))
        createNamespace(rabbitmqNamespace)
        t.is(0, sh.exec(`kubectl apply -f ${rabbitMqTmpFile.name} --namespace ${rabbitmqNamespace}`).code, 'creating a Rabbit MQ deployment should work.')
        // wait for rabbitmq to load
        t.is(0, waitForRollout('deployment', 'rabbitmq', rabbitmqNamespace))
    }

    static uninstallRabbit(rabbitmqNamespace: string) {
        sh.exec(`kubectl delete namespace ${rabbitmqNamespace}`)
    }

    static createDeployment(t, namespace: string, deployYaml: string, amqpURI: string, scaledObjectHost: string, queueName: string) {
        const base64ConStr = Buffer.from(scaledObjectHost).toString('base64')
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, deployYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr)
            .replace('{{CONNECTION_STRING}}', amqpURI)
            .replace('{{QUEUE_NAME}}', queueName))
        createNamespace(namespace)
        t.is(
            0,
            sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${namespace}`).code,
            'creating a deployment should work.'
        )
    }

    static createVhost(t, namespace: string, host: string, username: string, password: string, vhostName: string) {
      const tmpFile = tmp.fileSync()
      fs.writeFileSync(tmpFile.name, createVhostYaml.replace('{{HOST}}', host)
          .replace('{{USERNAME_PASSWORD}}', `${username}:${password}`)
          .replace('{{VHOST_NAME}}', vhostName)
          .replace('{{VHOST_NAME}}', vhostName))
      t.is(
          0,
          sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${namespace}`).code,
          'creating a vhost should work.'
      )
  }

    static publishMessages(t, namespace: string, connectionString: string, messageCount: number, queueName: string) {
        // publish messages
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, publishYaml.replace('{{CONNECTION_STRING}}', connectionString)
        .replace('{{MESSAGE_COUNT}}', messageCount.toString())
        .replace('{{QUEUE_NAME}}',  queueName)
        .replace('{{QUEUE_NAME}}',  queueName))

        t.is(
            0,
            sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${namespace}`).code,
            'publishing job should apply.'
        )
    }

}

const publishYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-publish-{{QUEUE_NAME}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command: ["send",  "{{CONNECTION_STRING}}", "{{MESSAGE_COUNT}}", "{{QUEUE_NAME}}"]
      restartPolicy: Never`

const createVhostYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-create-vhost-{{VHOST_NAME}}
spec:
  template:
    spec:
      containers:
      - name: curl-client
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-u", "{{USERNAME_PASSWORD}}", "-X", "PUT", "http://{{HOST}}/api/vhosts/{{VHOST_NAME}}"]
      restartPolicy: Never`

const rabbitmqDeployYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: rabbitmq-config
data:
  rabbitmq.conf: |
    default_user = {{USERNAME}}
    default_pass = {{PASSWORD}}
    default_vhost = {{VHOST}}
    management.tcp.port = 15672
    management.tcp.ip = 0.0.0.0
  enabled_plugins: |
    [rabbitmq_management].
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: rabbitmq
  name: rabbitmq
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rabbitmq
  template:
    metadata:
      labels:
        app: rabbitmq
    spec:
      containers:
      - image: rabbitmq:3-management
        name: rabbitmq
        volumeMounts:
          - mountPath: /etc/rabbitmq
            name: rabbitmq-config
        readinessProbe:
          tcpSocket:
            port: 5672
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
        - name: rabbitmq-config
          configMap:
            name: rabbitmq-config
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: rabbitmq
  name: rabbitmq
spec:
  ports:
  - name: amqp
    port: 5672
    protocol: TCP
    targetPort: 5672
  - name: http
    port: 80
    protocol: TCP
    targetPort: 15672
  selector:
    app: rabbitmq`
