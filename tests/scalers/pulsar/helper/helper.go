//go:build e2e
// +build e2e

package helper

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	apachePulsarVersion = "2.10.2"
	messageCount        = 3
	minReplicaCount     = 0
	maxReplicaCount     = 5
	msgBacklog          = 10
)

type templateData struct {
	ApachePulsarVersion string
	TestName            string // Used for most resource names
	NumPartitions       int    // Use 0 to create a non-partitioned topic
	MessageCount        int
	MinReplicaCount     int
	MaxReplicaCount     int
	MsgBacklog          int
}

const authSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
data:
  key.pub: MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnkggprp2GTl/2oQgLvnspbH0Lxthhmw3O3qpcx1FVUcJeD1JlUsuK6rO8uexfY/3JuZffzEm5busJB/5zuXQqO52ph8xDRiEeHOuFY0RKv8DAfpss+oG8Ou/LdHPYCbbyjbJXK/iVE/rUhicp7n6udv2/AaqJj/9535Qo49Q+3S/fbWqhNR6r84+Q+KTHtfwuoLsE4AbZ+g7FRpnyH3iYDxC4ISr1zIJiv4o41cwglaho/cOqCpBFwRHYyZTgeEIf9+7bjTPbpPThFztxO6DOAw73ikU7iT3T0H6hgpQqKa79kw1R8PAfeTYvkeQ4juQwlYmyGePTb9F4LZ+0w7a8wIDAQAB
  token.jwt: ZXlKaGJHY2lPaUpTVXpJMU5pSjkuZXlKemRXSWlPaUpoWkcxcGJpSjkubEg2TEVqcDU3Y2pFc2xhdWV2Z1ZKV1NTa19IaThFLVZGb29EZHVxUHRiQ1Q0U0NJQlluV0YtRlA5NzBMVUMxRzFWWnZFMmJFZGlkNGd3SzhKY3RnVHNMNGJTV2V5SW4yVVBNTnNnaDVGemhWQkQ4SXVaRnFLTXktLUZnUmtKWFZzWldrbUFwNW5yamU3MEZaRkJLME1uV0licWxSZ2Y2UUZKR2Vxd1FXbzlZV0RCOUh5cTRYR0oxUGx1SGR4T282eTJjVm1Ib3c2SFV3R0dfSDZfTmk0eTNBaU0zWEhvNlNvMkEtRGU5cGRBX3d6MHQzemFyXzhBNFJNeXdTYmtXYldNSVEwUnN5bEZhSk80SzYzT0lTRG5IQkp0TUNJTUNjNlo1WDFKYWt2eUdKek9FTVNQeDZRM1hXWG1MOFFDNjBrcG1xQkd0dXV4XzZlbWFSaHZTcDlB
  tls.crt: QmFnIEF0dHJpYnV0ZXMKICAgIGZyaWVuZGx5TmFtZTogbXlrZXkKICAgIGxvY2FsS2V5SUQ6IDU0IDY5IDZEIDY1IDIwIDMxIDM2IDM2IDM4IDM4IDMyIDM4IDM3IDMwIDMxIDMzIDMxIDMzIApzdWJqZWN0PS9PVT1wdWxzYXIvTz1wdWxzYXIvQ049cHVsc2FyLmFwYWNoZS5vcmcKaXNzdWVyPS9PVT1wdWxzYXIvTz1wdWxzYXIvQ049cHVsc2FyLmFwYWNoZS5vcmcKLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURzekNDQXB1Z0F3SUJBZ0lJU1p5aFpQbzhCcVV3RFFZSktvWklodmNOQVFFTEJRQXdQakVQTUEwR0ExVUUKQ3hNR2NIVnNjMkZ5TVE4d0RRWURWUVFLRXdad2RXeHpZWEl4R2pBWUJnTlZCQU1URVhCMWJITmhjaTVoY0dGagphR1V1YjNKbk1DQVhEVEl5TVRFeE9UQXpNekUwTVZvWUR6SXhNakl4TURJMk1ETXpNVFF4V2pBK01ROHdEUVlEClZRUUxFd1p3ZFd4ellYSXhEekFOQmdOVkJBb1RCbkIxYkhOaGNqRWFNQmdHQTFVRUF4TVJjSFZzYzJGeUxtRncKWVdOb1pTNXZjbWN3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLQW9JQkFRQ3BKckZ1Mm55QQp5d3BzZDRFZURCWlNMN24xamdoUzlrRFIvMkVYU1VGMGE1M1czeG13ckRKNUR0azBCQ0wrUnNlb2J0SXRTUnpFCk9Cd1lOTFl1RmxLNHVRbTdWRk1ic3FWbTJ0c2h6bXRpRzNCQ3l6K2kzdXpEWTloakVPUjVjbzJDVDlmc0lydE0KR1N0eitGMmNHbjI2WTJMZFZRVDNQNXpoUXhXZFVydSs0cTZicFZmQ25tdnltVG9QTS9aMmVnYnBiVGllbWphYwpiS3Uya0pMZTF3bmxmcFVmWlBHa0dGQy9uTUlVUWJjblpSNG5tU3dtVGJobm8vZGRpNGI5VHhCTUNZWW45K3lICmo5ZmcvaTBTeEZ3VzB2NjVmNjJjdnNNZi8rOGd5NlVBUHF2SzYxK1ZaSy81TWQyYlpBS3N4RkUyS0k0emQ4MzcKTCt0USsvVU5HOHozQWdNQkFBR2pnYkl3Z2E4d0hRWURWUjBPQkJZRUZQM05oMHJzdHVLQ2VEQjRrR1JSSnQ4QgpXVllCTUlHTkJnTlZIUkVFZ1lVd2dZS0NPM0IxYkhOaGNpMXdZWEowYVhScGIyNWxaQzEwYjNCcFl5MTBaWE4wCkxuQjFiSE5oY2kxd1lYSjBhWFJwYjI1bFpDMTBiM0JwWXkxMFpYTjBna053ZFd4ellYSXRibTl1TFhCaGNuUnAKZEdsdmJtVmtMWFJ2Y0dsakxYUmxjM1F1Y0hWc2MyRnlMVzV2Ymkxd1lYSjBhWFJwYjI1bFpDMTBiM0JwWXkxMApaWE4wTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBY1A3OStvN2E0VGZBY2EzamtQZFV6eFdGN1FKMytoVXJzCnRaMlpGNFpLSXhTa2Y2MmlNaFdJM1B0TG1qRDVLT2t6RFFua092VXk2bVdncVd5Q2tWdHF1TE1iT1p3TXJkZysKQ01JbmRNR2NDUi9lbkk1dzg4TzdnZzZIQkZ5RHNqRjh1RnZYbGMrRU9Nc3lyTWU3cUFQTlI4cVQyV0Eyd0djcQpsMjQvQkwxRFl1YWlsTi9hNU9nSDZENHh2OHhNaGlRcWJHZnBlUE8wY2YrT0hET0FURDExOGhSck1RQXlpWWs0CjJzNHBTOFAvRUFpaHdOdXhwb3VLWmFEUnAxa2hIZmlycUVNQUozTmtkOURsMXpRdmFRK2RPVXJ3ZWJpV2FkR3UKakx5ZVFWbEFhK0NVajNKWUVrS25ST0JXdmc4Y1Nvd0dYVTRBTDZxbXZLNTlIVkFBSHg4TgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: QmFnIEF0dHJpYnV0ZXMKICAgIGZyaWVuZGx5TmFtZTogbXlrZXkKICAgIGxvY2FsS2V5SUQ6IDU0IDY5IDZEIDY1IDIwIDMxIDM2IDM2IDM4IDM4IDMyIDM4IDM3IDMwIDMxIDMzIDMxIDMzIApLZXkgQXR0cmlidXRlczogPE5vIEF0dHJpYnV0ZXM+Ci0tLS0tQkVHSU4gUFJJVkFURSBLRVktLS0tLQpNSUlFdlFJQkFEQU5CZ2txaGtpRzl3MEJBUUVGQUFTQ0JLY3dnZ1NqQWdFQUFvSUJBUUNwSnJGdTJueUF5d3BzCmQ0RWVEQlpTTDduMWpnaFM5a0RSLzJFWFNVRjBhNTNXM3htd3JESjVEdGswQkNMK1JzZW9idEl0U1J6RU9Cd1kKTkxZdUZsSzR1UW03VkZNYnNxVm0ydHNoem10aUczQkN5eitpM3V6RFk5aGpFT1I1Y28yQ1Q5ZnNJcnRNR1N0egorRjJjR24yNlkyTGRWUVQzUDV6aFF4V2RVcnUrNHE2YnBWZkNubXZ5bVRvUE0vWjJlZ2JwYlRpZW1qYWNiS3UyCmtKTGUxd25sZnBVZlpQR2tHRkMvbk1JVVFiY25aUjRubVN3bVRiaG5vL2RkaTRiOVR4Qk1DWVluOSt5SGo5ZmcKL2kwU3hGd1cwdjY1ZjYyY3ZzTWYvKzhneTZVQVBxdks2MStWWksvNU1kMmJaQUtzeEZFMktJNHpkODM3TCt0UQorL1VORzh6M0FnTUJBQUVDZ2dFQUNXYTdwT0FpM0Z1c25DTzJPdS9NQzh4WVJ4UWFWVllYZXpSNDluemRWUFdvClE2bUp1WDZRblpiY0xwNXVQWGk4bnhsdHVCT2d0QzAwTG9vN2QrdEl0TGlnR0ZmUytLNmdyOHRKTTZOUDU1ZUQKMFVxUG9tTkdnSU9ib3NIdEdPenJmWXNuZ3BuWmxCeXdCQldSU2x4VWtaZjFoanl6OW5RRUthYjdYQStkbkxuUQpTcGdXZncrRHZlc2JlYWMrM2lIV2FaUnBoOWMwUklRQVZ1UHozODhFbTQwSVQwdVlURnliSDF0bU5sdG8rVDlpCkgzenNBWU9mS2ZKeFh4OXd6SWI4d2JOc3ZnSFVhSm1XNGNlRmFkbnd1YktFMU5RZ3ZYaXlXRVQ3Q1cvdjBDRnoKSHgvU2l2elpqQlJ0UjVhOEF5NnNHVHdmNE03Nm9kNE1hcit6M011azdRS0JnUUMyY2tiRHJHd0hhQXJCaDg4egp5S2pqeHAyekxLMzY4dnhmazJrRDBkTUg1bmdmc2J4TlRNRDdIRENjbitUSExvVzNGNWdsYitldW5tMksycHFUClNHUHlIVGl3S2twNjVJSGRTZzBUNk5HbDVrRS85SktCZWpQM3I0VlcwVHZ3M2R6N1ZvWHVsVmpsTGoxNUkxWU8KY0NvT3djczVTTHlvMGY3R2FiTWcyMHhEUXdLQmdRRHRXRUZ3d29hc2pKcjBCN1pBRWRBZGN2R1ptUmFSNnJZUAoxKzdTdHFXN0xzNkdSZG5lRkpTZStzZmIwUmM2Nm91WjRwQmRsYlZ4bGpkSHJlNTJaYkR3cVpXcHQrc05MTE5zCkxHL29VOGFhempGK0ovMnZCdTBQblYybkUxMy9RcWhONmpvWjVsU2k2YjFmTzEyVFpSSlp5MlIxRTBzZTZxbjUKR09VT2ZqM0NQUUtCZ0JqdzBFbXBoVzhSd3Y2bjBTUjBGdHBrYVdSNEJDU2RHUEQ3MXN4RjM4Smh1Q1FsQ09mTQpTVWxLbGo2akFRUlZrTVB4dnNQSFkzV1VoTWNKa1QzM0ZHcWhvZ0U3RnNscitYREYwYm5hQnViVjdpK1BBSVFnCnIzLzVoNUhSc283LzFWaXFnRTZZTGZuT2MycmU4TUd5aFoxVTBySTNCa3RSd2JGZis3UFBKc0svQW9HQVZxUkYKSDFpanVSR0s3MUp4WVdvZlF1RFcrVzg5SWY5QWZ3QWdtcU02Vk41OVhkN1o3WXd0eE90ZlVncytJNi9EVG1XNgp0YThWRVdYNHdCM3FVeVpFTlZaeTRBWFh0SE9BL0Jnc3NlOERMVGZnTVdGLzVnanRPU29GS2h5VHo3OFJtWC9MCnZmQ3JMTjJPMTlqZ0RCSjFaSG92TGQzaEttUVhzR3M2RXRSYXp6RUNnWUVBazBuT3lmbWd2YVVrMHZSTWJnRmEKVmIzUFdNNG0rMG5sbU4wdnpxSHBPbi9SRDc2MlZlT0cxNnNCOFpFKzVlOWFVVWxxTnUxL3M4blByajhDbGhweApmc0RkQnJKZnBHbjBJUGdKNUF4R3F1WEFEOFl3WmpodFp5RVR5bVRlRm9YemNyNVZWaC90cC9HVmlIOXB4VDhGCktKVm1ybWpiMm1HUjBHallNM2dhVGlJPQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==
