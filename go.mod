module github.com/kedacore/keda

go 1.13

require (
	bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c
	cloud.google.com/go v0.40.0
	github.com/Azure/azure-amqp-common-go v1.1.4
	github.com/Azure/azure-event-hubs-go v1.3.1
	github.com/Azure/azure-pipeline-go v0.1.8
	github.com/Azure/azure-sdk-for-go v21.4.0+incompatible
	github.com/Azure/azure-service-bus-go v0.4.1
	github.com/Azure/azure-storage-blob-go v0.6.0
	github.com/Azure/azure-storage-queue-go v0.0.0-20181215014128-6ed74e755687
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Azure/go-autorest v11.1.1+incompatible
	github.com/DataDog/zstd v1.3.5
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PuerkitoBio/purell v1.1.1
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/Shopify/sarama v1.21.0
	github.com/apache/thrift v0.12.0 // indirect
	github.com/aws/aws-sdk-go v1.19.27
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.8+incompatible
	github.com/coreos/go-semver v0.2.0
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/docker v1.13.1
	github.com/eapache/go-resiliency v1.1.0
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21
	github.com/eapache/queue v1.1.0
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/emicklei/go-restful v2.9.1+incompatible
	github.com/emicklei/go-restful-swagger12 v0.0.0-20170208215640-dcef7f557305
	github.com/evanphx/json-patch v3.0.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/jsonpointer v0.19.0
	github.com/go-openapi/jsonreference v0.19.0
	github.com/go-openapi/spec v0.19.0
	github.com/go-openapi/swag v0.19.0
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gogo/protobuf v1.2.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.3.1-0.20190508161146-9fa652df1129
	github.com/golang/protobuf v1.3.1
	github.com/golang/snappy v0.0.1
	github.com/google/btree v1.0.0
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.2.0
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/golang-lru v0.5.1
	github.com/imdario/mergo v0.3.7
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/jpillora/backoff v0.0.0-20180909062703-3050d21c67d7
	github.com/json-iterator/go v1.1.6
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20181126213231-bb8bae16c555
	github.com/mailru/easyjson v0.0.0-20190312143242-1de009706dbe
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mitchellh/mapstructure v1.1.2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742
	github.com/opentracing/opentracing-go v1.1.0
	github.com/openzipkin/zipkin-go v0.1.6 // indirect
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709
	github.com/petar/GoLLRB v0.0.0-20130427215148-53be0d36a84c
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.2.0
	github.com/prometheus/procfs v0.0.0-20190322151404-55ae3d9d5573
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/sirupsen/logrus v1.4.0
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/pflag v1.0.3
	github.com/streadway/amqp v0.0.0-20190404075320-75d898a42a94
	github.com/stretchr/testify v1.3.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v1.1.1
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.opencensus.io v0.21.0
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c
	golang.org/x/net v0.0.0-20190503192946-f4e77d36d62c
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190507160741-ecd444e8653b
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools v0.0.0-20190506145303-2d16b83fe98c
	google.golang.org/api v0.7.0
	google.golang.org/appengine v1.5.0
	google.golang.org/genproto v0.0.0-20190530194941-fb225487d101
	google.golang.org/grpc v1.21.1
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	k8s.io/api v0.0.0-20181126151915-b503174bad59
	k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc
	k8s.io/apiserver v0.0.0-20181207191401-9601a7bf41ef
	k8s.io/client-go v0.0.0-20190228133956-77e032213d34
	k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo v0.0.0-20190319205223-bc9033e9ec9e
	k8s.io/klog v0.2.0
	k8s.io/kube-openapi v0.0.0-20180719232738-d8ea2fe547a4
	k8s.io/metrics v0.0.0-20181128195641-3954d62a524d
	pack.ag/amqp v0.11.0
)
