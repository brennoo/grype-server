e_Y=\033[1;33m
C_C=\033[0;36m
C_M=\033[0;35m
C_R=\033[0;41m
C_N=\033[0m
SHELL=/bin/bash

# Project variables
BINARY_NAME ?= grype-server
DOCKER_REGISTRY ?= brennoo
VERSION ?= $(shell git rev-parse HEAD)
DOCKER_IMAGE ?= $(DOCKER_REGISTRY)/$(BINARY_NAME)
DOCKER_TAG ?= ${VERSION}
PLATFORMS ?= linux/amd64,linux/arm64
BASE_IMAGE ?= dhi.io/alpine-base:3.24

DOCKER_BUILD_ARGS := \
	--build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
	--build-arg IMAGE_VERSION=${VERSION} \
	--build-arg BASE_IMAGE=$(BASE_IMAGE)

# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

.PHONY: build
build: ## Build Grype Server
	@(echo "Building Grype Server ..." )
	@(cd grype-server && go mod tidy && CGO_ENABLED=0 go build -o bin/grype-server cmd/grype-server/main.go && ls -l bin/)


.PHONY: docker
docker: ## Build Grype Server docker image
	@(echo "Building Grype Server docker image [${DOCKER_IMAGE}:${DOCKER_TAG}] ..." )
	@(docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} . $(DOCKER_BUILD_ARGS))

.PHONY: docker-push
docker-push: docker ## Build Grype Server docker image and push it to remote
	@(echo "Pushing Grype Server docker image [${DOCKER_IMAGE}:${DOCKER_TAG}] ..." )
	@(docker push ${DOCKER_IMAGE}:${DOCKER_TAG})

.PHONY: docker-multiarch
docker-multiarch: ## Build multi-arch (linux/amd64,linux/arm64) image, kept in the buildx cache (not pushed)
	@(echo "Building multi-arch Grype Server image [${DOCKER_IMAGE}:${DOCKER_TAG}] ..." )
	@(docker buildx build --platform ${PLATFORMS} \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		$(DOCKER_BUILD_ARGS) \
		.)

.PHONY: push-multiarch
push-multiarch: ## Push the multi-arch image built by docker-multiarch to remote (reuses the buildx cache)
	@(echo "Pushing multi-arch Grype Server image [${DOCKER_IMAGE}:${DOCKER_TAG}] ..." )
	@(docker buildx build --platform ${PLATFORMS} \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		--push \
		$(DOCKER_BUILD_ARGS) \
		.)

.PHONY: api
api: ## Generating API code
	@(echo "Generating API code ..." )
	@(cd api; ./generate.sh)

.PHONY: test
test: ## Run Unit Tests
	@(cd grype-server && go test ./pkg/...)

.PHONY: lint
lint: ## Run golangci-lint
	@(cd grype-server && golangci-lint run ./...)

.PHONY: check
check: test lint ## Run tests and linters

.PHONY: clean
clean: ## Clean all build artifacts
	@(rm -rf grype-server/bin/* ; echo "Build artifacts cleanup done" )

