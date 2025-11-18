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
		if docker compose version > /dev/null 2>&1; then \
			docker compose -f $(CURDIR)/docker-compose.yaml up --build; \
		elif command -v docker-compose > /dev/null; then \
			docker-compose -f $(CURDIR)/docker-compose.yaml up --build; \
		fi; \
	else \
		echo "$(RED)unknown target command: $(TARGET)$(RESET)"; \
		exit 1; \
	fi
