BINARY ?= bono
BUILD_DIR ?= build
LOCAL_BIN ?= $(HOME)/.local/bin
GO ?= go
TAG_PATTERN ?= ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$$
CORE_DIR ?= ../bono-core
CORE_REPO_URL ?= https://github.com/webforspeed/bono-core.git

.PHONY: help build test deploy install uninstall clean release

help:
	@echo "Targets:"
	@echo "  make deploy    Build and install $(BINARY) to $(LOCAL_BIN)"
	@echo "  make build     Build binary into $(BUILD_DIR)/$(BINARY)"
	@echo "  make test      Run tests for bono and local bono-core (if present)"
	@echo "  make release   Tag and push a release (requires TAG=vX.Y.Z)"
	@echo "  make uninstall Remove $(LOCAL_BIN)/$(BINARY)"
	@echo "  make clean     Remove build artifacts"

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) .

test:
	@$(GO) test ./...
	@if [ -d ../bono-core ]; then \
		echo "Running bono-core tests from ../bono-core"; \
		(cd ../bono-core && $(GO) test ./...); \
	else \
		echo "Skipping ../bono-core tests (directory not found)."; \
	fi

deploy: install

install: build
	@mkdir -p $(LOCAL_BIN)
	@install -m 0755 $(BUILD_DIR)/$(BINARY) $(LOCAL_BIN)/$(BINARY)
	@echo "Installed $(BINARY) to $(LOCAL_BIN)/$(BINARY)"
	@if echo ":$$PATH:" | grep -q ":$(LOCAL_BIN):"; then \
		echo "$(LOCAL_BIN) is already in PATH"; \
	else \
		echo "$(LOCAL_BIN) is not in PATH. Add this to your shell config:"; \
		echo "  export PATH=\"$(LOCAL_BIN):\$$PATH\""; \
	fi

uninstall:
	@rm -f $(LOCAL_BIN)/$(BINARY)
	@echo "Removed $(LOCAL_BIN)/$(BINARY)"

clean:
	@rm -rf $(BUILD_DIR)

release:
	@if [ -z "$(TAG)" ]; then \
		echo "error: TAG is required. usage: make release TAG=v0.0.1"; \
		exit 1; \
	fi
	@if ! printf "%s" "$(TAG)" | grep -Eq "$(TAG_PATTERN)"; then \
		echo "error: invalid TAG format: $(TAG)"; \
		echo "expected format: vX.Y.Z (example: v0.0.1)"; \
		exit 1; \
	fi
	@if git rev-parse -q --verify "refs/tags/$(TAG)" >/dev/null; then \
		echo "error: local tag already exists: $(TAG)"; \
		exit 1; \
	fi
	@if git ls-remote --tags origin "refs/tags/$(TAG)" | grep -q .; then \
		echo "error: remote tag already exists on origin: $(TAG)"; \
		exit 1; \
	fi
	@if ! git ls-remote --tags "$(CORE_REPO_URL)" "refs/tags/$(TAG)" | grep -q .; then \
		if [ ! -d "$(CORE_DIR)/.git" ]; then \
			echo "error: bono-core tag $(TAG) is missing and $(CORE_DIR) is not available."; \
			echo "clone bono-core at $(CORE_DIR) or create the tag manually in bono-core first."; \
			exit 1; \
		fi; \
		if ! git -C "$(CORE_DIR)" remote get-url origin >/dev/null 2>&1; then \
			echo "error: $(CORE_DIR) has no git origin remote configured."; \
			exit 1; \
		fi; \
		if [ -n "$$(git -C "$(CORE_DIR)" status --porcelain)" ]; then \
			echo "error: $(CORE_DIR) has uncommitted changes. commit/stash before release."; \
			exit 1; \
		fi; \
		if ! git -C "$(CORE_DIR)" rev-parse -q --verify "refs/tags/$(TAG)" >/dev/null; then \
			git -C "$(CORE_DIR)" tag -a "$(TAG)" -m "bono-core $(TAG)"; \
		fi; \
		git -C "$(CORE_DIR)" push origin "$(TAG)"; \
	fi
	@$(MAKE) test
	git tag -a "$(TAG)" -m "$(BINARY) $(TAG)"
	git push origin "$(TAG)"
