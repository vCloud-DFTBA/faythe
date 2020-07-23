SHELL = /usr/bin/env bash

.DEFAULT_GOAL := all
.PHONY: all check-mod clean build

#############
# Variables #
#############

# When the value of empty, no -mod parameter will be passed to go.
# For Go 1.13, "readonly" and "vendor" can be used here.
# In Go 1.14, "vendor" and "mod" can be used here.
GOMOD?=vendor
ifeq ($(strip $(GOMOD)),) # Is empty?
	MOD_FLAG=
	GOLANGCI_ARG=
else
	MOD_FLAG=-mod=$(GOMOD)
	GOLANGCI_ARG=--modules-download-mode=$(GOMOD)
endif

# Docker image info
DOCKER_IMAGE_NAMESPACE       ?= kiennt26
DOCKER_IMAGE_NAME            ?= faythe
DOCKER_IMAGE_TAG             ?= $(shell ./tools/image-tag)
DOCKER_LOCAL_REGISTRY        ?= ""
ifeq ($(DOCKER_LOCAL_REGISTRY), "")
	DOCKER_IMAGE_FULL        ?= $(DOCKER_IMAGE_NAMESPACE)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	DOCKER_IMAGE_FULL_LATEST ?= $(DOCKER_IMAGE_NAMESPACE)/$(DOCKER_IMAGE_NAME):latest
else
	DOCKER_IMAGE_FULL        ?= $(DOCKER_LOCAL_REGISTRY)/$(DOCKER_IMAGE_NAMESPACE)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	DOCKER_IMAGE_FULL_LATEST ?= $(DOCKER_LOCAL_REGISTRY)/$(DOCKER_IMAGE_NAMESPACE)/$(DOCKER_IMAGE_NAME):latest
endif

# Version info for binaries
GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH   := $(shell git rev-parse --abbrev-ref HEAD)

# Build flags
VPREFIX    := github.com/vCloud-DFTBA/faythe/pkg/build
GO_LDFLAGS := -X $(VPREFIX).Branch=$(GIT_BRANCH) -X $(VPREFIX).Version=$(DOCKER_IMAGE_TAG) \
			  -X $(VPREFIX).Revision=$(GIT_REVISION) -X $(VPREFIX).BuildUser=$(shell whoami)@$(shell hostname) \
			  -X $(VPREFIX).BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_FLAGS   := -ldflags "-s -w $(GO_LDFLAGS)" $(MOD_FLAG)
# Output directory
GO_OUT     := cmd/faythe

build: cmd/faythe/main.go
	go build $(GO_FLAGS) -o $(GO_OUT) ./...

install:
	go install $(GO_FLAGS) ./cmd/faythe

build-image:
	docker build -t $(DOCKER_IMAGE_FULL_LATEST) .
	docker tag $(DOCKER_IMAGE_FULL_LATEST) $(DOCKER_IMAGE_FULL)

push-image:
	docker push $(DOCKER_IMAGE_FULL)
	docker push $(DOCKER_IMAGE_FULL_LATEST)

clean:
	rm -rf $(GO_OUT)/faythe
	go clean $(MOD_FLAG) ./...

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	GO111MODULE=on golangci-lint run

dist:
	zip -j -m faythe-$(DOCKER_IMAGE_TAG).zip $(GO_OUT)/faythe
