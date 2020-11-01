import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

export class StanHelper {
    static install(t, stanNamespace: string) {
        const tmpFile = tmp.fileSync()
        // deploy stan
        fs.writeFileSync(tmpFile.name, stanManifest)
        sh.exec('kubectl create namespace stan')
        t.is(
            0,
            sh.exec(`kubectl -n ${stanNamespace} apply -f ${tmpFile.name}`).code, 'creating stan statefulset should work.'
        )
        t.is(
            0,
            sh.exec(`kubectl -n ${stanNamespace} wait --for=condition=Ready --timeout=600s po/stan-nats-ss-0`).code, 'Stan pod should be available.'
        )

    }

    static uninstall(t, stanNamespace: string){
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, stanManifest)
        sh.exec(`kubectl -n ${stanNamespace} delete -f ${tmpFile.name}`)

    }

    static publishMessages(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, pubYaml)
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating stan producer should work.'
        )
    }

    static installConsumer(t, testNamespace: string) {
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, subYaml)
        t.is(
            0,
            sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating stan consumer deployment should work.'
        )
    }

    static uninstallWorkloads(t, testNamespace: string){
        const tmpFile = tmp.fileSync()
        fs.writeFileSync(tmpFile.name, subYaml)
        sh.exec(`kubectl -n ${testNamespace} delete -f ${tmpFile.name}`)
        fs.writeFileSync(tmpFile.name, pubYaml)
        sh.exec(`kubectl -n ${testNamespace} delete -f ${tmpFile.name}`)
    }

}

const stanManifest = `
# Source: nats-ss/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: stan-nats-ss
  labels:
    app: nats-ss
    chart: nats-ss-0.0.1
    release: stan
    heritage: Helm
spec:
  type: ClusterIP
  ports:
    - name: client
      port: 4222
      targetPort: 4222
      protocol: TCP
    - name: monitor
      port: 8222
      targetPort: 8222
      protocol: TCP
  selector:
    app: nats-ss
    release: stan
---
# Source: nats-ss/templates/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: stan-nats-ss
  labels:
    app: nats-ss
    chart: nats-ss-0.0.1
    release: stan
    heritage: Helm
spec:
  serviceName: nats-ss
  replicas: 1
  selector:
    matchLabels:
      app: nats-ss
  template:
    metadata:
      labels:
        app: nats-ss
        release: stan
    spec:
      containers:
      - name: nats-ss
        image: nats-streaming:0.16.2
        imagePullPolicy: IfNotPresent
        command:
          - /nats-streaming-server
        args:
          - -m=8222
          - -st=FILE
          - --dir=/nats-datastore
          - --cluster_id=local-stan
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        volumeMounts:
        - mountPath: /nats-datastore
          name: nats-datastore
      volumes:
      - name: nats-datastore
        emptyDir: {}
`

const pubYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pub
  labels:
    app.kubernetes.io/name: pub
    helm.sh/chart: pub-0.0.3
    app.kubernetes.io/instance: pub
    app.kubernetes.io/version: "0.0.3"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: pub
      app.kubernetes.io/instance: pub
  template:
    metadata:
      labels:
        app.kubernetes.io/name: pub
        app.kubernetes.io/instance: pub
    spec:
      containers:
        - name: pub
          image: "balchu/gonuts-pub:c02e4ee-dirty"
          imagePullPolicy: Always
          command: ["/app"]
          args: ["-s", "nats://stan-nats-ss.stan:4222", "-d", "10", "-limit", "1000", "Test"]
`

const subYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sub
  labels:
    app.kubernetes.io/name: sub
    helm.sh/chart: sub-0.0.3
    app.kubernetes.io/instance: sub
    app.kubernetes.io/version: "0.0.3"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 0
  selector:
    matchLabels:
      app.kubernetes.io/name: sub
      app.kubernetes.io/instance: sub
  template:
    metadata:
      labels:
        app.kubernetes.io/name: sub
        app.kubernetes.io/instance: sub
    spec:
      containers:
        - name: sub
          image: "balchu/gonuts-sub:c02e4ee"
          imagePullPolicy: Always
          command: ["/app"]
          args: ["-d", "5000", "-s", "nats://stan-nats-ss.stan:4222","-d","10","--durable","ImDurable", "--qgroup", "grp1", "Test"]
`