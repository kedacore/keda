//go:build e2e
// +build e2e

package helper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	Namespace     string
	RedisName     string
	RedisPassword string
}

var (
	redisStandaloneTemplates = []helper.Template{
		{Name: "standaloneRedisTemplate", Config: standaloneRedisTemplate},
		{Name: "standaloneRedisServiceTemplate", Config: standaloneRedisServiceTemplate},
	}

	redisClusterTemplates = []helper.Template{
		{Name: "clusterRedisSecretTemplate", Config: clusterRedisSecretTemplate},
		{Name: "clusterRedisConfig1Template", Config: clusterRedisConfig1Template},
		{Name: "clusterRedisConfig2Template", Config: clusterRedisConfig2Template},
		{Name: "clusterRedisHeadlessServiceTemplate", Config: clusterRedisHeadlessServiceTemplate},
		{Name: "clusterRedisServiceTemplate", Config: clusterRedisServiceTemplate},
		{Name: "clusterRedisStatefulSetTemplate", Config: clusterRedisStatefulSetTemplate},
	}
)

const (
	standaloneRedisTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.RedisName}}
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: {{.RedisName}}
  replicas: 1
  template:
    metadata:
      labels:
        app: {{.RedisName}}
    spec:
      containers:
      - name: master
        image: redis:6.0.6
        command: ["redis-server", "--requirepass", {{.RedisPassword}}]
        ports:
        - containerPort: 6379`

	standaloneRedisServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: {{.Namespace}}
  labels:
    app: {{.RedisName}}
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: {{.RedisName}}`

	clusterRedisSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: redis-cluster
  namespace: {{.Namespace}}
type: Opaque
stringData:
  redis-password: "{{.RedisPassword}}"`
	clusterRedisConfig1Template = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-cluster-default
  namespace: {{.Namespace}}
data:
  redis-default.conf: |-
    bind 127.0.0.1
    protected-mode yes
    port 6379
    tcp-backlog 511
    timeout 0
    tcp-keepalive 300
    daemonize no
    supervised no
    pidfile /opt/bitnami/redis/tmp/redis_6379.pid
    loglevel notice
    logfile ""
    databases 16
    always-show-logo yes
    save 900 1
    save 300 10
    save 60 10000
    stop-writes-on-bgsave-error yes
    rdbcompression yes
    rdbchecksum yes
    dbfilename dump.rdb
    rdb-del-sync-files no
    dir /bitnami/redis/data
    replica-serve-stale-data yes
    replica-read-only yes
    repl-diskless-sync no
    repl-diskless-sync-delay 5
    repl-diskless-load disabled
    repl-disable-tcp-nodelay no
    replica-priority 100
    acllog-max-len 128
    lazyfree-lazy-eviction no
    lazyfree-lazy-expire no
    lazyfree-lazy-server-del no
    replica-lazy-flush no
    lazyfree-lazy-user-del no
    appendonly no
    appendfilename "appendonly.aof"
    appendfsync everysec
    no-appendfsync-on-rewrite no
    auto-aof-rewrite-percentage 100
    auto-aof-rewrite-min-size 64mb
    aof-load-truncated yes
    aof-use-rdb-preamble yes
    lua-time-limit 5000
    cluster-enabled yes
    cluster-config-file /bitnami/redis/data/nodes.conf
    slowlog-log-slower-than 10000
    slowlog-max-len 128
    latency-monitor-threshold 0
    notify-keyspace-events ""
    hash-max-ziplist-entries 512
    hash-max-ziplist-value 64
    list-max-ziplist-size -2
    list-compress-depth 0
    set-max-intset-entries 512
    zset-max-ziplist-entries 128
    zset-max-ziplist-value 64
    hll-sparse-max-bytes 3000
    stream-node-max-bytes 4096
    stream-node-max-entries 100
    activerehashing yes
    client-output-buffer-limit normal 0 0 0
    client-output-buffer-limit replica 256mb 64mb 60
    client-output-buffer-limit pubsub 32mb 8mb 60
    hz 10
    dynamic-hz yes
    aof-rewrite-incremental-fsync yes
    rdb-save-incremental-fsync yes
    jemalloc-bg-thread yes`
	clusterRedisConfig2Template = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-cluster-scripts
  namespace: {{.Namespace}}
