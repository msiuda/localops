# LocalOps developer Makefile.
#
# This is a convenience wrapper, not a build system: Wails Taskfiles under
# cmd/desktop remain the canonical implementation of Wails-specific
# dev/build/package operations, and this Makefile only delegates to them.
# LocalOps does not require Make to build.

# ==============================================================================
# Colors
# ==============================================================================
BLUE       := \033[0;34m
BOLD_WHITE := \033[1;37m
NC         := \033[0m

# ==============================================================================
# Variables
# ==============================================================================
WAILS_VERSION := v3.0.0-beta.26
DESKTOP_DIR   := cmd/desktop
FRONTEND_DIR  := $(DESKTOP_DIR)/frontend
ARGS          ?=

.PHONY: default
default: help

##@ Help

.PHONY: help
help: ## Show list of targets with descriptions
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(BOLD_WHITE)Usage:$(NC)\n\n  make $(BLUE)<target>$(NC)\n\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  $(BLUE)%-24s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BOLD_WHITE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ General

.PHONY: bootstrap
bootstrap: install-wails frontend-install bindings ## First-time setup: install pinned Wails CLI, frontend deps, and generate bindings

.PHONY: dev
dev: desktop-dev ## Run the desktop app in Wails dev mode (day-to-day UI development)

.PHONY: build
build: desktop-build ## Build the production desktop application

.PHONY: run
run: ## Run the desktop application via the existing Wails task
	@printf "$(BLUE)Running desktop app (wails3 task run)...$(NC)\n"
	cd $(DESKTOP_DIR) && wails3 task run

##@ CLI

.PHONY: cli-build
cli-build: ## Build the LocalOps CLI to bin/localops
	@printf "$(BLUE)Building cmd/localops...$(NC)\n"
	go build -o bin/localops ./cmd/localops

.PHONY: cli-run
cli-run: ## Run the CLI via `go run`, e.g. make cli-run ARGS="overview"
	go run ./cmd/localops $(ARGS)

##@ Desktop

.PHONY: desktop-dev
desktop-dev: ## Run the desktop app in Wails dev mode
	@printf "$(BLUE)Starting Wails dev mode...$(NC)\n"
	cd $(DESKTOP_DIR) && wails3 task dev

.PHONY: desktop-build
desktop-build: ## Build the production desktop application via Wails
	@printf "$(BLUE)Building desktop app (wails3 build)...$(NC)\n"
	cd $(DESKTOP_DIR) && wails3 build

.PHONY: desktop-package
desktop-package: ## Package the desktop app via the Wails package task
	@printf "$(BLUE)Packaging desktop app (wails3 task package)...$(NC)\n"
	cd $(DESKTOP_DIR) && wails3 task package

.PHONY: bindings
bindings: ## Regenerate Wails TypeScript bindings
	@printf "$(BLUE)Generating Wails bindings...$(NC)\n"
	cd $(DESKTOP_DIR) && wails3 task common:generate:bindings

##@ Frontend

.PHONY: check-pnpm
check-pnpm: ## Fail clearly if pnpm (LocalOps's frontend package manager) is not installed
	@command -v pnpm >/dev/null 2>&1 || { \
		echo "pnpm is required but was not found on PATH."; \
		echo "LocalOps uses pnpm for frontend dependencies - install it from https://pnpm.io/installation"; \
		exit 1; \
	}

.PHONY: frontend-install
frontend-install: check-pnpm ## Install frontend dependencies (pnpm install --frozen-lockfile)
	@printf "$(BLUE)Installing frontend dependencies...$(NC)\n"
	cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile

.PHONY: frontend-typecheck
frontend-typecheck: check-pnpm ## Run the frontend typecheck script
	cd $(FRONTEND_DIR) && pnpm typecheck

.PHONY: frontend-build
frontend-build: check-pnpm ## Run the frontend production build
	cd $(FRONTEND_DIR) && pnpm build

.PHONY: frontend-lint
frontend-lint: check-pnpm ## Lint the frontend (ESLint)
	cd $(FRONTEND_DIR) && pnpm lint

.PHONY: frontend-lint-fix
frontend-lint-fix: check-pnpm ## Lint the frontend and auto-fix what ESLint can
	cd $(FRONTEND_DIR) && pnpm lint:fix

.PHONY: frontend-format
frontend-format: check-pnpm ## Format the frontend with Prettier (may modify files)
	cd $(FRONTEND_DIR) && pnpm format

.PHONY: frontend-format-check
frontend-format-check: check-pnpm ## Check frontend formatting without modifying files
	cd $(FRONTEND_DIR) && pnpm format:check

.PHONY: frontend-test
frontend-test: check-pnpm ## Run frontend tests (Vitest + Testing Library)
	cd $(FRONTEND_DIR) && pnpm test:run

##@ Quality

.PHONY: fmt
fmt: ## Format Go AND frontend source (may modify files) — use `check` to verify without changes
	@printf "$(BLUE)Formatting Go...$(NC)\n"
	gofmt -l -w .
	$(MAKE) frontend-format

.PHONY: test
test: ## Run Go tests
	go test ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: typecheck
typecheck: frontend-typecheck ## Run frontend typecheck (alias)

.PHONY: check
check: ## Run all non-mutating pre-commit verification (Go + frontend); never auto-fixes — use `fmt` for that
	@printf "$(BLUE)Checking gofmt cleanliness...$(NC)\n"
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	$(MAKE) test
	$(MAKE) vet
	$(MAKE) frontend-typecheck
	$(MAKE) frontend-lint
	$(MAKE) frontend-test
	$(MAKE) frontend-format-check
	$(MAKE) frontend-build

.PHONY: verify
verify: check ## Alias for check

##@ Cleanup

.PHONY: clean
clean: ## Remove generated build outputs (desktop bin/dist/bindings/coverage, root bin)
	@printf "$(BLUE)Removing generated build outputs...$(NC)\n"
	rm -rf bin
	rm -rf $(DESKTOP_DIR)/bin
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/bindings
	rm -rf $(FRONTEND_DIR)/coverage
# Deliberately NOT removed: src/routeTree.gen.ts — it's TanStack Router's
# generated route tree, but it's committed to source control (see
# docs/frontend.md), not a disposable build artifact.

.PHONY: clean-all
clean-all: clean ## Also remove frontend node_modules
	@printf "$(BLUE)Removing frontend node_modules...$(NC)\n"
	rm -rf $(FRONTEND_DIR)/node_modules

##@ Tools

.PHONY: install-wails
install-wails: ## Install the pinned Wails v3 CLI
	go install github.com/wailsapp/wails/v3/cmd/wails3@$(WAILS_VERSION)
