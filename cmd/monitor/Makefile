include ../../includes.mk

APP := monitor

.PHONY: build package lint test clean

build:
	@echo "=> building $(APP) binary"
	@$(GO_FLAGS) $(GO_LDFLAGS) $(GO) build -a -o $(BIN_DIR)/$(APP) .


package:
	@echo "=> packaging $(DOCKER_REPO)/$(APP):$(VERSION)"
	@docker build -t $(DOCKER_REPO)/$(APP):$(VERSION) -f $(DOCKERFILE) $(DOCKER_CONTEXT) $(LOG_OUTPUT)


lint:   bootstrap ## run golangci-linter
	@echo "=> linting codebase"
	@golangci-lint run ./...


test:   lint ## run all test suites
	@echo "=> running tests"
	@cd ../../pkg/apps/$(APP); go test -race -coverprofile=../../../coverage.txt -covermode=atomic ./...


clean:
	@rm -f monitor coverage.txt


deploy:
	@echo "=> deploy $(app)"
