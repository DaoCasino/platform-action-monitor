include includes.mk

APP ?= platform-action-monitor


.PHONY: all
all: help

.PHONY:
build:	## build binary
	@echo "=> building revision $(GIT_HASH)"
	@$(GO_FLAGS) go build -a -o $(APP) ./src/.


.PHONY: package
package:  ## package docker image
	@echo "=> packaging $(DOCKER_IMAGE)"
	@docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) $(DOCKER_CONTEXT)


.PHONY: publish
publish: ## publish docker image
	@echo "=> pushing $(DOCKER_IMAGE)"
	@echo 'docker push $(DOCKER_REPO)/$(APP):$(VERSION)'
ifeq (${DOCKER_TAG_LATEST},true)
	@echo 'docker tag $(DOCKER_REPO)/$(APP):$(VERSION) $(DOCKER_REPO)/$(APP):latest'
	@echo 'docker push $(DOCKER_REPO)/$(APP):latest'
endif


.PHONY: test
test:	## run all test suites
	@echo "=> running all available tests"
	@go test -v ./...


.PHONY: deploy
deploy:	## deploy
	@echo "=> deploy $(APP):$(VERSION)"


.PHONY: clean
clean: 	## clean
	@rm -f platform-action-monitor coverage.txt


.PHONY: help
help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'
