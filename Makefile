# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

OS = $(shell uname -s)

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

GOTESTSUM_VERSION = 0.3.3
GOLANGCI_VERSION = 1.15.0
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.1.0
OPENAPI_GENERATOR_VERSION = 3.3.0
GOBIN_VERSION = 0.0.9
GQLGEN_VERSION = 0.8.3

GOLANG_VERSION = 1.12
SWAGGER_VERSION = 0.19.0

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


generate-pi-client:
	swagger generate client -f $(SWAGGER_PI_TMP_FILE) -A cloudinfo -t pkg/cloudinfo-client/

bin/swagger: bin/swagger-${SWAGGER_VERSION}
	@ln -sf swagger-${SWAGGER_VERSION} bin/swagger
bin/swagger-${SWAGGER_VERSION}: bin/gobin
	@mkdir -p bin
	GOBIN=bin/ bin/gobin github.com/go-swagger/go-swagger/cmd/swagger@v${SWAGGER_VERSION}
	@mv bin/swagger bin/swagger-${SWAGGER_VERSION}

.PHONY: swagger
swagger: bin/swagger
	GO111MODULE="off" bin/swagger generate spec -m -b ./cmd/cloudinfo -o $(SWAGGER_PI_TMP_FILE)
	GO111MODULE="off" swagger2openapi -y $(SWAGGER_PI_TMP_FILE) > $(SWAGGER_PI_FILE)


