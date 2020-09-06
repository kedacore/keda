module github.com/kedacore/keda

go 1.15

require (
	cloud.google.com/go v0.62.0
	github.com/Azure/azure-amqp-common-go/v3 v3.0.1
	github.com/Azure/azure-event-hubs-go v1.3.1
	github.com/Azure/azure-sdk-for-go v46.0.0+incompatible
	github.com/Azure/azure-service-bus-go v0.10.6
	github.com/Azure/azure-storage-blob-go v0.10.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.1
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.27.0
	github.com/aws/aws-sdk-go v1.34.11
	github.com/go-logr/logr v0.1.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.11
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20200618121405-54026617ec44
	github.com/lib/pq v1.8.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.1
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	google.golang.org/api v0.29.0
	google.golang.org/genproto v0.0.0-20200731012542-8145dea6a485
	google.golang.org/grpc v1.31.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.8
	k8s.io/klog v1.0.0
	k8s.io/metrics v0.18.8
	knative.dev/pkg v0.0.0-20200810223505-473bba04ee7f
	sigs.k8s.io/controller-runtime v0.6.2
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

// WORKAROUND - we can remove this once k8s v1.18+ is present in knative/pkg
replace knative.dev/pkg => github.com/zroubalik/pkg v0.0.0-20200714090639-88ee0a9b8a22
