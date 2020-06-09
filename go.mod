module github.com/banzaicloud/cloudinfo

go 1.14

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	emperror.dev/emperror v0.32.0
	emperror.dev/errors v0.7.0
	emperror.dev/handler/logur v0.4.0
	github.com/99designs/gqlgen v0.8.3
	github.com/Azure/azure-sdk-for-go v33.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.1
	github.com/Azure/go-autorest/autorest/azure/auth v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.267
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/aws/aws-sdk-go v1.27.0
	github.com/banzaicloud/go-gin-prometheus v0.0.0-20190121125239-fa3b20bd0ba9
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/digitalocean/godo v1.37.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.6.3
	github.com/go-kit/kit v0.10.0
	github.com/go-openapi/runtime v0.18.0
	github.com/gobuffalo/logger v1.0.3 // indirect
	github.com/gobuffalo/packd v1.0.0 // indirect
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/gocql/gocql v0.0.0-20190402132108-0e1d5de854df
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/mitchellh/mapstructure v1.3.2
	github.com/moogar0880/problems v0.1.1
	github.com/oracle/oci-go-sdk v12.5.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/common v0.10.0
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/sagikazarmark/viperx v0.4.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/vektah/gqlparser v1.1.2
	go.opencensus.io v0.22.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200308013534-11ec41452d41 // indirect
	google.golang.org/api v0.13.0
	gopkg.in/go-playground/validator.v8 v8.18.2
	logur.dev/adapter/logrus v0.5.0
	logur.dev/logur v0.16.2
)

replace (
	// Kubernetes 1.13.5
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
)
