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
