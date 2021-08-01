module github.com/kedacore/keda/v2

go 1.15

require (
	cloud.google.com/go v0.86.0
	github.com/Azure/azure-amqp-common-go/v3 v3.1.0
	github.com/Azure/azure-event-hubs-go/v3 v3.3.10
	github.com/Azure/azure-sdk-for-go v55.8.0+incompatible
	github.com/Azure/azure-service-bus-go v0.10.13
	github.com/Azure/azure-storage-blob-go v0.14.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.29.1
	github.com/aws/aws-sdk-go v1.39.4
	github.com/denisenkom/go-mssqldb v0.10.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/go-openapi/spec v0.20.3
	github.com/go-playground/assert/v2 v2.0.1
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/vault/api v1.1.0
	github.com/imdario/mergo v0.3.12
	github.com/influxdata/influxdb-client-go/v2 v2.4.0
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20210706132418-63a83be59f5f
	github.com/lib/pq v1.10.2
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/robfig/cron/v3 v3.0.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.8.1
	github.com/xdg/scram v1.0.3
	go.mongodb.org/mongo-driver v1.5.4
	google.golang.org/api v0.50.0
	google.golang.org/genproto v0.0.0-20210708141623-e76da96a951f
	google.golang.org/grpc v1.39.0
	k8s.io/api v0.20.8
	k8s.io/apimachinery v0.20.8
	k8s.io/apiserver v0.20.8
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.20.8
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-openapi v0.0.0-20210305164622-f622666832c1
	k8s.io/metrics v0.20.8
	knative.dev/pkg v0.0.0-20210708145023-4a3e56dc13b2
	sigs.k8s.io/controller-runtime v0.6.5
)

replace k8s.io/client-go => k8s.io/client-go v0.20.8

// adapter uses k8s.io/apiserver/pkg/server, which indirectly uses go.etcd.io/etcd/proxy/grpcproxy.
// etcd is not compatible with newer grpc version, see here https://github.com/etcd-io/etcd/issues/12124
// so until that is fixed, we will pin the grpc version to v1.29.1
replace google.golang.org/grpc => google.golang.org/grpc v1.29.1