data:
  ping_readiness_local.sh: |-
    #!/bin/sh
    set -e
    REDIS_STATUS_FILE=/tmp/.redis_cluster_check
    if [ ! -z "$REDIS_PASSWORD" ]; then export REDISCLI_AUTH=$REDIS_PASSWORD; fi;
    response=$(
      timeout -s 3 $1 \
      redis-cli \
        -h localhost \
        -p $REDIS_PORT \
        ping
    )
    if [ "$?" -eq "124" ]; then
      echo "Timed out"
      exit 1
    fi
    if [ "$response" != "PONG" ]; then
      echo "$response"
      exit 1
    fi
    if [ ! -f "$REDIS_STATUS_FILE" ]; then
      response=$(
        timeout -s 3 $1 \
        redis-cli \
          -h localhost \
          -p $REDIS_PORT \
          CLUSTER INFO | grep cluster_state | tr -d '[:space:]'
      )
      if [ "$?" -eq "124" ]; then
        echo "Timed out"
        exit 1
      fi
      if [ "$response" != "cluster_state:ok" ]; then
        echo "$response"
        exit 1
      else
        touch "$REDIS_STATUS_FILE"
      fi
    fi
  ping_liveness_local.sh: |-
    #!/bin/sh
    set -e
    if [ ! -z "$REDIS_PASSWORD" ]; then export REDISCLI_AUTH=$REDIS_PASSWORD; fi;
    response=$(
      timeout -s 3 $1 \
      redis-cli \
        -h localhost \
        -p $REDIS_PORT \
        ping
    )
    if [ "$?" -eq "124" ]; then
      echo "Timed out"
      exit 1
    fi
    responseFirstWord=$(echo $response | head -n1 | awk '{print $1;}')
    if [ "$response" != "PONG" ] && [ "$responseFirstWord" != "LOADING" ] && [ "$responseFirstWord" != "MASTERDOWN" ]; then
      echo "$response"
      exit 1
    fi`
	clusterRedisHeadlessServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.RedisName}}-headless
  namespace: {{.Namespace}}
spec:
  type: ClusterIP
  clusterIP: None
  publishNotReadyAddresses: true
  ports:
    - name: tcp-redis
      port: 6379
      targetPort: tcp-redis
    - name: tcp-redis-bus
      port: 16379
      targetPort: tcp-redis-bus
  selector:
    app.kubernetes.io/name: redis-cluster
    app.kubernetes.io/instance: redis`
	clusterRedisServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.RedisName}}
  namespace: {{.Namespace}}
  annotations:
