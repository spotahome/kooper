SERVICE_NAME := kooper
SHELL := $(shell which bash)
DOCKER := $(shell command -v docker)
OSTYPE := $(shell uname)
GID := $(shell id -g)
UID := $(shell id -u)

# cmds
UNIT_TEST_CMD := ./hack/scripts/unit-test.sh
INTEGRATION_TEST_CMD := ./hack/scripts/integration-test.sh 
CI_INTEGRATION_TEST_CMD := ./hack/scripts/integration-test-kind.sh
MOCKS_CMD := ./hack/scripts/mockgen.sh
DOCKER_RUN_CMD := docker run --env ostype=$(OSTYPE) -v ${PWD}:/src --rm -it ${SERVICE_NAME}
DEPS_CMD := go mod tidy
CHECK_CMD := ./hack/scripts/check.sh


help: ## Show this help
	@echo "Help"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-20s\033[93m %s\n", $$1, $$2}'

.PHONY: default
default: help


.PHONY: build
build: ## Build the development docker image.
	docker build -t $(SERVICE_NAME) --build-arg uid=$(UID) --build-arg  gid=$(GID) -f ./docker/dev/Dockerfile .

.PHONY: deps
deps: ## Updates the required dependencies.
	$(DEPS_CMD)

.PHONY: integration-test
integration-test: build ## Runs integration tests out of CI.
	echo "[WARNING] Requires a kubernetes cluster configured (and running) on your kubeconfig!!"
	$(INTEGRATION_TEST_CMD)

.PHONY: test
test: build ## Runs unit tests out of CI.
	$(DOCKER_RUN_CMD) /bin/sh -c '$(UNIT_TEST_CMD)'

.PHONY: check
check: build ## Runs checks.
	@$(DOCKER_RUN_CMD) /bin/sh -c '$(CHECK_CMD)'

.PHONY: ci-unit-test
ci-unit-test: ## Runs unit tests in CI.
	$(UNIT_TEST_CMD)

.PHONY: ci-integration-test
ci-integration-test:  ## Runs integration tests in CI.
	$(CI_INTEGRATION_TEST_CMD)

.PHONY: ci  ## Runs all tests in CI.
ci: ci-unit-test ci-integration-test

.PHONY: mocks
mocks: build  ## Generates mocks.
	$(DOCKER_RUN_CMD) /bin/sh -c '$(MOCKS_CMD)'
