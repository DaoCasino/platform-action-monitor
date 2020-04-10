SHELL := /bin/bash

GIT_BRANCH := $(shell git symbolic-ref --short -q HEAD)
GIT_HASH := $(shell git rev-parse --short HEAD)
GIT_TAG_HASH ?=

VERSION = $(GIT_BRANCH)-$(GIT_HASH)

DOCKER_REGISTRY ?= test
DOCKER_REGISTRY_ORG ?=
DOCKER_CONTEXT ?= ../..
DOCKERFILE ?= Dockerfile
DOCKER_TAG_LATEST ?= false
DOCKER_IMAGE ?= $(DOCKER_REPO)/$(APP):$(VERSION)

GO = go
GO_FLAGS ?= CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_LDFLAAGS ?= -ldflags="-X 'main.Version=$(VERSION)'"
GO_HAS_LINT := $(shell command -v golangci-lint;)

BIN ?= bin
BIN_DIR ?= $(join $(dir $(lastword $(MAKEFILE_LIST))), $(BIN))

APPS_DIR ?= cmd

ifneq (${DOCKER_REGISTRY_ORG},)
	DOCKER_REPO=$(DOCKER_REGISTRY)/$(DOCKER_REGISTRY_ORG)
else
	DOCKER_REPO=$(DOCKER_REGISTRY)
endif

ifdef TRAVIS_TAG
	VERSION := $(TRAVIS_TAG)
	GIT_TAG_HASH := $(shell git rev-list -n 1 $(TRAVIS_TAG) | cut -c1-7)
endif

check-bin:
	@mkdir -p $(BIN_DIR)

bootstrap:
ifndef GO_HAS_LINT
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint > /dev/null 2>&1
endif


registry-login:
ifdef DOCKER_PASSWORD
	@echo $$DOCKER_PASSWORD | docker login -u $$DOCKER_LOGIN $$DOCKER_REGISTRY --password-stdin > /dev/null 2>&1
else
	$(error '!!! DOCKER_LOGIN and DOCKER_PASSWORD is required for authentication !!!')
endif


promote: registry-login ## promote artefact
	@echo "=> release"
	@docker pull $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH)
	@docker tag $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH) $(DOCKER_REPO)/$(APP):$(VERSION)
	@docker push $(DOCKER_REPO)/$(APP):$(VERSION)


publish: registry-login ## publish docker image
	@echo "=> pushing $(DOCKER_IMAGE)"
	@echo 'docker push $(DOCKER_IMAGE)'
ifeq (${DOCKER_TAG_LATEST},true)
	@echo 'docker tag $(DOCKER_IMAGE) $(DOCKER_REPO)/$(APP):latest'
	@echo 'docker push $(DOCKER_REPO)/$(APP):latest'
endif