spec:
  type: ClusterIP
  ports:
    - name: tcp-redis
      port: 6379
      targetPort: tcp-redis
      protocol: TCP
      nodePort: null
  selector:
    app.kubernetes.io/name: redis-cluster
    app.kubernetes.io/instance: redis`
	clusterRedisStatefulSetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{.RedisName}}
  namespace: {{.Namespace}}
  labels:
    app.kubernetes.io/name: redis-cluster
    helm.sh/chart: redis-cluster-7.5.1
    app.kubernetes.io/instance: redis
    app.kubernetes.io/managed-by: Helm
spec:
  updateStrategy:
    rollingUpdate:
      partition: 0
    type: RollingUpdate
  selector:
    matchLabels:
      app.kubernetes.io/name: redis-cluster
      app.kubernetes.io/instance: redis
  replicas: 6
  serviceName: {{.RedisName}}-headless
  podManagementPolicy: Parallel
  template:
    metadata:
      labels:
        app.kubernetes.io/name: redis-cluster
        helm.sh/chart: redis-cluster-7.5.1
        app.kubernetes.io/instance: redis
        app.kubernetes.io/managed-by: Helm
      annotations:
        checksum/scripts: ce1a29fc397d40685cec7ddd4275fe0bda4455d37d622ca301781996a1dc0fa1
        checksum/secret: 289df422d0e95311f552b860b794245d1372a2bd362835f6431f1ba128a90843
        checksum/config: 5c811a8da6bdb4552e50761422ae69932819fffd90a8c705455296107accc667
    spec:
      hostNetwork: false
      enableServiceLinks: false
      securityContext:
        fsGroup: 1001
        runAsUser: 1001
        sysctls: []
      serviceAccountName: default
      affinity:
        podAffinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app.kubernetes.io/name: redis-cluster
                    app.kubernetes.io/instance: redis
                namespaces:
                  - {{.Namespace}}
                topologyKey: kubernetes.io/hostname
              weight: 1
        nodeAffinity:
      containers:
        - name: redis-cluster
          image: docker.io/bitnami/redis-cluster:6.2.7-debian-10-r0
          imagePullPolicy: "IfNotPresent"
          securityContext:
            runAsNonRoot: true
            runAsUser: 1001
          command: ['/bin/bash', '-c']
          args:
            - |
              if ! [[ -f /opt/bitnami/redis/etc/redis.conf ]]; then
                  echo COPYING FILE
                  cp  /opt/bitnami/redis/etc/redis-default.conf /opt/bitnami/redis/etc/redis.conf
              fi
              pod_index=($(echo "$POD_NAME" | tr "-" "\n"))
              pod_index="${pod_index[-1]}"
              if [[ "$pod_index" == "0" ]]; then
                export REDIS_CLUSTER_CREATOR="yes"
                export REDIS_CLUSTER_REPLICAS="1"
              fi

              /opt/bitnami/scripts/redis-cluster/entrypoint.sh /opt/bitnami/scripts/redis-cluster/run.sh
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: REDIS_NODES
              value: "{{.RedisName}}-0.{{.RedisName}}-headless {{.RedisName}}-1.{{.RedisName}}-headless {{.RedisName}}-2.{{.RedisName}}-headless {{.RedisName}}-3.{{.RedisName}}-headless {{.RedisName}}-4.{{.RedisName}}-headless {{.RedisName}}-5.{{.RedisName}}-headless "
            - name: REDISCLI_AUTH
              valueFrom:
                secretKeyRef:
                  name: redis-cluster
                  key: redis-password
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-cluster
                  key: redis-password
            - name: REDIS_AOF_ENABLED
              value: "yes"
            - name: REDIS_TLS_ENABLED
              value: "no"
            - name: REDIS_PORT
              value: "6379"
          ports:
            - name: tcp-redis
              containerPort: 6379
            - name: tcp-redis-bus
              containerPort: 16379
          livenessProbe:
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 6
            successThreshold: 1
            failureThreshold: 5
            exec:
              command:
                - sh
                - -c
                - /scripts/ping_liveness_local.sh 5
          readinessProbe:
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 2
            successThreshold: 1
            failureThreshold: 5
            exec:
              command:
                - sh
                - -c
                - /scripts/ping_readiness_local.sh 1
          resources:
            limits: {}
            requests: {}
          volumeMounts:
            - name: scripts
              mountPath: /scripts
            - name: default-config
              mountPath: /opt/bitnami/redis/etc/redis-default.conf
              subPath: redis-default.conf
            - name: redis-tmp-conf
              mountPath: /opt/bitnami/redis/etc/
      volumes:
        - name: scripts
          configMap:
            name: redis-cluster-scripts
            defaultMode: 0755
        - name: default-config
          configMap:
            name: redis-cluster-default
        - name: redis-tmp-conf
          emptyDir: {}`
)

func InstallStandalone(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	var data = templateData{
		Namespace:     namespace,
		RedisName:     name,
		RedisPassword: password,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisStandaloneTemplates)
}

func RemoveStandalone(t *testing.T, kc *kubernetes.Clientset, name, namespace string) {
	var data = templateData{
		Namespace: namespace,
		RedisName: name,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisStandaloneTemplates)
	helper.DeleteNamespace(t, kc, namespace)
}

func InstallSentinel(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	_, err := helper.ExecuteCommand("helm repo add bitnami https://charts.bitnami.com/bitnami")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set sentinel.enabled=true --set master.persistence.enabled=false --set replica.persistence.enabled=false --set global.redis.password=%s bitnami/redis`,
		name,
		namespace,
		password))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func RemoveSentinel(t *testing.T, kc *kubernetes.Clientset, name, namespace string) {
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --timeout 900s %s --namespace %s`,
		name,
		namespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	helper.DeleteNamespace(t, kc, namespace)
}

func InstallCluster(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	var data = templateData{
		Namespace:     namespace,
		RedisName:     name,
		RedisPassword: password,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisClusterTemplates)
	assert.True(t, helper.WaitForStatefulsetReplicaReadyCount(t, kc, name, namespace, 6, 60, 3),
		"redis-cluster should be up")
}

func RemoveCluster(t *testing.T, kc *kubernetes.Clientset, name, namespace string) {
	var data = templateData{
		Namespace: namespace,
		RedisName: name,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisClusterTemplates)
	helper.DeleteNamespace(t, kc, namespace)
}
