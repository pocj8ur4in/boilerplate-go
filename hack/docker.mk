%:
	@:

.PHONY: help-docker
help-docker:
	@echo "$(WHITE)$(CYAN)[docker target]$(RESET)"
	@echo "- $(WHITE)make docker dev$(RESET)                          $(CYAN)Run project on docker with dev environment$(RESET)"

.PHONY: docker
docker:
	$(eval TARGET := $(filter-out $@,$(MAKECMDGOALS)))
	@if [ -z "$(TARGET)" ]; then \
		echo "$(RED)target command is required: Use 'make go <service>'.$(RESET)"; \
		exit 1; \
	fi
	@if [ "$(TARGET)" = "dev" ]; then \
		echo "$(BLUE)Running project on docker with dev environment...$(RESET)"; \
		docker-compose -f $(CURDIR)/docker-compose.yaml up --build; \
	else \
		echo "$(RED)unknown target command: $(TARGET)$(RESET)"; \
		exit 1; \
	fi
