module github.com/kedacore/keda

go 1.13

require (
	cloud.google.com/go v0.50.0
	github.com/Azure/azure-amqp-common-go/v3 v3.0.0
	github.com/Azure/azure-event-hubs-go v1.3.1
	github.com/Azure/azure-sdk-for-go v41.1.0+incompatible
	github.com/Azure/azure-service-bus-go v0.10.0
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.10.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.26.1
	github.com/aws/aws-sdk-go v1.30.3
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.7
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.3.5
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/imdario/mergo v0.3.9
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20200323093244-5046ce1afe6b
	github.com/lib/pq v1.3.0
	github.com/operator-framework/operator-sdk v0.16.1-0.20200402200254-b448429687fd
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71
	github.com/stretchr/testify v1.5.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d // indirect
	google.golang.org/api v0.14.0
	google.golang.org/genproto v0.0.0-20191115194625-c23dd37a84c9
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/metrics v0.17.4
	knative.dev/pkg v0.0.0-20200404181734-92cdec5b3593
	pack.ag/amqp v0.12.5 // indirect
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/apiserver => k8s.io/apiserver v0.17.4 // Required by kubernetes-incubator/custom-metrics-apiserver
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)

// Required to resolve go/grpc issues
replace (
	cloud.google.com/go => cloud.google.com/go v0.46.3
	google.golang.org/api => google.golang.org/api v0.10.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20191002211648-c459b9ce5143
	google.golang.org/grpc => google.golang.org/grpc v1.24.0
)
