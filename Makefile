# Include
include hack/go.mk

# Colors
WHITE   := \033[1;37m
BLACK   := \033[30m
RED     := \033[31m
GREEN   := \033[32m
YELLOW  := \033[33m
BLUE    := \033[34m
MAGENTA := \033[35m
CYAN    := \033[36m
GRAY    := \033[1;30m
RESET   := \033[0m

# Project Variables
BINARY_NAME=boilerplate
CONFIG_PATH=$(CURDIR)/config.json
BUILD_DIR=$(CURDIR)/build
HACK_DIR := $(CURDIR)/hack
CMD_DIR := $(CURDIR)/cmd
REPO_NAME := $(shell basename `git rev-parse --show-toplevel`)
GIT_BRANCH := $(shell git branch --show-current)
GIT_COMMIT := $(shell git log -1 --oneline)
REPO_NAME := pocj8ur4in/boilerplate-go
PROJECT_NAME := boilerplate

# help Target
.DEFAULT_GOAL := help
.PHONY: help
help:
	@echo "$(WHITE)$(CYAN)[info]$(RESET)"
	@echo "- $(CYAN)project: $(RESET)$(WHITE)$(REPO_NAME)$(RESET)"
	@echo "- $(CYAN)branch: $(RESET)$(WHITE)$(GIT_BRANCH)$(RESET)"
	@echo "- $(CYAN)commit: $(RESET)$(WHITE)$(GIT_COMMIT)$(RESET)"
	@echo
	@echo "$(WHITE)$(CYAN)[default target]$(RESET)"
	@echo "- $(WHITE)make prepare$(RESET)                             $(CYAN)Prepare project for development$(RESET)"
	@echo
	@$(MAKE) -f $(HACK_DIR)/go.mk help-go CURDIR="$(CURDIR)" BINARY_NAME="$(BINARY_NAME)" CONFIG_FILE="$(CONFIG_FILE)" BUILD_DIR="$(BUILD_DIR)" HACK_DIR="$(HACK_DIR)" CMD_DIR="$(CMD_DIR)" WHITE="$(WHITE)" BLACK="$(BLACK)" RED="$(RED)" CYAN="$(CYAN)" YELLOW="$(YELLOW)" BLUE="$(BLUE)" MAGENTA="$(MAGENTA)" CYAN="$(CYAN)" BLUE="$(BLUE)" GRAY="$(GRAY)" RESET="$(RESET)"
	@echo

.PHONY: prepare
prepare:
	@echo "$(BLUE)Preparing environment...$(RESET)"

	@echo "$(BLUE)Checking Go...$(RESET)"
	@if ! command -v go > /dev/null; then \
		echo "$(YELLOW)go is not installed. installing go...$(RESET)"; \
		if command -v brew > /dev/null; then \
			brew install go; \
		elif command -v apt-get > /dev/null; then \
			sudo apt-get update && sudo apt-get install -y golang-go; \
		elif command -v yum > /dev/null; then \
			sudo yum install -y golang; \
		elif command -v dnf > /dev/null; then \
			sudo dnf install -y golang; \
		else \
			echo "$(YELLOW)trying to install go from official binary...$(RESET)"; \
			mkdir -p /tmp/go && cd /tmp/go; \
			curl -L -o go1.23.6.linux-amd64.tar.gz https://go.dev/dl/go1.23.6.linux-amd64.tar.gz; \
			if [ -f go1.23.6.linux-amd64.tar.gz ]; then \
				sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.6.linux-amd64.tar.gz; \
				if [ -n "$$ZSH_VERSION" ]; then echo 'export PATH=$$PATH:/usr/local/go/bin' >> ~/.zshrc; fi; \
				if [ -n "$$BASH_VERSION" ]; then echo 'export PATH=$$PATH:/usr/local/go/bin' >> ~/.bashrc; fi; \
				export PATH=$$PATH:/usr/local/go/bin; \
				cd $(CURDIR) && rm -rf /tmp/go; \
			else \
				echo "$(RED)failed to download go. please install manually$(RESET)"; \
				exit 1; \
			fi; \
		fi; \
	else \
		echo "$(BLUE)go is already installed: $(shell go version)$(RESET)"; \
	fi

	@echo "$(BLUE)Checking golangci-lint...$(RESET)"
	@if ! command -v golangci-lint > /dev/null; then \
		echo "$(YELLOW)golangci-lint is not installed. installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.4.0; \
		echo 'export PATH=$$PATH:$$HOME/go/bin' >> ~/.bashrc; \
	fi

	@echo "$(BLUE)Preparing go project...$(RESET)"
	go mod download;

