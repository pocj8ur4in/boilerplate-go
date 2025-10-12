%:
	@:

.PHONY: help-sqlc
help-sqlc:
	@echo "$(WHITE)$(CYAN)[sqlc target]$(RESET)"
	@echo "- $(WHITE)make sqlc generate$(RESET)                       $(CYAN)Generate Go code from SQL queries$(RESET)"

.PHONY: sqlc
sqlc:
	$(eval TARGET := $(filter-out $@,$(MAKECMDGOALS)))
	@if [ -z "$(TARGET)" ]; then \
		echo "$(RED)target command is required: Use 'make sqlc <target>'.$(RESET)"; \
		exit 1; \
	fi
	@if [ "$(TARGET)" = "generate" ]; then \
		echo "$(BLUE)Generating Go code from SQL queries...$(RESET)"; \
		sqlc generate; \
	else \
		echo "$(RED)unknown target command: $(TARGET)$(RESET)"; \
		exit 1; \
	fi
