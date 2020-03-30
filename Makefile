# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

OS = $(shell uname | tr A-Z a-z)

# Project variables
BUILD_PACKAGE ?= ./cmd/cloudinfo
BINARY_NAME ?= cloudinfo
DOCKER_IMAGE = banzaicloud/cloudinfo

# Build variables
BUILD_DIR ?= build
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
ifeq ($(filter -v,${GOARGS}),)
	GOARGS += -v
endif
TEST_FORMAT = short-verbose
endif

# Docker variables
DOCKER_TAG ?= ${VERSION}

MISSPELL_VERSION = 0.3.4
GQLGEN_VERSION = 0.8.3
PACKR_VERSION = 2.2.0

GOLANGCI_VERSION = 1.24.0
LICENSEI_VERSION = 0.2.0
OPENAPI_GENERATOR_VERSION = v4.1.3
GOTESTSUM_VERSION = 0.3.5
GOBIN_VERSION = 0.0.13
PROTOTOOL_VERSION = 1.8.0
PROTOC_GEN_GO_VERSION = 1.3.2
MGA_VERSION = 0.0.10

GOLANG_VERSION = 1.13
SWAGGER_VERSION = 0.21.0

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

SWAGGER_PI_TMP_FILE = ./api/openapi-spec/cloudinfo.json
SWAGGER_PI_FILE = ./api/openapi-spec/cloudinfo.yaml

## include "generic" targets
include main-targets.mk


.PHONY: swagger2openapi
swagger2openapi:
ifeq ($(shell which swagger2openapi),)
	npm install -g swagger2openapi
endif

bin/swagger: bin/swagger-${SWAGGER_VERSION}
	@ln -sf swagger-${SWAGGER_VERSION} bin/swagger
bin/swagger-${SWAGGER_VERSION}: bin/gobin
	@mkdir -p bin
	GOBIN=bin/ bin/gobin github.com/go-swagger/go-swagger/cmd/swagger@v${SWAGGER_VERSION}
	@mv bin/swagger bin/swagger-${SWAGGER_VERSION}

.PHONY: swagger
swagger: bin/swagger
	bin/swagger generate spec -m -o $(SWAGGER_PI_TMP_FILE)
	swagger2openapi -y $(SWAGGER_PI_TMP_FILE) > $(SWAGGER_PI_FILE)

bin/packr2: bin/packr2-${PACKR_VERSION}
	@ln -sf packr2-${PACKR_VERSION} bin/packr2
bin/packr2-${PACKR_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/gobuffalo/packr/releases/download/v${PACKR_VERSION}/packr_${PACKR_VERSION}_${OS}_amd64.tar.gz | tar -zOxf - packr2 > ./bin/packr2-${PACKR_VERSION} && chmod +x ./bin/packr2-${PACKR_VERSION}

.PHONY: uibundle
uibundle: bin/packr2
	GO111MODULE=on cd cmd/cloudinfo && $(abspath bin/packr2)

define generate_openapi_client
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ${3}; else rm -rf ${3}; fi
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=${2} \
	--additional-properties withGoCodegenComment=true \
	-i /local/${1} \
	-g go \
	-o /local/${3}
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) ${3}; fi
	rm ${3}/{.travis.yml,git_push.sh,go.*}
endef

api/openapi-spec/cloudinfo.yaml:swagger

.PHONY: generate-cloudinfo-client
generate-cloudinfo-client: api/openapi-spec/cloudinfo.yaml ## Generate client from Cloudinfo OpenAPI spec
	$(call generate_openapi_client,api/openapi-spec/cloudinfo.yaml,cloudinfo,.gen/cloudinfo-client)
