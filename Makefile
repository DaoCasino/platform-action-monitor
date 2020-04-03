include includes.mk

PHONY: help install build package publish test deploy clean


APP ?= platform-action-monitor

.DEFAULT_GOAL := help

install: ## download dependencies
	@go mod download


build:	## build binary
	@echo "=> building revision $(GIT_BRANCH):$(GIT_HASH)"
	@$(GO_FLAGS) go build -a -o $(APP) ./src/.


package:  ## package docker image
	@echo "=> packaging $(DOCKER_REPO)/$(APP):$(VERSION)"
	@docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) $(DOCKER_CONTEXT)


publish: ## publish docker image
	@echo "=> pushing $(DOCKER_IMAGE)"
	@echo 'docker push $(DOCKER_IMAGE)'
ifeq (${DOCKER_TAG_LATEST},true)
	@echo 'docker tag $(DOCKER_IMAGE) $(DOCKER_REPO)/$(APP):latest'
	@echo 'docker push $(DOCKER_REPO)/$(APP):latest'
endif


test:	## run all test suites
	@echo "=> running all available tests"
	@go test -race -coverprofile=coverage.txt -cover ./...


deploy:	## deploy
	@echo "=> deploy $(APP):$(VERSION)"


clean: 	## clean
	@rm -f platform-action-monitor coverage.txt


help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'
