include includes.mk

PHONY: help install build package publish test deploy clean promote lint bootstrap registry-login

APPS ?= monitor

.DEFAULT_GOAL := help



install: ## download dependencies
	@go mod download > /dev/null >&1


build:	install ## build binary
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) build ;)


package:  ## package docker image
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) package ;)


lint:   bootstrap ## run golangci-linter
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) lint ;)


test:   ## run all test suites
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) test ;)


deploy:	## deploy
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) deploy ;)


clean: 	## clean
	@rm -rf coverage.txt bin/
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) clean ;)


help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'
