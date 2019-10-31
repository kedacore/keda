module github.com/kedacore/keda

go 1.13.1

// Required deps for operator-sdk v0.11.0 <-> kubernetes-incubator/custom-metrics-apiserver on kubernetes-1.14.1
replace (
	github.com/kubernetes-incubator/custom-metrics-apiserver => github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20190703094830-abe433176c52
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	github.com/ugorji/go => github.com/ugorji/go v1.1.7

	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190302045857-e85c7b244fd2

)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190409021813-1ec86e4da56c
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190409022812-850dadb8b49c
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1
	// Pinned to v2.10.0 (kubernetes-1.14.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20190525122359-d20e84d0fb64
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.2
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.11.0

require (
	cloud.google.com/go v0.46.3
	github.com/Azure/azure-event-hubs-go v1.3.1
	github.com/Azure/azure-service-bus-go v0.9.1
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20190416192124-a17745f1cdbf
	github.com/Azure/go-autorest v12.0.0+incompatible
	github.com/Shopify/sarama v1.23.1
	github.com/aws/aws-sdk-go v1.25.6
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/imdario/mergo v0.3.8
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20190918110929-3d9be26a50eb
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/operator-framework/operator-sdk v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/streadway/amqp v0.0.0-20190827072141-edfb9018d271
	github.com/stretchr/testify v1.4.0
	google.golang.org/api v0.10.0
	google.golang.org/genproto v0.0.0-20191002211648-c459b9ce5143
	google.golang.org/grpc v1.24.0
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20191014065749-fb3eea214746
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/component-base v0.0.0-20191014071552-ca590c444ad5
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20190401085232-94e1e7b7574c
	k8s.io/metrics v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/structured-merge-diff v0.0.0-20191009170950-ae447d53f5c3 // indirect

)