`

const pulsarStatefulsetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
 name: {{.TestName}}
 namespace: {{.TestName}}
 labels:
  app: pulsar
spec:
  selector:
    matchLabels:
      app: pulsar
  replicas: 1
  serviceName: {{.TestName}}
  template:
    metadata:
      labels:
        app: pulsar
    spec:
      containers:
      - name: pulsar
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/pulsar/secrets"
          readOnly: true
        readinessProbe:
          tcpSocket:
            port: 8080
        ports:
        - name: pulsar
          containerPort: 6650
          protocol: TCP
        - name: http
          containerPort: 8080
          protocol: TCP
        - name: https
          containerPort: 8443
          protocol: TCP
        env:
        - name: brokerDeleteInactiveTopicsEnabled
          value: "false"
        - name: authenticationEnabled
          value: "true"
        - name: authenticationProviders
          value: "org.apache.pulsar.broker.authentication.AuthenticationProviderToken"
        - name: PULSAR_PREFIX_tokenPublicKey
          value: "/pulsar/secrets/key.pub"
        - name: brokerClientAuthenticationPlugin
          value: "org.apache.pulsar.client.impl.auth.AuthenticationToken"
        - name: brokerClientAuthenticationParameters
          value: "file:///pulsar/secrets/token.jwt"
        - name: PULSAR_PREFIX_webServicePortTls
          value: "8443"
        - name: tlsKeyFilePath
          value: "/pulsar/secrets/tls.key"
        - name: tlsCertificateFilePath
          value: "/pulsar/secrets/tls.crt"
        command:
        - sh
        - -c
        args: ["bin/apply-config-from-env.py conf/client.conf && bin/apply-config-from-env.py conf/standalone.conf && exec bin/pulsar standalone -nfw -nss"]
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
`

const pulsarServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  type: ClusterIP
  clusterIP: None
  ports:
  - name: http
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: https
    port: 8443
    targetPort: 8443
    protocol: TCP
  - name: pulsar
    port: 6650
    targetPort: 6650
    protocol: TCP
  selector:
    app: pulsar
`

const topicInitJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.TestName}}-topic-init
  namespace: {{.TestName}}
spec:
  template:
    spec:
      containers:
      - name: pulsar-topic-init
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/pulsar/auth"
          readOnly: true
        command:
        - sh
        - -c
        args: ["bin/pulsar-admin --admin-url http://{{.TestName}}.{{.TestName}}:8080 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt topics {{ if .NumPartitions }} create-partitioned-topic -p {{.NumPartitions}} {{ else }} create {{ end }} persistent://public/default/keda"]
      restartPolicy: Never
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
  backoffLimit: 4
`

const consumerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.TestName}}-consumer
  namespace: {{.TestName}}
  labels:
    app: pulsar-consumer
spec:
  selector:
    matchLabels:
      app: pulsar-consumer
  template:
    metadata:
      labels:
        app: pulsar-consumer
    spec:
      containers:
        - name: pulsar-consumer
          image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
          imagePullPolicy: IfNotPresent
          volumeMounts:
          - name: auth-data
            mountPath: "/pulsar/auth"
            readOnly: true
          command:
          - sh
          - -c
          args: ["bin/pulsar-perf consume --service-url pulsar://{{.TestName}}.{{.TestName}}:6650 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt --receiver-queue-size 1 --subscription-type Shared --rate 1 --subscriptions keda persistent://public/default/keda"]
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
`

const scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  scaleTargetRef:
    name: {{.TestName}}-consumer
  pollingInterval: 5 # Optional. Default: 30 seconds
  cooldownPeriod: 30 # Optional. Default: 300 seconds
  maxReplicaCount: {{.MaxReplicaCount}}
  minReplicaCount: {{.MinReplicaCount}}
  triggers:
    - type: pulsar
      metadata:
        msgBacklogThreshold: "{{.MsgBacklog}}"
        activationMsgBacklogThreshold: "5"
        adminURL: https://{{.TestName}}.{{.TestName}}:8443
        topic:  persistent://public/default/keda
        isPartitionedTopic: {{ if .NumPartitions }} "true" {{else}} "false" {{end}}
        authModes: "bearer"
        subscription: keda
      authenticationRef:
        name: {{.TestName}}
          `

const authenticationRefTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  secretTargetRef:
    - parameter: bearerToken
      name: {{.TestName}}
      key: token.jwt
    - parameter: ca
      name: {{.TestName}}
      key: tls.crt
`

const topicPublishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.TestName}}-producer
  namespace: {{.TestName}}
spec:
  template:
    spec:
      containers:
      - name: pulsar-producer
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/pulsar/auth"
          readOnly: true
        command:
        - sh
        - -c
        args: ["bin/pulsar-perf produce --admin-url http://{{.TestName}}.{{.TestName}}:8080 --service-url pulsar://{{.TestName}}.{{.TestName}}:6650 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt --num-messages {{.MessageCount}} {{ if .NumPartitions }} --partitions {{.NumPartitions}} {{ end }} --batch-max-messages 1 persistent://public/default/keda"]
      restartPolicy: Never
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
  backoffLimit: 4
`

func TestScalerWithConfig(t *testing.T, testName string, numPartitions int) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := helper.GetKubernetesClient(t)
	data, templates := getTemplateData(testName, numPartitions)
	t.Cleanup(func() {
		helper.KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
		helper.KubectlDeleteWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
		helper.KubectlDeleteWithTemplate(t, data, "topicInitJobTemplate", topicInitJobTemplate)

		helper.DeleteKubernetesResources(t, testName, data, templates)
	})

	helper.CreateKubernetesResources(t, kc, testName, data, templates)

	require.True(t, helper.WaitForStatefulsetReplicaReadyCount(t, kc, testName, testName, 1, 300, 1),
		"replica count should be 1 within 5 minutes")

	helper.KubectlReplaceWithTemplate(t, data, "topicInitJobTemplate", topicInitJobTemplate)

	require.True(t, helper.WaitForJobSuccess(t, kc, getTopicInitJobName(testName), testName, 300, 1),
		"job should succeed within 5 minutes")

	helper.KubectlApplyWithTemplate(t, data, "consumerTemplate", consumerTemplate)

	// run consumer for create subscription
	require.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 1, 300, 1),
		"replica count should be 1 within 5 minutes")

	helper.KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 0, 60, 1),
		"replica count should be 0 after a minute")

	testActivation(t, kc, data)
	// scale out
	testScaleOut(t, kc, data)
	// scale in
	testScaleIn(t, kc, testName)
}

