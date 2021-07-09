import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

export class SolaceHelper {

    static getUpdateSolaceHelmChart(t) {
        t.is(
            0,
            sh.exec(`helm repo add solacecharts https://solaceproducts.github.io/pubsubplus-kubernetes-quickstart/helm-charts`).code,
            'Should retrieve Solace Helm Chart from Repo'
        )
        t.is(
            0,
            sh.exec(`helm repo update`).code, 
            'Should update Helm Charts'
        )
    }

    static installSolaceBroker(t, testNamespace: string) {
        t.is(
          0,
          sh.exec(`kubectl create namespace ${testNamespace}`).code, 'Should create solace namespace'
        )  
        t.is(
            0,
            sh.exec(`helm install kedalab solacecharts/pubsubplus-dev --namespace ${testNamespace} --set solace.usernameAdminPassword=KedaLabAdminPwd1 --set storage.persistent=false`).code, 'Solace Broker should install'
        )
        sh.exec('sleep 2s')
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} wait --for=condition=Ready --timeout=120s pod/kedalab-pubsubplus-dev-0`).code, 'Solace should be available.'
        )
        sh.exec('sleep 2s')
    }

    static installSolaceTestHelper(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceTestHelperYaml)
        t.is(
            0,
            sh.exec(`kubectl apply -f ${tmpFile.name}`).code, 'creating test helper pod should work'
        )
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} wait --for=condition=Ready --timeout=120s pod/kedalab-helper`).code, 'kedalab-helper should be available'
        )
        sh.exec('sleep 5s')
    }

    static configSolacePubSubBroker(t, testNamespace: string) {
        t.is(
            0,
            sh.exec(`kubectl exec -n ${testNamespace} kedalab-helper -- ./config/config_solace.sh`).code, 'should be able to configure Solace PubSub Broker'
        )
    }

    static installSolaceConsumer(t) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceConsumerYaml)
        t.is(
            0,
            sh.exec(`kubectl apply -f ${tmpFile.name}`).code, 'create solace-consumer deployment should work.'
        )
        sh.exec('sleep 10s')
    }

    static installSolaceKedaSecret(t) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceKedaSecretYaml)
        sh.exec(`kubectl apply -f ${tmpFile.name}`).code, 'creating secret should work.'
    }

    static installSolaceKedaTriggerAuth(t) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceKedaTriggerAuthYaml)
        sh.exec(`kubectl apply -f ${tmpFile.name}`).code, 'creating scaled object should work.'
    }

    static installSolaceKedaScaledObject(t) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceKedaScaledObjectYaml)
        sh.exec(`kubectl apply -f ${tmpFile.name}`).code, 'creating scaled object should work.'
    }

    static publishMessages(t, testNamespace: string, messageRate: string, messageNumber: string) {
        t.is(
            0,
            sh.exec(`kubectl exec -n ${testNamespace} kedalab-helper -- ./sdkperf/sdkperf_java.sh -cip=kedalab-pubsubplus-dev:55555 -cu consumer_user@keda_vpn -cp=consumer_pwd -mr ${messageRate} -mn ${messageNumber} -mt=persistent -pql=SCALED_CONSUMER_QUEUE1`).code, 'creating solace producer deployment should work.'
        )
    }

    static uninstallSolaceKedaObjects(t){
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceKedaScaledObjectYaml)
        sh.exec(`kubectl delete -f ${tmpFile.name}`)
        fs.writeFileSync(tmpFile.name, solaceKedaTriggerAuthYaml)
        sh.exec(`kubectl delete -f ${tmpFile.name}`)
        fs.writeFileSync(tmpFile.name, solaceKedaSecretYaml)
        sh.exec(`kubectl delete -f ${tmpFile.name}`)
    }

    static uninstallSolaceTestPods(t) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, solaceConsumerYaml)
        sh.exec(`kubectl delete -f  ${tmpFile.name}`)
        fs.writeFileSync(tmpFile.name, solaceTestHelperYaml)
        sh.exec(`kubectl delete -f  ${tmpFile.name}`)
    }

    static uninstallSolace(t, solaceNamespace: string){
        sh.exec(`helm uninstall kedalab --namespace=${solaceNamespace}`)
        sh.exec(`sleep 6s`)
        sh.exec(`kubectl delete namespace ${solaceNamespace}`)
    }
}

const solaceKedaScaledObjectYaml = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name:      kedalab-scaled-object
  namespace: solace
spec:
  scaleTargetRef:
    apiVersion:    apps/v1
    kind:          Deployment
    name:          solace-consumer
  pollingInterval:  5
  cooldownPeriod:  20
  minReplicaCount:  0
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Percent
            value:         100
            periodSeconds: 10
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Pods
            value:         10
            periodSeconds: 10
          selectPolicy:    Max  
  triggers:
  - type: solace-queue
    metadata:
      brokerBaseUrl:       http://kedalab-pubsubplus-dev.solace.svc.cluster.local:8080
      msgVpn:              keda_vpn
      queueName:           SCALED_CONSUMER_QUEUE1
      msgCountTarget:      '20'
      msgSpoolUsageTarget: '100000'
    authenticationRef: 
      name: kedalab-trigger-auth
`

const solaceKedaTriggerAuthYaml = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: kedalab-trigger-auth
  namespace: solace
spec:
  secretTargetRef:
    - parameter:   username
      name:        kedalab-solace-secret
      key:         SEMP_USER
    - parameter:   password
      name:        kedalab-solace-secret
      key:         SEMP_PASSWORD
`

const solaceKedaSecretYaml = `
apiVersion: v1
kind: Secret
metadata:
  name:      kedalab-solace-secret
  namespace: solace
  labels:
    app: solace-consumer
type: Opaque
data:
  SEMP_USER:         YWRtaW4=
  SEMP_PASSWORD:     S2VkYUxhYkFkbWluUHdkMQ==
`

const solaceConsumerYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: solace-consumer
  namespace: solace
spec:
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
      name: docker-test-pod
    spec:
      containers:
      - name: solace-jms-consumer
        image: ghcr.io/solacelabs/kedalab-consumer:latest
        env:
        - name:  SOLACE_CLIENT_HOST
          value: tcp://kedalab-pubsubplus-dev:55555
        - name:  SOLACE_CLIENT_MSGVPN
          value: keda_vpn
        - name:  SOLACE_CLIENT_USERNAME
          value: consumer_user
        - name:  SOLACE_CLIENT_PASSWORD
          value: consumer_pwd
        - name:  SOLACE_CLIENT_QUEUENAME
          value: SCALED_CONSUMER_QUEUE1
        - name:  SOLACE_CLIENT_CONSUMER_DELAY
          value: '1000'
        imagePullPolicy: Always
      restartPolicy: Always
`

const solaceTestHelperYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: kedalab-helper
  namespace: solace
spec:
  containers:
  - name: sdk-perf
    image: ghcr.io/solacelabs/kedalab-helper:latest
    # Just spin & wait forever
    command: [ "/bin/bash", "-c", "--" ]
    args: [ "while true; do sleep 10; done;" ]
`
