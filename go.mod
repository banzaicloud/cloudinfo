module github.com/banzaicloud/cloudinfo

go 1.14

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	emperror.dev/emperror v0.33.0
	emperror.dev/errors v0.8.0
	emperror.dev/handler/logur v0.4.0
	github.com/99designs/gqlgen v0.8.3
	github.com/Azure/azure-sdk-for-go v45.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.2
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.0
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.545
	github.com/asaskevich/EventBus v0.0.0-20200428142821-4fc0642a29f3
	github.com/aws/aws-sdk-go v1.33.19
	github.com/banzaicloud/go-gin-prometheus v0.1.0
	github.com/digitalocean/godo v1.42.0
	github.com/gin-contrib/cors v0.0.0-20170318125340-cf4846e6a636
	github.com/gin-contrib/static v0.0.0-20181225054800-cf5e10bbd933
	github.com/gin-gonic/gin v1.4.0
	github.com/go-kit/kit v0.10.0
	github.com/gocql/gocql v0.0.0-20200624222514-34081eda590e
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/markbates/pkger v0.17.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/moogar0880/problems v0.1.1
	github.com/oracle/oci-go-sdk v23.0.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.11.1
	github.com/sagikazarmark/viperx v0.8.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vektah/gqlparser v1.1.2
	go.opencensus.io v0.22.4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.30.0
	gopkg.in/go-playground/validator.v8 v8.18.2
	logur.dev/adapter/logrus v0.5.0
	logur.dev/logur v0.17.0
	sigs.k8s.io/controller-runtime v0.5.2 // indirect
)

replace (
	// Kubernetes 1.17.2
	k8s.io/api => k8s.io/api v0.17.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
)
