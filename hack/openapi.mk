%:
	@:

.PHONY: help-openapi
help-openapi:
	@echo "$(WHITE)$(CYAN)[openapi target]$(RESET)"
	@echo "- $(WHITE)make openapi generate$(RESET)                    $(CYAN)Generate Go code from OpenAPI spec$(RESET)"
	@echo "- $(WHITE)make openapi validate$(RESET)                    $(CYAN)Validate OpenAPI spec$(RESET)"

.PHONY: openapi
openapi:
	$(eval TARGET := $(filter-out $@,$(MAKECMDGOALS)))
	@if [ -z "$(TARGET)" ]; then \
		echo "$(RED)target command is required: Use 'make openapi <target>'.$(RESET)"; \
		exit 1; \
	fi
	@if [ "$(TARGET)" = "generate" ]; then \
		echo "$(BLUE)Generating Go code from OpenAPI spec...$(RESET)"; \
		swagger-cli bundle api/server.yaml --outfile api/boilerplate.bundled.yaml; \
		oapi-codegen -package api -generate types -o ./internal/gen/api/types.go api/boilerplate.bundled.yaml; \
		oapi-codegen -package api -generate spec -o ./internal/gen/api/spec.go api/boilerplate.bundled.yaml; \
		oapi-codegen -package api -generate chi-server -o ./internal/gen/api/server.go api/boilerplate.bundled.yaml; \
	elif [ "$(TARGET)" = "validate" ]; then \
		echo "$(BLUE)Validating OpenAPI spec...$(RESET)"; \
		swagger-cli validate api/server.yaml; \
	else \
		echo "$(RED)unknown target command: $(TARGET)$(RESET)"; \
		exit 1; \
	fi
