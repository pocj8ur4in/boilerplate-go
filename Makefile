# Include
include hack/go.mk
include hack/sqlc.mk
include hack/docker.mk

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
CONFIG_EXAMPLE_PATH=$(CURDIR)/config.example.json
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
	@$(MAKE) -f $(HACK_DIR)/go.mk help-go CURDIR="$(CURDIR)" BINARY_NAME="$(BINARY_NAME)" CONFIG_FILE="$(CONFIG_FILE)" CONFIG_EXAMPLE_FILE="$(CONFIG_EXAMPLE_FILE)" BUILD_DIR="$(BUILD_DIR)" HACK_DIR="$(HACK_DIR)" CMD_DIR="$(CMD_DIR)" WHITE="$(WHITE)" BLACK="$(BLACK)" RED="$(RED)" CYAN="$(CYAN)" YELLOW="$(YELLOW)" BLUE="$(BLUE)" MAGENTA="$(MAGENTA)" CYAN="$(CYAN)" BLUE="$(BLUE)" GRAY="$(GRAY)" RESET="$(RESET)"
	@echo
	@$(MAKE) -f $(HACK_DIR)/sqlc.mk help-sqlc CURDIR="$(CURDIR)" BINARY_NAME="$(BINARY_NAME)" CONFIG_FILE="$(CONFIG_FILE)" CONFIG_EXAMPLE_FILE="$(CONFIG_EXAMPLE_FILE)" BUILD_DIR="$(BUILD_DIR)" HACK_DIR="$(HACK_DIR)" CMD_DIR="$(CMD_DIR)" WHITE="$(WHITE)" BLACK="$(BLACK)" RED="$(RED)" CYAN="$(CYAN)" YELLOW="$(YELLOW)" BLUE="$(BLUE)" MAGENTA="$(MAGENTA)" CYAN="$(CYAN)" BLUE="$(BLUE)" GRAY="$(GRAY)" RESET="$(RESET)"
	@echo
	@$(MAKE) -f $(HACK_DIR)/oapi.mk help-oapi CURDIR="$(CURDIR)" BINARY_NAME="$(BINARY_NAME)" CONFIG_FILE="$(CONFIG_FILE)" CONFIG_EXAMPLE_FILE="$(CONFIG_EXAMPLE_FILE)" BUILD_DIR="$(BUILD_DIR)" HACK_DIR="$(HACK_DIR)" CMD_DIR="$(CMD_DIR)" WHITE="$(WHITE)" BLACK="$(BLACK)" RED="$(RED)" CYAN="$(CYAN)" YELLOW="$(YELLOW)" BLUE="$(BLUE)" MAGENTA="$(MAGENTA)" CYAN="$(CYAN)" BLUE="$(BLUE)" GRAY="$(GRAY)" RESET="$(RESET)"
	@echo
	@$(MAKE) -f $(HACK_DIR)/docker.mk help-docker CURDIR="$(CURDIR)" BINARY_NAME="$(BINARY_NAME)" CONFIG_FILE="$(CONFIG_FILE)" CONFIG_EXAMPLE_FILE="$(CONFIG_EXAMPLE_FILE)" BUILD_DIR="$(BUILD_DIR)" HACK_DIR="$(HACK_DIR)" CMD_DIR="$(CMD_DIR)" WHITE="$(WHITE)" BLACK="$(BLACK)" RED="$(RED)" CYAN="$(CYAN)" YELLOW="$(YELLOW)" BLUE="$(BLUE)" MAGENTA="$(MAGENTA)" CYAN="$(CYAN)" BLUE="$(BLUE)" GRAY="$(GRAY)" RESET="$(RESET)"
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

	@echo "$(BLUE)Checking air...$(RESET)"
	@if ! command -v air > /dev/null; then \
		echo "$(YELLOW)air is not installed. installing air...$(RESET)"; \
		go install github.com/cosmtrek/air@v1.61.7; \
		echo 'export PATH=$$PATH:$$HOME/go/bin' >> ~/.bashrc; \
	fi

	@echo "$(BLUE)Checking golangci-lint...$(RESET)"
	@if ! command -v golangci-lint > /dev/null; then \
		echo "$(YELLOW)golangci-lint is not installed. installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.4.0; \
		echo 'export PATH=$$PATH:$$HOME/go/bin' >> ~/.bashrc; \
	fi

	@echo "$(BLUE)Checking gosec...$(RESET)"
	@if ! command -v gosec > /dev/null; then \
		echo "$(YELLOW)gosec is not installed. installing gosec...$(RESET)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4; \
		echo 'export PATH=$$PATH:$$HOME/go/bin' >> ~/.bashrc; \
	else \
		echo "$(BLUE)gosec is already installed: $(shell gosec --version 2>&1 | head -n 1)$(RESET)"; \
	fi

	@echo "$(BLUE)Checking docker...$(RESET)"
	@if ! command -v docker > /dev/null; then \
		echo "$(YELLOW)docker is not installed. installing docker...$(RESET)"; \
		if command -v brew > /dev/null; then \
			brew install --cask docker; \
		elif command -v apt-get > /dev/null; then \
			sudo apt-get update && sudo apt-get install -y docker.io docker-compose-plugin; \
			sudo systemctl enable docker && sudo systemctl start docker; \
			sudo usermod -aG docker $$USER; \
		elif command -v yum > /dev/null; then \
			sudo yum install -y docker docker-compose; \
			sudo systemctl enable docker && sudo systemctl start docker; \
			sudo usermod -aG docker $$USER; \
		elif command -v dnf > /dev/null; then \
			sudo dnf install -y docker docker-compose; \
			sudo systemctl enable docker && sudo systemctl start docker; \
			sudo usermod -aG docker $$USER; \
		else \
			echo "$(RED)please install docker manually$(RESET)"; \
			exit 1; \
		fi; \
	else \
		echo "$(BLUE)docker is already installed: $(shell docker --version)$(RESET)"; \
	fi

	@echo "$(BLUE)Checking docker compose...$(RESET)"
	@if ! command -v docker-compose > /dev/null && ! docker compose version > /dev/null 2>&1; then \
		echo "$(YELLOW)docker compose is not installed. installing...$(RESET)"; \
		if command -v brew > /dev/null; then \
			brew install docker-compose; \
		elif command -v apt-get > /dev/null; then \
			sudo apt-get install -y docker-compose-plugin; \
		elif command -v yum > /dev/null; then \
			sudo yum install -y docker-compose; \
		elif command -v dnf > /dev/null; then \
			sudo dnf install -y docker-compose; \
		else \
			sudo curl -L "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/docker-compose; \
			sudo chmod +x /usr/local/bin/docker-compose; \
		fi; \
	else \
		if command -v docker-compose > /dev/null; then \
			echo "$(BLUE)docker compose is already installed: $(shell docker-compose --version)$(RESET)"; \
		else \
			echo "$(BLUE)docker compose plugin is available: $(shell docker compose version 2>/dev/null)$(RESET)"; \
		fi; \
	fi

	@echo "$(BLUE)Checking sqlc...$(RESET)"
	@if ! command -v sqlc > /dev/null; then \
		echo "$(YELLOW)sqlc is not installed. installing sqlc...$(RESET)"; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0; \
		echo 'export PATH=$$PATH:$$HOME/go/bin' >> ~/.bashrc; \
	else \
		echo "$(BLUE)sqlc is already installed: $(shell sqlc version)$(RESET)"; \
	fi

	@echo "$(BLUE)Checking swagger-cli...$(RESET)"
	@if ! command -v swagger-cli > /dev/null; then \
		echo "$(YELLOW)swagger-cli is not installed. installing swagger-cli...$(RESET)"; \
		npm install -g swagger-cli; \
	else \
		echo "$(BLUE)swagger-cli is already installed: $(shell swagger-cli --version)$(RESET)"; \
	fi

	@echo "$(BLUE)Preparing go project...$(RESET)"
	go mod download;
	go mod tidy;

