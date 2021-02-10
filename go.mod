module github.com/kedacore/keda/v2

go 1.15

require (
	cloud.google.com/go v0.73.0
	github.com/Azure/azure-amqp-common-go/v3 v3.1.0
	github.com/Azure/azure-event-hubs-go/v3 v3.3.4
	github.com/Azure/azure-sdk-for-go v48.2.2+incompatible
	github.com/Azure/azure-service-bus-go v0.10.7
	github.com/Azure/azure-storage-blob-go v0.11.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.11.15
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.5
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.27.2
	github.com/aws/aws-sdk-go v1.36.19
	github.com/denisenkom/go-mssqldb v0.9.0 // indirect
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.3.0 // indirect
	github.com/go-openapi/spec v0.20.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.11
	github.com/influxdata/influxdb-client-go/v2 v2.2.1
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20201216091021-1b9fa998bbaa
	github.com/lib/pq v1.9.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.7
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	go.mongodb.org/mongo-driver v1.1.2
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20201203001206-6486ece9c497
	google.golang.org/grpc v1.34.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/apiserver v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.20.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/metrics v0.20.0
	knative.dev/pkg v0.0.0-20201019114258-95e9532f0457
	sigs.k8s.io/controller-runtime v0.6.4
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2

// adapter uses k8s.io/apiserver/pkg/server, which indirectly uses go.etcd.io/etcd/proxy/grpcproxy.
// etcd is not compatible with newer grpc version, see here https://github.com/etcd-io/etcd/issues/12124
// so until that is fixed, we will pin the grpc version to v1.29.1
replace google.golang.org/grpc => google.golang.org/grpc v1.29.1
