%:
	@:

.PHONY: help-go
help-go:
	@echo "$(WHITE)$(CYAN)[go target]$(RESET)"
	@echo "- $(WHITE)make go build$(RESET)                            $(CYAN)Build go project$(RESET)"
	@echo "- $(WHITE)make go run$(RESET)                              $(CYAN)Run go project$(RESET)"
	@echo "- $(WHITE)make go dev$(RESET)                              $(CYAN)Run go project in development mode$(RESET)"
	@echo "- $(WHITE)make go test$(RESET)                             $(CYAN)Test go project$(RESET)"
	@echo "- $(WHITE)make go benchmark$(RESET)                        $(CYAN)Benchmark go project$(RESET)"
	@echo "- $(WHITE)make go coverage$(RESET)                         $(CYAN)Test coverage for go project$(RESET)"
	@echo "- $(WHITE)make go fmt$(RESET)                              $(CYAN)Format go project$(RESET)"
	@echo "- $(WHITE)make go lint$(RESET)                             $(CYAN)Lint go project$(RESET)"

.PHONY: go
go:
	$(eval TARGET := $(filter-out $@,$(MAKECMDGOALS)))
	@if [ -z "$(TARGET)" ]; then \
		echo "$(RED)target command is required: Use 'make go <service>'.$(RESET)"; \
		exit 1; \
	fi
	@if [ "$(TARGET)" = "build" ]; then \
		echo "$(BLUE)Building go project...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_PATH) CGO_ENABLED=1 go build -o build/$(BINARY_NAME) $(CMD_DIR)/$(BINARY_NAME); \
	elif [ "$(TARGET)" = "run" ]; then \
		echo "$(BLUE)Running go project...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_PATH) CGO_ENABLED=1 go run $(CMD_DIR)/$(BINARY_NAME)/main.go; \
	elif [ "$(TARGET)" = "dev" ]; then \
		echo "$(BLUE)Running go project in development mode...$(RESET)"; \
		if command -v air > /dev/null; then \
			CONFIG_PATH=$(CONFIG_PATH) air; \
		fi; \
	elif [ "$(TARGET)" = "test" ]; then \
		echo "$(BLUE)Testing go project...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_EXAMPLE_PATH) CGO_ENABLED=1 go test $$(go list $(CURDIR)/... | grep -v /internal/gen/) -v; \
	elif [ "$(TARGET)" = "benchmark" ]; then \
		echo "$(BLUE)Benchmarking go project...$(RESET)"; \
		cd $(CURDIR) && CONFIG_PATH=$(CONFIG_EXAMPLE_PATH) CGO_ENABLED=1 go test $$(go list ./... | grep -v /internal/gen/) -bench=. -benchmem -run=^$$; \
	elif [ "$(TARGET)" = "coverage" ]; then \
		echo "$(BLUE)Testing go project with coverage...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_EXAMPLE_PATH) CGO_ENABLED=1 go test $$(go list $(CURDIR)/... | grep -v /internal/gen/) -v -coverprofile=coverage.out -covermode=atomic && \
		go tool cover -func=coverage.out; \
	elif [ "$(TARGET)" = "fmt" ]; then \
		echo "$(BLUE)Formatting go project...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_PATH) CGO_ENABLED=1 golangci-lint fmt $(CURDIR)/... -c $(CURDIR)/golangci.yaml; \
	elif [ "$(TARGET)" = "lint" ]; then \
		echo "$(BLUE)Linting go project...$(RESET)"; \
		CONFIG_PATH=$(CONFIG_PATH) CGO_ENABLED=1 golangci-lint run $(CURDIR)/... -c $(CURDIR)/golangci.yaml; \
	else \
		echo "$(RED)unknown target command: $(TARGET)$(RESET)"; \
		exit 1; \
	fi
