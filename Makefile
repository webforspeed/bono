# Bono CLI Makefile
BINARY_NAME := bono
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install clean uninstall

# Build the binary
build:
	go build -o $(BINARY_NAME) .

# Build and install to ~/.local/bin (already in PATH on most systems)
install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Installed $(BINARY_NAME) to $(INSTALL_DIR)"
	@echo "  Run 'bono' from anywhere to use"

# Remove installed binary
uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Removed $(BINARY_NAME) from $(INSTALL_DIR)"

# Clean local build
clean:
	@rm -f $(BINARY_NAME)
	@echo "✓ Cleaned local binary"
