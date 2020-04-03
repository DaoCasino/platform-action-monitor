include includes.mk

PHONY: help install build package publish test deploy clean promote lint bootstrap registry-login

APP ?= platform-action-monitor

.DEFAULT_GOAL := help


bootstrap:
ifndef GO_HAS_LINT
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint > /dev/null 2>&1
endif


install: ## download dependencies
	@go mod download


build:	## build binary
	@echo "=> building revision $(GIT_BRANCH):$(GIT_HASH)"
	@$(GO_FLAGS) go build -a -o $(APP) ./src/.


registry-login:
ifdef DOCKER_PASSWORD
	@echo $$DOCKER_PASSWORD | docker login -u $$DOCKER_LOGIN $$DOCKER_REGISTRY --password-stdin > /dev/null 2>&1
else
	$(error '!!! DOCKER_LOGIN and DOCKER_PASSWORD is required for authentication !!!')
endif


package:  ## package docker image
	@echo "=> packaging $(DOCKER_REPO)/$(APP):$(VERSION)"
	@docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) $(DOCKER_CONTEXT)


publish: registry-login ## publish docker image
	@echo "=> pushing $(DOCKER_IMAGE)"
	@echo 'docker push $(DOCKER_IMAGE)'
ifeq (${DOCKER_TAG_LATEST},true)
	@echo 'docker tag $(DOCKER_IMAGE) $(DOCKER_REPO)/$(APP):latest'
	@echo 'docker push $(DOCKER_REPO)/$(APP):latest'
endif


lint:   bootstrap ## run golangci-linter
	@echo "=> linting"
	@golangci-lint run ./...


test:   lint ## run all test suites
	@echo "=> running all available tests"
	@go test -race -coverprofile=coverage.txt -covermode=atomic ./...


deploy:	## deploy
	@echo "=> deploy $(APP):$(VERSION)"


clean: 	## clean
	@rm -f platform-action-monitor coverage.txt


promote: registry-login ## promote artefact
	@echo "=> release"
	@docker pull $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH)
	@docker tag $(DOCKER_REPO)/$(APP):master-$(GIT_TAG_HASH) $(DOCKER_REPO)/$(APP):$(VERSION)
	@docker push $(DOCKER_REPO)/$(APP):$(VERSION)


help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'
