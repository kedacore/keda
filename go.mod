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
	github.com/Azure/go-autorest/autorest v0.11.13
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.5
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.27.2
	github.com/aws/aws-sdk-go v1.36.12
	github.com/go-logr/logr v0.1.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.11
	github.com/influxdata/influxdb-client-go/v2 v2.2.0
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20200618121405-54026617ec44
	github.com/lib/pq v1.9.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.4
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20200904004341-0bd0a958aa1d
	google.golang.org/grpc v1.34.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.8
	k8s.io/klog v1.0.0
	k8s.io/metrics v0.18.8
	knative.dev/pkg v0.0.0-20201019114258-95e9532f0457
	sigs.k8s.io/controller-runtime v0.6.4
)

replace (
	k8s.io/apiserver => k8s.io/apiserver v0.18.8 // Required by kubernetes-incubator/custom-metrics-apiserver
	k8s.io/client-go => k8s.io/client-go v0.18.8
)

// Required to resolve go/grpc issues
// (grpc version needed by k8s.io/apiserver vs kubernetes-incubator/custom-metrics-apiserver)
replace (
	cloud.google.com/go => cloud.google.com/go v0.48.0
	google.golang.org/api => google.golang.org/api v0.15.1
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20191002211648-c459b9ce5143
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

// Required for k8s.io/apiserver, using k8s.io/kube-openapi branch release-1.18
replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
