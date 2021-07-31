import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

const seleniumGridNamespace = 'selenium-grid';
const seleniumGridHostName = `selenium-hub.${seleniumGridNamespace}`;
const seleniumGridPort = "4444";
const seleniumGridGraphQLUrl = `http://${seleniumGridHostName}:${seleniumGridPort}/graphql`;
const seleniumGridTestName = 'selenium-random-tests';

test.before(t => {
  sh.exec(`kubectl create namespace ${seleniumGridNamespace}`);

  const seleniumGridDeployTmpFile = tmp.fileSync();
  fs.writeFileSync(seleniumGridDeployTmpFile.name, seleniumGridYaml.replace(/{{NAMESPACE}}/g, seleniumGridNamespace));

  t.is(0, sh.exec(`kubectl apply --namespace ${seleniumGridNamespace} -f ${seleniumGridDeployTmpFile.name}`).code, 'creating a Selenium Grid deployment should work.')

  let seleniumHubReplicaCount = '0';

  for (let i = 0; i < 30; i++) {
    seleniumHubReplicaCount = sh.exec(`kubectl get deploy/selenium-hub -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    if (seleniumHubReplicaCount == '1') {
      break;
    }
    console.log('Waiting for selenium hub to be ready');
    sh.exec('sleep 2s')
  }
  t.is('1', seleniumHubReplicaCount, 'Selenium Hub is not in a ready state')
});

test.serial('should have one node for chrome and firefox each at start', t => {
  let seleniumChromeNodeReplicaCount = '0';
  let seleniumFireFoxReplicaCount = '0';
  for (let i = 0; i < 30; i++) {
    seleniumChromeNodeReplicaCount = sh.exec(`kubectl get deploy/selenium-chrome-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    seleniumFireFoxReplicaCount = sh.exec(`kubectl get deploy/selenium-firefox-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    if (seleniumChromeNodeReplicaCount == '1' && seleniumFireFoxReplicaCount == '1') {
      break;
    }
    console.log('Waiting for chrome and firefox node to be ready');
    sh.exec('sleep 2s')
  }

  t.is('1', seleniumChromeNodeReplicaCount, 'Selenium Chrome Node did not scale up to 1 pods')
  t.is('1', seleniumFireFoxReplicaCount, 'Selenium Firefox Node did not scale up to 1 pods')
});

test.serial('should scale down browser nodes to 0', t => {
  const scaledObjectDeployTmpFile = tmp.fileSync();
  fs.writeFileSync(scaledObjectDeployTmpFile.name, scaledObjectYaml.replace(/{{NAMESPACE}}/g, seleniumGridNamespace).replace(/{{SELENIUM_GRID_GRAPHQL_URL}}/g, seleniumGridGraphQLUrl));

  t.is(0, sh.exec(`kubectl apply --namespace ${seleniumGridNamespace} -f ${scaledObjectDeployTmpFile.name}`).code, 'creating a Scaled Object CRD should work.')

  let seleniumChromeNodeReplicaCount = '1';
  let seleniumFireFoxReplicaCount = '1';
  for (let i = 0; i < 60; i++) {
    seleniumChromeNodeReplicaCount = sh.exec(`kubectl get deploy/selenium-chrome-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    seleniumFireFoxReplicaCount = sh.exec(`kubectl get deploy/selenium-firefox-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    if (seleniumChromeNodeReplicaCount == '0' && seleniumFireFoxReplicaCount == '0') {
      break;
    }
    console.log('Waiting for chrome and firefox to scale down to 0 pods')
    sh.exec('sleep 5s')
  }

  t.is('0', seleniumChromeNodeReplicaCount, 'Selenium Chrome Node did not scale down to 0 pods')
  t.is('0', seleniumFireFoxReplicaCount, 'Selenium Firefox Node did not scale down to 0 pods')
});

test.serial('should create one chrome and firefox node', t => {
  const seleniumGridTestDeployTmpFile = tmp.fileSync();
  fs.writeFileSync(
    seleniumGridTestDeployTmpFile.name,
    seleniumGridTestsYaml
      .replace(/{{JOB_NAME}}/g, seleniumGridTestName)
      .replace(/{{CONTAINER_NAME}}/g, seleniumGridTestName)
      .replace(/{{HOST_NAME}}/g, seleniumGridHostName)
      .replace(/{{PORT}}/g, seleniumGridPort)
      .replace(/{{WITH_VERSION}}/g, "false")
  );

  t.is(0, sh.exec(`kubectl apply --namespace ${seleniumGridNamespace} -f ${seleniumGridTestDeployTmpFile.name}`).code, 'creating a Selenium Grid Tests deployment should work.');

  // wait for selenium grid tests to start running
  for (let i = 0; i < 20; i++) {
    const running = sh.exec(`kubectl get job ${seleniumGridTestName} --namespace ${seleniumGridNamespace} -o jsonpath='{.items[0].status.running}'`).stdout
    if (running == '1') {
      break;
    }
    sh.exec('sleep 1s')
  }

  let seleniumChromeNodeReplicaCount = '0';
  let seleniumFireFoxReplicaCount = '0';
  for (let i = 0; i < 30; i++) {
    seleniumChromeNodeReplicaCount = seleniumChromeNodeReplicaCount != '1' ? sh.exec(`kubectl get deploy/selenium-chrome-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout : seleniumChromeNodeReplicaCount;
    seleniumFireFoxReplicaCount = seleniumFireFoxReplicaCount != '1' ? sh.exec(`kubectl get deploy/selenium-firefox-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout : seleniumFireFoxReplicaCount;
    if (seleniumChromeNodeReplicaCount == '1' && seleniumFireFoxReplicaCount == '1') {
      break;
    }
    console.log('Waiting for chrome to scale up 1 pod and firefox to 1 pod');
    sh.exec('sleep 2s')
  }

  t.is('1', seleniumChromeNodeReplicaCount, 'Selenium Chrome Node did not scale up to 1 pod')
  t.is('1', seleniumFireFoxReplicaCount, 'Selenium Firefox Node did not scale up to 1 pod')

  // wait for selenium grid tests to complete
  let succeeded = '0';
  for (let i = 0; i < 60; i++) {
    succeeded = sh.exec(`kubectl get job ${seleniumGridTestName} --namespace ${seleniumGridNamespace} -o jsonpath='{.items[0].status.succeeded}'`).stdout
    if (succeeded == '1') {
      break;
    }
    sh.exec('sleep 1s')
  }

  sh.exec(`kubectl delete job/${seleniumGridTestName} --namespace ${seleniumGridNamespace}`)
});

test.serial('should scale down chrome and firefox nodes to 0', t => {

  let seleniumChromeNodeReplicaCount = '1';
  let seleniumFireFoxReplicaCount = '1';
  for (let i = 0; i < 65; i++) {
    seleniumChromeNodeReplicaCount = sh.exec(`kubectl get deploy/selenium-chrome-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout;
    seleniumFireFoxReplicaCount = sh.exec(`kubectl get deploy/selenium-firefox-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout;
    if (seleniumChromeNodeReplicaCount == '0' && seleniumFireFoxReplicaCount == '0') {
      break;
    }
    console.log('Waiting for chrome and firefox to scale down to 0 pod');
    sh.exec('sleep 5s')
  }

  t.is('0', seleniumChromeNodeReplicaCount, 'Selenium Chrome Node did not scale down to 0 pod')
  t.is('0', seleniumFireFoxReplicaCount, 'Selenium Firefox Node did not scale down to 0 pod')
});

test.serial('should create two chrome and one firefox nodes', t => {
  const chrome91DeployTmpFile = tmp.fileSync();
  fs.writeFileSync(chrome91DeployTmpFile.name, chrome91Yaml.replace(/{{NAMESPACE}}/g, seleniumGridNamespace).replace(/{{SELENIUM_GRID_GRAPHQL_URL}}/g, seleniumGridGraphQLUrl));

  t.is(0, sh.exec(`kubectl apply --namespace ${seleniumGridNamespace} -f ${chrome91DeployTmpFile.name}`).code, 'creating Chrome 91 node should work.')

  let seleniumChrome91NodeReplicaCount = '1';
  for (let i = 0; i < 60; i++) {
    seleniumChrome91NodeReplicaCount = sh.exec(`kubectl get deploy/selenium-chrome-node-91 -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    if (seleniumChrome91NodeReplicaCount == '0') {
      break;
    }
    console.log('Waiting for chrome 91 to scale down to 0 pods')
    sh.exec('sleep 5s')
  }

  const seleniumGridTestDeployTmpFile = tmp.fileSync();
  fs.writeFileSync(
    seleniumGridTestDeployTmpFile.name,
    seleniumGridTestsYaml
      .replace(/{{JOB_NAME}}/g, seleniumGridTestName)
      .replace(/{{CONTAINER_NAME}}/g, seleniumGridTestName)
      .replace(/{{HOST_NAME}}/g, seleniumGridHostName)
      .replace(/{{PORT}}/g, seleniumGridPort)
      .replace(/{{WITH_VERSION}}/g, "true")
  );

  t.is(0, sh.exec(`kubectl apply --namespace ${seleniumGridNamespace} -f ${seleniumGridTestDeployTmpFile.name}`).code, 'creating a Selenium Grid Tests deployment should work.');

  // wait for selenium grid tests to start running
  for (let i = 0; i < 20; i++) {
    const running = sh.exec(`kubectl get job ${seleniumGridTestName} --namespace ${seleniumGridNamespace} -o jsonpath='{.items[0].status.running}'`).stdout
    if (running == '1') {
      break;
    }
    sh.exec('sleep 1s')
  }

  let seleniumChromeNodeReplicaCount = '0';
  let seleniumFireFoxReplicaCount = '0';
  seleniumChrome91NodeReplicaCount = '0';
  for (let i = 0; i < 30; i++) {
    seleniumChromeNodeReplicaCount = seleniumChromeNodeReplicaCount != '1' ? sh.exec(`kubectl get deploy/selenium-chrome-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout : seleniumChromeNodeReplicaCount;
    seleniumFireFoxReplicaCount = seleniumFireFoxReplicaCount != '1' ? sh.exec(`kubectl get deploy/selenium-firefox-node -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout : seleniumFireFoxReplicaCount;
    seleniumChrome91NodeReplicaCount = seleniumChrome91NodeReplicaCount != '1' ? sh.exec(`kubectl get deploy/selenium-chrome-node-91 -n ${seleniumGridNamespace} -o jsonpath='{.spec.replicas}'`).stdout : seleniumChrome91NodeReplicaCount;
    if (seleniumChromeNodeReplicaCount == '1' && seleniumFireFoxReplicaCount == '1' && seleniumChrome91NodeReplicaCount == '1') {
      break;
    }
    console.log('Waiting for chrome to scale up 2 pods and firefox to 1 pod');
    sh.exec('sleep 2s')
  }

  sh.exec(`kubectl delete job/${seleniumGridTestName} --namespace ${seleniumGridNamespace}`)

  t.is('1', seleniumChromeNodeReplicaCount, 'Selenium Chrome Node did not scale up to 1 pod')
  t.is('1', seleniumChrome91NodeReplicaCount, 'Selenium Chrome 91 Node did not scale up to 1 pod')
  t.is('1', seleniumFireFoxReplicaCount, 'Selenium Firefox Node did not scale up to 1 pod')
});

test.after.always.cb('clean up prometheus deployment', t => {
  let resources = [
    'scaledobject.keda.sh/selenium-grid-chrome-scaledobject',
    'scaledobject.keda.sh/selenium-grid-firefox-scaledobject',
    'service/selenium-chrome-node',
    'deployment.apps/selenium-chrome-node',
    'service/selenium-firefox-node',
    'deployment.apps/selenium-firefox-node',
    'service/selenium-hub',
    'deployment.apps/selenium-hub',
    `job/${seleniumGridTestName}`,
    'config/selenium-event-bus-config'
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${seleniumGridNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${seleniumGridNamespace}`)

  t.end()
});

const seleniumGridYaml = `---
# Source: selenium-grid/templates/event-bus-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: selenium-event-bus-config
  namespace: {{NAMESPACE}}
  labels:
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
data:
  SE_EVENT_BUS_HOST: selenium-hub
  SE_EVENT_BUS_PUBLISH_PORT: "4442"
  SE_EVENT_BUS_SUBSCRIBE_PORT: "4443"
---
# Source: selenium-grid/templates/chrome-node-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: selenium-chrome-node
  namespace: {{NAMESPACE}}
  labels:
    name: selenium-chrome-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  type: ClusterIP
  selector:
    app: selenium-chrome-node
  ports:
    - name: tcp-chrome
      protocol: TCP
      port: 6900
      targetPort: 5900
---
# Source: selenium-grid/templates/firefox-node-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: selenium-firefox-node
  namespace: {{NAMESPACE}}
  labels:
    name: selenium-firefox-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  type: ClusterIP
  selector:
    app: selenium-firefox-node
  ports:
    - name: tcp-firefox
      protocol: TCP
      port: 6900
      targetPort: 5900
---
# Source: selenium-grid/templates/hub-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: selenium-hub
  namespace: {{NAMESPACE}}
  labels:
    app: selenium-hub
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  selector:
    app: selenium-hub
  type: NodePort
  ports:
    - name: http-hub
      protocol: TCP
      port: 4444
      targetPort: 4444
    - name: tcp-hub-pub
      protocol: TCP
      port: 4442
      targetPort: 4442
    - name: tcp-hub-sub
      protocol: TCP
      port: 4443
      targetPort: 4443
---
# Source: selenium-grid/templates/chrome-node-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: selenium-chrome-node
  namespace: {{NAMESPACE}}
  labels: &chrome_node_labels
    app: selenium-chrome-node
    app.kubernetes.io/name: selenium-chrome-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  replicas: 1
  selector:
    matchLabels:
      app: selenium-chrome-node
  template:
    metadata:
      labels: *chrome_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
        - name: selenium-chrome-node
          image: selenium/node-chrome:4.0.0-rc-1-prerelease-20210618
          imagePullPolicy: IfNotPresent
          envFrom:
            - configMapRef:
                name: selenium-event-bus-config
          ports:
            - containerPort: 5553
              protocol: TCP
          volumeMounts:
            - name: dshm
              mountPath: /dev/shm
          resources:
            limits:
              cpu: "1"
              memory: 1Gi
            requests:
              cpu: "1"
              memory: 1Gi
      volumes:
        - name: dshm
          emptyDir:
            medium: Memory
            sizeLimit: 1Gi
---
# Source: selenium-grid/templates/firefox-node-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: selenium-firefox-node
  namespace: {{NAMESPACE}}
  labels: &firefox_node_labels
    app: selenium-firefox-node
    app.kubernetes.io/name: selenium-firefox-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  replicas: 1
  selector:
    matchLabels:
      app: selenium-firefox-node
  template:
    metadata:
      labels: *firefox_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
        - name: selenium-firefox-node
          image: selenium/node-firefox:4.0.0-rc-1-prerelease-20210618
          imagePullPolicy: IfNotPresent
          envFrom:
            - configMapRef:
                name: selenium-event-bus-config
          ports:
            - containerPort: 5553
              protocol: TCP
          volumeMounts:
            - name: dshm
              mountPath: /dev/shm
          resources:
            limits:
              cpu: "1"
              memory: 1Gi
            requests:
              cpu: "1"
              memory: 1Gi
      volumes:
        - name: dshm
          emptyDir:
            medium: Memory
            sizeLimit: 1Gi
---
# Source: selenium-grid/templates/hub-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: selenium-hub
  namespace: {{NAMESPACE}}
  labels: &hub_labels
    app: selenium-hub
    app.kubernetes.io/name: selenium-hub
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  replicas: 1
  selector:
    matchLabels:
      app: selenium-hub
  template:
    metadata:
      labels: *hub_labels
    spec:
      containers:
        - name: selenium-hub
          image: selenium/hub:4.0.0-rc-1-prerelease-20210618
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 4444
              protocol: TCP
            - containerPort: 4442
              protocol: TCP
            - containerPort: 4443
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /wd/hub/status
              port: 4444
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 10
            successThreshold: 1
            failureThreshold: 10
          readinessProbe:
            httpGet:
              path: /wd/hub/status
              port: 4444
            initialDelaySeconds: 12
            periodSeconds: 10
            timeoutSeconds: 10
            successThreshold: 1
            failureThreshold: 10`

const chrome91Yaml = `# Source: selenium-grid/templates/chrome-node-91-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: selenium-chrome-node-91
  namespace: {{NAMESPACE}}
  labels: &chrome_node_labels
    app: selenium-chrome-node-91
    app.kubernetes.io/name: selenium-chrome-node-91
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  replicas: 1
  selector:
    matchLabels:
      app: selenium-chrome-node-91
  template:
    metadata:
      labels: *chrome_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
        - name: selenium-chrome-node-91
          image: selenium/node-chrome:4.0.0-rc-1-prerelease-20210618
          imagePullPolicy: IfNotPresent
          envFrom:
            - configMapRef:
                name: selenium-event-bus-config
          ports:
            - containerPort: 5553
              protocol: TCP
          volumeMounts:
            - name: dshm
              mountPath: /dev/shm
          resources:
            limits:
              cpu: "1"
              memory: 1Gi
            requests:
              cpu: "1"
              memory: 1Gi
      volumes:
        - name: dshm
          emptyDir:
            medium: Memory
            sizeLimit: 1Gi
---
# Source: selenium-grid/templates/chrome-node-91-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: selenium-chrome-node-91
  namespace: {{NAMESPACE}}
  labels:
    name: selenium-chrome-node-91
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: 4.0.0-beta-1-prerelease-20210114
    app.kubernetes.io/component: selenium-grid-4.0.0-beta-1-prerelease-20210114
    helm.sh/chart: selenium-grid-0.2.0
spec:
  type: ClusterIP
  selector:
    app: selenium-chrome-node-91
  ports:
    - name: tcp-chrome
      protocol: TCP
      port: 6900
      targetPort: 5900
---
# Source: selenium-grid/templates/chrome-node-91-hpa.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: selenium-grid-chrome-91-scaledobject
  namespace: {{NAMESPACE}}
  labels:
    deploymentName: selenium-chrome-node-91
spec:
  maxReplicaCount: 8
  scaleTargetRef:
    name: selenium-chrome-node-91
  triggers:
    - type: selenium-grid
      metadata:
        url: '{{SELENIUM_GRID_GRAPHQL_URL}}'
        browserName: 'chrome'
        browserVersion: '91.0'
---`

const scaledObjectYaml = `---
# Source: selenium-grid/templates/chrome-node-hpa.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: selenium-grid-chrome-scaledobject
  namespace: {{NAMESPACE}}
  labels:
    deploymentName: selenium-chrome-node
spec:
  maxReplicaCount: 8
  scaleTargetRef:
    name: selenium-chrome-node
  triggers:
    - type: selenium-grid
      metadata:
        url: '{{SELENIUM_GRID_GRAPHQL_URL}}'
        browserName: 'chrome'
---
# Source: selenium-grid/templates/firefox-node-hpa.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: selenium-grid-firefox-scaledobject
  namespace: {{NAMESPACE}}
  labels:
    deploymentName: selenium-firefox-node
spec:
  maxReplicaCount: 8
  scaleTargetRef:
    name: selenium-firefox-node
  triggers:
    - type: selenium-grid
      metadata:
        url: '{{SELENIUM_GRID_GRAPHQL_URL}}'
        browserName: 'firefox'`

const seleniumGridTestsYaml = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: {{JOB_NAME}}
  name: {{JOB_NAME}}
spec:
  template:
    metadata:
      labels:
        app: {{JOB_NAME}}
    spec:
      containers:
      - name: {{CONTAINER_NAME}}
        image: prashanth0007/selenium-random-tests:v1.0.2
        imagePullPolicy: Always
        env:
        - name: HOST_NAME
          value: "{{HOST_NAME}}"
        - name: PORT
          value: "{{PORT}}"
        - name: WITH_VERSION
          value: "{{WITH_VERSION}}"
      restartPolicy: Never`
