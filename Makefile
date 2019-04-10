# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

OS = $(shell uname -s)

# Project variables
BUILD_PACKAGE ?= ./cmd/cloudinfo
BINARY_NAME ?= cloudinfo
DOCKER_IMAGE = banzaicloud/cloudinfo

# Build variables
BUILD_DIR ?= build
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

# Docker variables
DOCKER_TAG ?= ${VERSION}

GOTESTSUM_VERSION = 0.3.3
GOLANGCI_VERSION = 1.15.0
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.1.0
OPENAPI_GENERATOR_VERSION = 3.3.0

GOLANG_VERSION = 1.12

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

SWAGGER_PI_TMP_FILE = ./api/openapi-spec/cloudinfo.json
SWAGGER_PI_FILE = ./api/openapi-spec/cloudinfo.yaml

## include "generic" targets
include main-targets.mk

deps-swagger:
ifeq ($(shell which swagger),)
	go get -u github.com/go-swagger/go-swagger/cmd/swagger
endif
ifeq ($(shell which swagger2openapi),)
	npm install -g swagger2openapi
endif

deps: deps-swagger
	go get ./...


swagger:
	swagger generate spec -m -b ./cmd/cloudinfo -o $(SWAGGER_PI_TMP_FILE)
	swagger2openapi -y $(SWAGGER_PI_TMP_FILE) > $(SWAGGER_PI_FILE)

generate-pi-client:
	swagger generate client -f $(SWAGGER_PI_TMP_FILE) -A cloudinfo -t pkg/cloudinfo-client/
