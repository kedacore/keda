//go:build e2e
// +build e2e

package helper

import (
	"fmt"
	"strings"
)

const (
	PostgresqlImage = "postgres:10.5"
)

const (
	DeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql-update-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: postgresql-update-worker
  template:
    metadata:
      labels:
        app: postgresql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "6000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
`
	ScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: postgresql
    metadata:
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
	LowLevelRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-low-level-job
  name: postgresql-insert-low-level-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-low-level-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "20"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
	InsertRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-job
  name: postgresql-insert-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
	TriggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: connection
    name: {{.SecretName}}
    key: {{.SecretKey}}
`
	PostgreSQLStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  serviceName: {{.PostgreSQLStatefulSetName}}
  selector:
    matchLabels:
      app: {{.PostgreSQLStatefulSetName}}
  template:
    metadata:
      labels:
        app: {{.PostgreSQLStatefulSetName}}
    spec:
      containers:
      - name: postgresql
        image: {{.PostgreSQLImage}}
        env:
        - name: POSTGRES_USER
          value: {{.PostgreSQLUsername}}
        - name: POSTGRES_PASSWORD
          value: {{.PostgreSQLPassword}}
        - name: POSTGRES_DB
          value: {{.PostgreSQLDatabase}}
        ports:
          - name: postgresql
            protocol: TCP
            containerPort: 5432
        readinessProbe:
          tcpSocket:
            port: 5432
          initialDelaySeconds: 5
          timeoutSeconds: 5
        livenessProbe:
          tcpSocket:
            port: 5432
          initialDelaySeconds: 5
          timeoutSeconds: 5
        volumeMounts:
        - name: init-scripts
          mountPath: /docker-entrypoint-initdb.d
      volumes:
      - name: init-scripts
        configMap:
          name: master-slave-config
          optional: true
`
	PostgreSQLServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: {{.PostgreSQLStatefulSetName}}
  type: ClusterIP
`
)

func GetConnectionString(username string, password string, svcNames []string, namespace string, database string) string {
	svcFQDNs := make([]string, 0, len(svcNames))
	for _, svcName := range svcNames {
		svcFQDNs = append(svcFQDNs, fmt.Sprintf("%s.%s.svc.cluster.local", svcName, namespace))
	}
	fqdns := strings.Join(svcFQDNs, ",")
	return fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", username, password, fqdns, database)
}
