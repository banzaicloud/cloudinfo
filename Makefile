# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

SHELL = /bin/bash
OS = $(shell uname -s)

# Project variables
PACKAGE = github.com/banzaicloud/cloudinfo
BINARY_NAME = cloudinfo
SHELL = /bin/bash

# Build variables
BUILD_DIR ?= build
BUILD_PACKAGE = ${PACKAGE}/cmd/cloudinfo
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

DEP_VERSION = 0.5.0
GOTESTSUM_VERSION = 0.3.2
GOLANGCI_VERSION = 1.15.0
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.0.7
OPENAPI_GENERATOR_VERSION = 3.3.0

GOLANG_VERSION = 1.11

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")


SWAGGER_PI_TMP_FILE = ./api/openapi-spec/cloudinfo.json
SWAGGER_PI_FILE = ./api/openapi-spec/cloudinfo.yaml

DOCKER_TAG ?= ${VERSION}
DOCKER_IMAGE = $(shell echo ${PACKAGE} | cut -d '/' -f 2,3)

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


## starts the cloudinfo app with docker-compose
pi-start:
	docker-compose -f docker-compose.yml up -d

## stops the cloudinfo app with docker-compose
pi-stop:
	docker-compose -f docker-compose.yml stop
