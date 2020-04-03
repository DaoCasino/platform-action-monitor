SHELL := /bin/bash

GIT_BRANCH := $(shell git symbolic-ref --short -q HEAD)
GIT_HASH := $(shell git rev-parse --short HEAD)

VERSION = $(GIT_BRANCH)-$(GIT_HASH)

DOCKER_REGISTRY ?= test
DOCKER_REGISTRY_ORG ?=
DOCKER_CONTEXT ?= .
DOCKERFILE ?= Dockerfile
DOCKER_TAG_LATEST ?= false
DOCKER_IMAGE ?= $(DOCKER_REPO)/$(APP):$(VERSION)

GO = go
GO_FLAGS ?= CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_LDFLAAGS ?= -ldflags="-X 'main.Version=$(VERSION)'"

ifneq (${DOCKER_REGISTRY_ORG},)
	DOCKER_REPO=$(DOCKER_REGISTRY)/$(DOCKER_REGISTRY_ORG)
else
	DOCKER_REPO=$(DOCKER_REGISTRY)
endif