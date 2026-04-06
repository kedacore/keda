//go:build e2e
// +build e2e

package helper

const (
	DeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mssql-consumer-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mssql-consumer-worker
  template:
    metadata:
      labels:
        app: mssql-consumer-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-consumer-worker
        command: ["/app"]
        args: ["-mode", "consumer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
`

	SecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  mssql-sa-password: {{.MssqlPassword}}
  mssql-connection-string: {{.MssqlConnectionString}}
`

	TriggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
    secretTargetRef:
    - parameter: password
      name: {{.SecretName}}
      key: mssql-sa-password
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
  - type: mssql
    metadata:
      host: {{.MssqlHostname}}
      port: "1433"
      database: {{.MssqlDatabase}}
      username: sa
      driverName: {{.DriverName}}
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "1" # one replica per row
      activationTargetValue: "15"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	MssqlStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{.MssqlServerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: mssql
spec:
  replicas: 1
  serviceName: {{.MssqlServerName}}
  selector:
     matchLabels:
       app: mssql
  template:
    metadata:
      labels:
        app: mssql
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: mssql
        image: mcr.microsoft.com/mssql/server:2019-latest
        ports:
        - containerPort: 1433
        env:
        - name: MSSQL_PID
          value: "Developer"
        - name: ACCEPT_EULA
          value: "Y"
        - name: SA_PASSWORD
          value: {{.MssqlPassword}}
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - "/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P '{{.MssqlPassword}}' -Q 'SELECT @@Version'"
`

	MssqlServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.MssqlServerName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: mssql
  ports:
    - protocol: TCP
      port: 1433
      targetPort: 1433
  type: ClusterIP
`

	// inserts 10 records in the table
	InsertRecordsJobTemplate1 = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job1
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        command: ["/app"]
        args: ["-mode", "producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4
`

	// inserts 10 records in the table
	InsertRecordsJobTemplate2 = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job2
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        command: ["/app"]
        args: ["-mode", "producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4
  `
)
