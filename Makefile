.PHONY: default deps-development build dev shell unit-test integration-test test ci-unit-test ci-integration-test ci mocks

# Name of this service/application
SERVICE_NAME := kooper

# Path of the go service inside docker
DOCKER_GO_SERVICE_PATH := /go/src/github.com/spotahome/kooper

# Shell to use for running scripts
SHELL := $(shell which bash)

# Get docker path or an empty string
DOCKER := $(shell command -v docker)

# Get the main unix group for the user running make (to be used by docker-compose later)
GID := $(shell id -g)

# Get the unix user id for the user running make (to be used by docker-compose later)
UID := $(shell id -u)

# cmds
UNIT_TEST_CMD := ./hack/scripts/unit-test.sh
INTEGRATION_TEST_CMD := ./hack/scripts/integration-test.sh 
MOCKS_CMD := ./hack/scripts/mockgen.sh
DOCKER_RUN_CMD := docker run -v ${PWD}:$(DOCKER_GO_SERVICE_PATH) --rm -it $(SERVICE_NAME)

# environment dirs
DEV_DIR := docker/dev

# The default action of this Makefile is to build the development docker image
default: build

# Test if the dependencies we need to run this Makefile are installed
deps-development:
ifndef DOCKER
	@echo "Docker is not available. Please install docker"
	@exit 1
endif

# Build the development docker image
build:
	docker build -t $(SERVICE_NAME) --build-arg uid=$(UID) --build-arg  gid=$(GID) -f ./docker/dev/Dockerfile .

# Test stuff in dev
unit-test: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(UNIT_TEST_CMD)'
integration-test: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(INTEGRATION_TEST_CMD)'
test: integration-test

# Test stuff in ci
ci-unit-test: 
	$(UNIT_TEST_CMD)
ci-integration-test:
	$(INTEGRATION_TEST_CMD)
ci: ci-integration-test

# Mocks stuff in dev
mocks: build
	$(DOCKER_RUN_CMD) /bin/sh -c '$(MOCKS_CMD)'
