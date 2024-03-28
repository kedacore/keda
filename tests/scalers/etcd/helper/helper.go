//go:build e2e
// +build e2e

package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	Namespace string
	EtcdName  string
}

const (
	statefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: {{.EtcdName}}
  name: {{.EtcdName}}
  namespace: {{.Namespace}}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{.EtcdName}}
  serviceName: etcd-headless
  template:
    metadata:
      labels:
        app: {{.EtcdName}}
      name: {{.EtcdName}}
    spec:
      containers:
        - env:
          - name: MY_POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: MY_POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
          image: gcr.io/etcd-development/etcd:v3.4.20
          command:
          - sh
          - -c
          - "/usr/local/bin/etcd --name $MY_POD_NAME \
            --data-dir /etcd-data \
            --listen-client-urls http://$MY_POD_IP:2379 \
            --advertise-client-urls http://$MY_POD_IP:2379 \
            --listen-peer-urls http://$MY_POD_IP:2380 \
            --initial-advertise-peer-urls http://$MY_POD_IP:2380 \
            --initial-cluster {{.EtcdName}}-0=http://{{.EtcdName}}-0.etcd-headless.{{.Namespace}}:2380,{{.EtcdName}}-1=http://{{.EtcdName}}-1.etcd-headless.{{.Namespace}}:2380,{{.EtcdName}}-2=http://{{.EtcdName}}-2.etcd-headless.{{.Namespace}}:2380 \
            --initial-cluster-token tkn \
            --initial-cluster-state new \
            --experimental-watch-progress-notify-interval 10s \
			--log-level info \
            --logger zap \
            --log-outputs stderr"
          imagePullPolicy: IfNotPresent
          name: etcd
          ports:
          - containerPort: 2380
            name: peer
            protocol: TCP
          - containerPort: 2379
            name: client
            protocol: TCP
          volumeMounts:
          - mountPath: /etcd-data
            name: cache-volume
      volumes:
      - name: cache-volume
        emptyDir: {}
`
	headlessServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.EtcdName}}
  name: etcd-headless
  namespace: {{.Namespace}}
spec:
  clusterIP: None
  ports:
  - name: infra-etcd-cluster-2379
    port: 2379
    protocol: TCP
    targetPort: 2379
  - name: infra-etcd-cluster-2380
    port: 2380
    protocol: TCP
    targetPort: 2380
  selector:
    app: {{.EtcdName}}
  type: ClusterIP
`
	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.EtcdName}}
  name: etcd-svc
  namespace: {{.Namespace}}
spec:
  ports:
  - name: etcd-cluster
    port: 2379
    targetPort: 2379
  selector:
    app: {{.EtcdName}}
  sessionAffinity: None
  type: NodePort
`
)

var etcdClusterTemplates = []helper.Template{
	{Name: "statefulSetTemplate", Config: statefulSetTemplate},
	{Name: "headlessServiceTemplate", Config: headlessServiceTemplate},
	{Name: "serviceTemplate", Config: serviceTemplate},
}

func InstallCluster(t *testing.T, kc *kubernetes.Clientset, name, namespace string) {
	var data = templateData{
		Namespace: namespace,
		EtcdName:  name,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, etcdClusterTemplates)
	require.True(t, helper.WaitForStatefulsetReplicaReadyCount(t, kc, name, namespace, 3, 60, 5),
		"etcd-cluster should be up")
}
