import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import {waitForRollout} from "./helpers";

export class RabbitMQHelper {

    static installRabbit(t, username: string, password: string,
        vhost: string, rabbitmqNamespace: string) {
        const rabbitMqTmpFile = tmp.fileSync()
        fs.writeFileSync(rabbitMqTmpFile.name, rabbitmqDeployYaml.replace('{{USERNAME}}', username)
            .replace('{{PASSWORD}}', password)
            .replace('{{VHOST}}', vhost))
        sh.exec(`kubectl create namespace ${rabbitmqNamespace}`)
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
        sh.exec(`kubectl create namespace ${namespace}`)
        t.is(
            0,
            sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${namespace}`).code,
            'creating a deployment should work.'
        )
    }

    static publishMessages(t, namespace: string, connectionString: string, messageCount: number) {
        // publish messages
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, publishYaml.replace('{{CONNECTION_STRING}}', connectionString)
            .replace('{{MESSAGE_COUNT}}', messageCount.toString()))
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
  name: rabbitmq-publish
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: jeffhollan/rabbitmq-client:dev
        imagePullPolicy: Always
        command: ["send",  "{{CONNECTION_STRING}}", "{{MESSAGE_COUNT}}"]
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
