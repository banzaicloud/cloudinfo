# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html


.PHONY: clear
clear: ## Clear the working area and the project
	rm -rf bin/ vendor/

.PHONY: clean
clean: ## Clean builds
	rm -rf ${BUILD_DIR}/

bin/dep: bin/dep-${DEP_VERSION}
	@ln -sf dep-${DEP_VERSION} bin/dep
bin/dep-${DEP_VERSION}:
	@mkdir -p bin
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | INSTALL_DIRECTORY=bin DEP_RELEASE_TAG=v${DEP_VERSION} sh
	@mv bin/dep $@

.PHONY: vendor
vendor: bin/dep ## Install dependencies
	bin/dep ensure -v -vendor-only

.PHONY: build-release
build-release: ## Build a binary without debug information
	@${MAKE} LDFLAGS="-w ${LDFLAGS}" build

.PHONY: build-debug
build-debug: ## Build a binary with remote debugging capabilities
	@${MAKE} GOARGS="${GOARGS} -gcflags \"all=-N -l\"" BINARY_NAME="${BINARY_NAME}-debug" build

.PHONY: build
build: ## Build a binary
ifneq (${IGNORE_GOLANG_VERSION_REQ}, 1)
	@printf "${GOLANG_VERSION}\n$$(go version | awk '{sub(/^go/, "", $$3);print $$3}')" | sort -t '.' -k 1,1 -k 2,2 -k 3,3 -g | head -1 | grep -q -E "^${GOLANG_VERSION}$$" || (printf "Required Go version is ${GOLANG_VERSION}\nInstalled: `go version`" && exit 1)
endif
	go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME} ${BUILD_PACKAGE}

.PHONY: docker
docker: ## Build a Docker image
ifneq (${DOCKER_PREBUILT}, 1)
	@${MAKE} BINARY_NAME="${BINARY_NAME}-docker" GOOS=linux build-release
endif
	docker build --build-arg BUILD_DIR=${BUILD_DIR} --build-arg BINARY_NAME=${BINARY_NAME}-docker -t ${DOCKER_IMAGE}:${DOCKER_TAG} -f Dockerfile .
ifeq (${DOCKER_LATEST}, 1)
	docker tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:latest
endif


.PHONY: docker-build
docker-build: ## Builds go binary in docker image
	docker run -it -v $(PWD):/go/src/${PACKAGE} -w /go/src/${PACKAGE} golang:${GOLANG_VERSION}-alpine go build -o ${BINARY_NAME}_linux ${BUILD_PACKAGE}

.PHONY: debug
debug: GOARGS += -gcflags "-N -l"
debug: BINARY_NAME := ${BINARY_NAME}-debug
debug: build ## Builds binary package

.PHONY: debug-docker
debug-docker: debug ## Builds binary package
	docker build -t banzaicloud/${BINARY_NAME}:debug -f Dockerfile.dev .

.PHONY: check
check: test lint ## Run tests and linters

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	@bin/golangci-lint run -v

.PHONY: fmt
fmt:
	@gofmt -s -w ${GOFILES_NOVENDOR}

bin/misspell: bin/misspell-${MISSPELL_VERSION}
	@ln -sf misspell-${MISSPELL_VERSION} bin/misspell
bin/misspell-${MISSPELL_VERSION}:
	@mkdir -p bin
	curl -sfL https://git.io/misspell | bash -s -- -b ./bin/ v${MISSPELL_VERSION}
	@mv bin/misspell $@

.PHONY: misspell
misspell: bin/misspell ## Fix spelling mistakes
	misspell -w ${GOFILES_NOVENDOR}

bin/licensei: bin/licensei-${LICENSEI_VERSION}
	@ln -sf licensei-${LICENSEI_VERSION} bin/licensei
bin/licensei-${LICENSEI_VERSION}:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s v${LICENSEI_VERSION}
	@mv bin/licensei $@

.PHONY: license-check
license-check: bin/licensei ## Run license check
	bin/licensei check
	./scripts/check-header.sh

.PHONY: license-cache
license-cache: bin/licensei ## Generate license cache
	bin/licensei cache


bin/gotestsum: bin/gotestsum-${GOTESTSUM_VERSION}
	@ln -sf gotestsum-${GOTESTSUM_VERSION} bin/gotestsum
bin/gotestsum-${GOTESTSUM_VERSION}:
	@mkdir -p bin
ifeq (${OS}, Darwin)
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_darwin_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum-${GOTESTSUM_VERSION} && chmod +x ./bin/gotestsum-${GOTESTSUM_VERSION}
endif
ifeq (${OS}, Linux)
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum-${GOTESTSUM_VERSION} && chmod +x ./bin/gotestsum-${GOTESTSUM_VERSION}
endif

TEST_PKGS ?= ./...
TEST_REPORT_NAME ?= results.xml
.PHONY: test
test: TEST_REPORT ?= main
test: TEST_FORMAT ?= short
test: SHELL = /bin/bash
test: bin/gotestsum ## Run tests
	@mkdir -p ${BUILD_DIR}/test_results/${TEST_REPORT}
	bin/gotestsum --no-summary=skipped --junitfile ${BUILD_DIR}/test_results/${TEST_REPORT}/${TEST_REPORT_NAME} --format ${TEST_FORMAT} -- $(filter-out -v,${GOARGS}) $(if ${TEST_PKGS},${TEST_PKGS},./...)

.PHONY: test-all
test-all: ## Run all tests
	@${MAKE} GOARGS="${GOARGS} -run .\*" TEST_REPORT=all test

.PHONY: test-integration
test-integration: ## Run integration tests
	@${MAKE} GOARGS="${GOARGS} -run ^TestIntegration\$$\$$" TEST_REPORT=integration test


.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)
