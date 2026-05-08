# =============================================================================
#  MPanel — Makefile
#  Usage: make <target>   |   make help
# =============================================================================

# ── Project metadata ──────────────────────────────────────────────────────────
APP_NAME   := mpanel
BINARY     := $(APP_NAME)
MAIN       := main.go
VERSION    := $(shell cat config/version 2>/dev/null | tr -d '[:space:]')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# ── Go toolchain ──────────────────────────────────────────────────────────────
GO         := go
GOFLAGS    :=
CGO        := CGO_ENABLED=1
CGO_FLAGS  := CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
BUILD_TAGS := with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale
LDFLAGS    := -w -s
TEST_PKGS  := ./config/... ./database/... ./api/... ./service/... ./util/...

# ── Docker / Compose ─────────────────────────────────────────────────────────
COMPOSE    := docker compose
IMAGE_NAME := $(APP_NAME)
DC_FILE    := docker-compose.yml

# ── Frontend ──────────────────────────────────────────────────────────────────
FRONTEND_DIR := frontend
WEB_HTML     := web/html

# ── Colours ───────────────────────────────────────────────────────────────────
BOLD  := \033[1m
GREEN := \033[32m
CYAN  := \033[36m
YELLOW:= \033[33m
RESET := \033[0m

# =============================================================================
#  Default target
# =============================================================================
.DEFAULT_GOAL := help

# =============================================================================
#  HELP
# =============================================================================
.PHONY: help
help: ## Show this help message
	@printf "\n$(BOLD)$(CYAN)MPanel $(VERSION) ($(GIT_COMMIT))$(RESET)\n\n"
	@printf "$(BOLD)Usage:$(RESET)  make $(CYAN)<target>$(RESET)\n\n"
	@awk 'BEGIN {FS = ":.*##"} \
	     /^[a-zA-Z_%-]+:.*##/ { \
	       printf "  $(CYAN)%-22s$(RESET) %s\n", $$1, $$2 \
	     } \
	     /^##@/ { \
	       printf "\n$(BOLD)%s$(RESET)\n", substr($$0, 5) \
	     }' $(MAKEFILE_LIST)
	@printf "\n"

# =============================================================================
##@ Build
# =============================================================================

.PHONY: build
build: ## Build the native binary (CGO, all tags)
	@printf "$(GREEN)▶ Building $(APP_NAME) $(VERSION)...$(RESET)\n"
	$(CGO) $(CGO_FLAGS) $(GO) build \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(LDFLAGS)" \
		-o $(BINARY) $(MAIN)
	@printf "$(GREEN)✔ Binary: ./$(BINARY)$(RESET)\n"

.PHONY: build-dev
build-dev: ## Build without stripping (debug symbols, race detector)
	@printf "$(GREEN)▶ Building dev binary...$(RESET)\n"
	$(CGO) $(CGO_FLAGS) $(GO) build -race \
		-tags "$(BUILD_TAGS)" \
		-o $(BINARY)-dev $(MAIN)
	@printf "$(GREEN)✔ Binary: ./$(BINARY)-dev$(RESET)\n"

.PHONY: build-frontend
build-frontend: ## Build the Vue/Node frontend
	@printf "$(GREEN)▶ Building frontend...$(RESET)\n"
	cd $(FRONTEND_DIR) && npm install && npm run build
	@mkdir -p $(WEB_HTML)
	@cp -r $(FRONTEND_DIR)/dist/. $(WEB_HTML)/
	@printf "$(GREEN)✔ Frontend built → $(WEB_HTML)$(RESET)\n"

.PHONY: build-all
build-all: build-frontend build ## Build frontend then binary

.PHONY: clean
clean: ## Remove build artefacts
	@printf "$(YELLOW)▶ Cleaning...$(RESET)\n"
	rm -f $(BINARY) $(BINARY)-dev
	rm -f mpanel_backup_*.json
	rm -rf mpanel mpanel-linux-*.tar.gz
	@printf "$(GREEN)✔ Clean$(RESET)\n"

# =============================================================================
##@ Testing
# =============================================================================

.PHONY: test
test: ## Run all unit tests (no DB required)
	@printf "$(GREEN)▶ Running tests...$(RESET)\n"
	$(CGO) $(GO) test $(TEST_PKGS) -v -count=1 2>&1
	@printf "$(GREEN)✔ Tests complete$(RESET)\n"

.PHONY: test-short
test-short: ## Run tests without verbose output
	$(CGO) $(GO) test $(TEST_PKGS) -count=1

.PHONY: test-race
test-race: ## Run tests with the race detector
	$(CGO) $(GO) test -race $(TEST_PKGS) -count=1

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	@printf "$(GREEN)▶ Running tests with coverage...$(RESET)\n"
	$(CGO) $(GO) test $(TEST_PKGS) -coverprofile=coverage.out -count=1
	$(GO) tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)✔ Coverage report: coverage.html$(RESET)\n"

.PHONY: vet
vet: ## Run go vet on all packages
	$(GO) vet $(TEST_PKGS)

.PHONY: lint
lint: ## Run staticcheck linter (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
	@command -v staticcheck >/dev/null 2>&1 || { echo "staticcheck not found — run: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; }
	staticcheck $(TEST_PKGS)

# =============================================================================
##@ Dependencies
# =============================================================================

.PHONY: deps
deps: ## Download and tidy Go modules
	$(GO) mod download
	$(GO) mod tidy

.PHONY: deps-update
deps-update: ## Update all Go dependencies to latest compatible versions
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: npm-install
npm-install: ## Install Node.js frontend dependencies
	cd $(FRONTEND_DIR) && npm install

# =============================================================================
##@ Docker
# =============================================================================

.PHONY: env-init
env-init: ## Copy .env.example → .env (skips if .env already exists)
	@if [ -f .env ]; then \
		printf "$(YELLOW)⚠ .env already exists — not overwriting$(RESET)\n"; \
	else \
		cp .env.example .env; \
		printf "$(GREEN)✔ Created .env from .env.example$(RESET)\n"; \
		printf "$(YELLOW)  Edit .env and set a strong POSTGRES_PASSWORD before running.$(RESET)\n"; \
	fi

# Internal guard — fails fast if .env is missing with a clear message.
.env:
	@printf "$(YELLOW)⚠ .env file not found.$(RESET)\n"
	@printf "  Run: $(CYAN)make env-init$(RESET) to create one from .env.example\n"
	@exit 1

.PHONY: docker-build
docker-build: .env ## Build Docker image
	@printf "$(GREEN)▶ Building Docker image $(IMAGE_NAME)...$(RESET)\n"
	$(COMPOSE) -f $(DC_FILE) build
	@printf "$(GREEN)✔ Docker image built$(RESET)\n"

.PHONY: docker-up
docker-up: .env ## Start all services (postgres + mpanel) in the background
	@printf "$(GREEN)▶ Starting services...$(RESET)\n"
	$(COMPOSE) -f $(DC_FILE) up -d
	@printf "$(GREEN)✔ Services started. Panel: http://localhost:2095$(RESET)\n"

.PHONY: docker-up-build
docker-up-build: .env ## Build images then start all services
	$(COMPOSE) -f $(DC_FILE) up -d --build

.PHONY: docker-down
docker-down: ## Stop and remove containers (keeps volumes)
	@printf "$(YELLOW)▶ Stopping services...$(RESET)\n"
	$(COMPOSE) -f $(DC_FILE) down
	@printf "$(GREEN)✔ Services stopped$(RESET)\n"

.PHONY: docker-down-v
docker-down-v: ## Stop containers AND remove volumes (⚠ destroys DB data)
	@printf "$(YELLOW)▶ Stopping services and removing volumes...$(RESET)\n"
	$(COMPOSE) -f $(DC_FILE) down -v
	@printf "$(GREEN)✔ Done$(RESET)\n"

.PHONY: docker-restart
docker-restart: ## Restart all services
	$(COMPOSE) -f $(DC_FILE) restart

.PHONY: docker-restart-app
docker-restart-app: ## Restart only the mpanel container
	$(COMPOSE) -f $(DC_FILE) restart mpanel

.PHONY: docker-logs
docker-logs: ## Tail live logs from all services
	$(COMPOSE) -f $(DC_FILE) logs -f

.PHONY: docker-logs-app
docker-logs-app: ## Tail live logs from the mpanel container only
	$(COMPOSE) -f $(DC_FILE) logs -f mpanel

.PHONY: docker-logs-db
docker-logs-db: ## Tail live logs from the postgres container only
	$(COMPOSE) -f $(DC_FILE) logs -f postgres

.PHONY: docker-ps
docker-ps: ## Show status of all containers
	$(COMPOSE) -f $(DC_FILE) ps

.PHONY: docker-pull
docker-pull: ## Pull latest base images
	$(COMPOSE) -f $(DC_FILE) pull

.PHONY: docker-shell
docker-shell: ## Open a shell inside the mpanel container
	$(COMPOSE) -f $(DC_FILE) exec mpanel /bin/sh

.PHONY: docker-db-shell
docker-db-shell: ## Open a psql shell inside the postgres container
	$(COMPOSE) -f $(DC_FILE) exec postgres \
		psql -U $${POSTGRES_USER:-mpanel} -d $${POSTGRES_DB:-mpanel}

# =============================================================================
##@ Database
# =============================================================================

.PHONY: db-migrate
db-migrate: ## Run schema migrations against the configured database
	@printf "$(GREEN)▶ Running migrations...$(RESET)\n"
	./$(BINARY) migrate

.PHONY: db-backup
db-backup: ## Run an immediate pg_dump backup (saves to ./backups/)
	@printf "$(GREEN)▶ Running database backup...$(RESET)\n"
	@bash scripts/backup.sh
	@printf "$(GREEN)✔ Backup complete$(RESET)\n"

.PHONY: db-restore
db-restore: ## Restore from a backup file: make db-restore FILE=backups/mpanel_xxx.sql.gz
	@test -n "$(FILE)" || { echo "$(YELLOW)Usage: make db-restore FILE=backups/<file>.sql.gz$(RESET)"; exit 1; }
	@bash scripts/restore.sh $(FILE)

.PHONY: db-backup-list
db-backup-list: ## List all available backups
	@ls -lhtr backups/mpanel_*.sql.gz 2>/dev/null || echo "$(YELLOW)No backups found in ./backups/$(RESET)"

# =============================================================================
##@ Install / Uninstall
# =============================================================================

INSTALL_DIR := /usr/local/bin
SERVICE_DIR := /etc/systemd/system

.PHONY: install
install: build ## Install the binary to $(INSTALL_DIR)
	@printf "$(GREEN)▶ Installing $(APP_NAME) to $(INSTALL_DIR)...$(RESET)\n"
	install -Dm755 $(BINARY) $(INSTALL_DIR)/$(APP_NAME)
	@printf "$(GREEN)✔ Installed$(RESET)\n"

.PHONY: install-service
install-service: install ## Install binary + systemd service
	@printf "$(GREEN)▶ Installing systemd service...$(RESET)\n"
	install -Dm644 mpanel.service $(SERVICE_DIR)/$(APP_NAME).service
	systemctl daemon-reload
	systemctl enable $(APP_NAME)
	@printf "$(GREEN)✔ Service installed. Start with: systemctl start $(APP_NAME)$(RESET)\n"

.PHONY: uninstall
uninstall: ## Remove binary and systemd service
	@printf "$(YELLOW)▶ Uninstalling $(APP_NAME)...$(RESET)\n"
	-systemctl stop $(APP_NAME) 2>/dev/null || true
	-systemctl disable $(APP_NAME) 2>/dev/null || true
	rm -f $(INSTALL_DIR)/$(APP_NAME)
	rm -f $(SERVICE_DIR)/$(APP_NAME).service
	-systemctl daemon-reload 2>/dev/null || true
	@printf "$(GREEN)✔ Uninstalled$(RESET)\n"

# =============================================================================
##@ Release / Package
# =============================================================================

.PHONY: package
package: build ## Package the binary into a tar.gz archive
	@printf "$(GREEN)▶ Packaging...$(RESET)\n"
	mkdir -p dist/$(APP_NAME)
	cp $(BINARY) dist/$(APP_NAME)/
	cp mpanel.service dist/$(APP_NAME)/
	cp mpanel.sh dist/$(APP_NAME)/
	cp .env.example dist/$(APP_NAME)/
	tar -zcvf $(APP_NAME)-$(VERSION).tar.gz -C dist $(APP_NAME)
	rm -rf dist
	@printf "$(GREEN)✔ Package: $(APP_NAME)-$(VERSION).tar.gz$(RESET)\n"

.PHONY: env-check
env-check: ## Verify required environment variables are set
	@test -n "$$POSTGRES_PASSWORD" || \
		{ echo "$(YELLOW)⚠ POSTGRES_PASSWORD is not set. Copy .env.example → .env and fill it in.$(RESET)"; exit 1; }
	@printf "$(GREEN)✔ Environment OK$(RESET)\n"

# =============================================================================
##@ Shortcuts
# =============================================================================

.PHONY: up
up: docker-up ## Alias for docker-up

.PHONY: down
down: docker-down ## Alias for docker-down

.PHONY: restart
restart: docker-restart ## Alias for docker-restart

.PHONY: logs
logs: docker-logs ## Alias for docker-logs

.PHONY: ps
ps: docker-ps ## Alias for docker-ps

.PHONY: all
all: deps vet test build ## Run deps → vet → test → build (full pipeline)