func getTemplateData(testName string, numPartitions int) (templateData, []helper.Template) {
	return templateData{
			ApachePulsarVersion: apachePulsarVersion,
			TestName:            testName,
			NumPartitions:       numPartitions,
			MessageCount:        messageCount,
			MinReplicaCount:     minReplicaCount,
			MaxReplicaCount:     maxReplicaCount,
			MsgBacklog:          msgBacklog,
		}, []helper.Template{
			{Name: "statefulsetTemplate", Config: pulsarStatefulsetTemplate},
			{Name: "serviceTemplate", Config: pulsarServiceTemplate},
			{Name: "authenticationRefTemplate", Config: authenticationRefTemplate},
			{Name: "secretTemplate", Config: authSecretTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	// publish message and less than MsgBacklog
	helper.KubectlReplaceWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, getConsumerDeploymentName(data.TestName), data.TestName, data.MinReplicaCount, 60)
	helper.KubectlReplaceWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	data.MessageCount = 100
	helper.KubectlReplaceWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(data.TestName), data.TestName, 5, 300, 1),
		"replica count should be 5 within 5 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, testName string) {
	t.Log("--- testing scale in ---")
	// Check if deployment scale in to 0 after 5 minutes
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 0, 300, 1),
		"Replica count should be 0 within 5 minutes")
}

func getConsumerDeploymentName(testName string) string {
	return fmt.Sprintf("%s-consumer", testName)
}

func getTopicInitJobName(testName string) string {
	return fmt.Sprintf("%s-topic-init", testName)
}
