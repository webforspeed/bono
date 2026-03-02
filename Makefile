# Bono CLI Makefile
BINARY_NAME := bono
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build test install clean uninstall

# Build bono-core first, then bono
build:
	cd ../bono-core && go build ./...
	go build -o $(BINARY_NAME) .

# Run tests across both repos
test:
	cd ../bono-core && go test ./...

# Test, build, and install to ~/.local/bin
install: test build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@codesign -s - -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"

# Remove installed binary
uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Removed $(BINARY_NAME) from $(INSTALL_DIR)"

# Clean local build
clean:
	@rm -f $(BINARY_NAME)
