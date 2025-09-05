APP_NAME := hosts-manager
PYTHON := python3
PIP := $(PYTHON) -m pip
DIST_DIR := dist
LOCAL_BIN := $(HOME)/.local/bin/$(APP_NAME)
VENV_DIR := .venv
EXTERNALLY_MANAGED := $(shell find /usr/lib -maxdepth 1 -type d -name "python3.*" -exec test -f '{}/EXTERNALLY-MANAGED' \; -print | head -n 1)

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make venv       - Create a local virtual environment"
	@echo "  make build      - Build the wheel package"
	@echo "  make install    - Install the package (auto-handles PEP 668)"
	@echo "  make uninstall  - Uninstall the package cleanly"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make reinstall  - Clean, build, uninstall & reinstall"

.PHONY: venv
venv:
	@if [ ! -d "$(VENV_DIR)" ]; then \
	    echo "🐍 Creating virtual environment..."; \
	    $(PYTHON) -m venv --system-site-packages $(VENV_DIR); \
	    $(VENV_DIR)/bin/pip install --upgrade pip; \
	fi

.PHONY: build
build: venv
	@echo "📦 Building wheel..."
	@$(VENV_DIR)/bin/pip install --upgrade build
	@$(VENV_DIR)/bin/python -m build

.PHONY: install
install: build
	@echo "🔧 Installing $(APP_NAME)..."
	@if [ "$(EXTERNALLY_MANAGED)" ]; then \
	    echo "⚠️  Detected externally-managed environment. Installing in user mode..."; \
	    $(VENV_DIR)/bin/pip install --user $(DIST_DIR)/*.whl; \
	else \
	    $(VENV_DIR)/bin/pip install $(DIST_DIR)/*.whl; \
	fi
	@echo "✅ Installed $(APP_NAME)"

.PHONY: uninstall
uninstall:
	@echo "🧹 Uninstalling $(APP_NAME)..."
	@if [ "$(EXTERNALLY_MANAGED)" ]; then \
	    echo "⚠️  Detected externally-managed environment. Uninstalling in user mode..."; \
	    pip uninstall --break-system-packages -y $(APP_NAME) || true; \
	else \
		@$(VENV_DIR)/bin/pip uninstall -y $(APP_NAME) || true; \
	fi
	@if [ -f "$(LOCAL_BIN)" ]; then \
	    echo "🗑️  Removing leftover binary: $(LOCAL_BIN)"; \
	    rm -f "$(LOCAL_BIN)"; \
	fi
	@echo "✅ Uninstalled $(APP_NAME)"

.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(DIST_DIR) build *.egg-info $(VENV_DIR)
	@echo "✅ Cleanup complete"

.PHONY: reinstall
reinstall: clean build uninstall install
