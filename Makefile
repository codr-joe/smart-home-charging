.PHONY: help dev-db dev-api dev-web mock-p1 test test-api test-web lint-api lint-web setup \
        build build-api build-web push push-api push-web release test-build

API_DIR    := src/api
WEB_DIR    := src/web
REGISTRY   := harbor.hooyberghs.eu/smart-charging
TAG        ?= latest
API_IMAGE  := $(REGISTRY)/api:$(TAG)
WEB_IMAGE  := $(REGISTRY)/web:$(TAG)

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ── Setup ────────────────────────────────────────────────────────────────────

setup: ## Copy .env.example files and install web dependencies
	@if [ ! -f $(API_DIR)/.env ]; then \
		cp $(API_DIR)/.env.example $(API_DIR)/.env; \
		echo "Created $(API_DIR)/.env — review and adjust values"; \
	fi
	@if [ ! -f $(WEB_DIR)/.env ]; then \
		cp $(WEB_DIR)/.env.example $(WEB_DIR)/.env; \
		echo "Created $(WEB_DIR)/.env — review and adjust values"; \
	fi
	cd $(WEB_DIR) && npm install

# ── Infrastructure ────────────────────────────────────────────────────────────

dev-db: ## Start the TimescaleDB container (runs in background)
	docker compose up -d --wait

dev-db-stop: ## Stop and remove the TimescaleDB container
	docker compose down

# ── Mock P1 Meter ─────────────────────────────────────────────────────────────

mock-p1: ## Run the simulated HomeWizard P1 meter on :8090
	cd $(API_DIR) && go run ./cmd/p1mock

# ── Application ───────────────────────────────────────────────────────────────

dev-api: ## Run the API server (requires dev-db and mock-p1 or real P1 meter)
	cd $(API_DIR) && go run ./cmd/server

dev-web: ## Start the SvelteKit development server
	cd $(WEB_DIR) && npm run dev

# ── Tests ─────────────────────────────────────────────────────────────────────

test: test-api test-web ## Run all tests

test-api: ## Run Go unit tests
	cd $(API_DIR) && go test ./...

test-web: ## Run frontend unit tests
	cd $(WEB_DIR) && npm test

# ── Lint ──────────────────────────────────────────────────────────────────────

lint: lint-api lint-web ## Run all linters

lint-api: ## Lint and vet Go code
	cd $(API_DIR) && go vet ./...

lint-web: ## Lint and format-check the frontend
	cd $(WEB_DIR) && npm run lint

# ── Container Build & Push ────────────────────────────────────────────────────

build: build-api build-web ## Build all container images

build-api: ## Build the API container image
	docker build -t $(API_IMAGE) $(API_DIR)

build-web: ## Build the web container image
	docker build -t $(WEB_IMAGE) $(WEB_DIR)

push: push-api push-web ## Push all container images to Harbor

push-api: ## Push the API container image to Harbor
	docker push $(API_IMAGE)

push-web: ## Push the web container image to Harbor
	docker push $(WEB_IMAGE)

release: ## Prompt for a version tag, build, tag as latest, and push all images
	@read -p "Version tag (e.g. 1.0.1): " tag; \
	if [ -z "$$tag" ]; then echo "Error: version tag is required"; exit 1; fi; \
	$(MAKE) build TAG=$$tag; \
	docker tag $(REGISTRY)/api:$$tag $(REGISTRY)/api:latest; \
	docker tag $(REGISTRY)/web:$$tag $(REGISTRY)/web:latest; \
	$(MAKE) push TAG=$$tag; \
	$(MAKE) push TAG=latest

test-build: ## Test the container build Makefile targets
	bash tests/build_test.sh
