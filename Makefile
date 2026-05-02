# Binary output directory
BIN_DIR := bin

# List of commands to build
COMMANDS := oplus-ota c16_transer changelog_query downgrade_query downgrade_query-v2 iot_query opex_query opex_analyzer realme_edl_query sota_query sota_changelog_query

# Default target
.PHONY: all
all: build

# Build all binaries
.PHONY: build
build: $(COMMANDS)

# Rule for building each command
.PHONY: $(COMMANDS)
$(COMMANDS):
	@echo "Building $@..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$@ ./cmd/$@

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning bin directory..."
	@rm -rf $(BIN_DIR)

# Help message
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all       - Build all binaries (default)"
	@echo "  build     - Build all binaries"
	@echo "  clean     - Remove build artifacts"
	@echo "  <command> - Build a specific tool (e.g., make oplus-ota)"
	@echo ""
	@echo "Available tools: $(COMMANDS)"
