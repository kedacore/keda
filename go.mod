module github.com/kedacore/keda/v2

go 1.16

require (
	cloud.google.com/go v0.97.0
	cloud.google.com/go/monitoring v1.1.0
	github.com/Azure/azure-amqp-common-go/v3 v3.2.2
	github.com/Azure/azure-event-hubs-go/v3 v3.3.16
	github.com/Azure/azure-sdk-for-go v59.4.0+incompatible
	github.com/Azure/azure-service-bus-go v0.11.5
	github.com/Azure/azure-storage-blob-go v0.14.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.11.22
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.9
	github.com/Huawei/gophercloud v1.0.21
	github.com/Shopify/sarama v1.30.0
	github.com/aws/aws-sdk-go v1.42.16
	github.com/denisenkom/go-mssqldb v0.11.0
	github.com/elastic/go-elasticsearch/v7 v7.15.1
	github.com/go-logr/logr v0.4.0
	github.com/go-playground/assert/v2 v2.0.1
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gocql/gocql v0.0.0-20211015133455-b225f9b53fa1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/vault/api v1.3.0
	github.com/imdario/mergo v0.3.12
	github.com/influxdata/influxdb-client-go/v2 v2.6.0
	github.com/lib/pq v1.10.4
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/robfig/cron/v3 v3.0.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.12.1
	github.com/xdg/scram v1.0.3
	go.mongodb.org/mongo-driver v1.8.0
	google.golang.org/api v0.60.0
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/apiserver v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/code-generator v0.22.4
	k8s.io/klog/v2 v2.10.0
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	k8s.io/metrics v0.22.4
	knative.dev/pkg v0.0.0-20211123135150-787aec59e70a
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/custom-metrics-apiserver v1.22.0
)

// Needed for CVE-2020-28483 https://github.com/advisories/GHSA-h395-qcrw-5vmq
// we need version github.com/gin-gonic/gin >= 1.7.0
replace github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.3

require (
	github.com/dysnix/ai-scale-libs v0.0.0-20211217063709-f888549b2d75
	github.com/dysnix/ai-scale-proto v0.0.0-20211216191415-a75682995da1
	github.com/go-playground/validator/v10 v10.9.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/prometheus/common v0.32.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/xdg/stringprep v1.0.3 // indirect
	github.com/xhit/go-str2duration/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
)
