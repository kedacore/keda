import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

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
        for (let i = 0; i < 10; i++) {
            const readyReplicaCount = sh.exec(`kubectl get deploy/rabbitmq -n ${rabbitmqNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
            if (readyReplicaCount != '2') {
                sh.exec('sleep 2s')
            }
        }
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

        // wait for the publishing job to complete
        for (let i = 0; i < 20; i++) {
            const succeeded = sh.exec(`kubectl get job rabbitmq-publish --namespace ${namespace} -o jsonpath='{.status.succeeded}'`).stdout
            if (succeeded == '1') {
                break
            }
            sh.exec('sleep 1s')
        }
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

const rabbitmqDeployYaml = `apiVersion: apps/v1
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
        env:
        - name: RABBITMQ_DEFAULT_USER 
          value: "{{USERNAME}}"
        - name: RABBITMQ_DEFAULT_PASS 
          value: "{{PASSWORD}}"
        - name: RABBITMQ_DEFAULT_VHOST
          value: "{{VHOST}}"
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
