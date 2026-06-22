# Capital-Hub developer tasks.
.DEFAULT_GOAL := help

VERSION ?= dev
DIST := backend/internal/web/dist

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: dev-backend
dev-backend: ## Run the Go backend (dev mode)
	cd backend && CH_ENV=dev CH_DATA_DIR=./.devdata go run ./cmd/server

.PHONY: dev-frontend
dev-frontend: ## Run the SvelteKit dev server
	cd frontend && npm run dev

.PHONY: frontend
frontend: ## Build the frontend and embed it into the backend
	cd frontend && npm ci && npm run build
	rm -rf $(DIST)
	cp -r frontend/build $(DIST)

.PHONY: build
build: frontend ## Build the production binary (frontend embedded)
	cd backend && CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o bin/capital-hub ./cmd/server

.PHONY: test
test: ## Run backend tests
	cd backend && go test ./...

.PHONY: tidy
tidy: ## Tidy Go modules
	cd backend && go mod tidy

.PHONY: docker
docker: ## Build the container image
	docker build -t capital-hub:$(VERSION) --build-arg VERSION=$(VERSION) .

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf backend/bin frontend/build frontend/.svelte-kit
	rm -rf $(DIST) && mkdir -p $(DIST) && touch $(DIST)/.gitkeep
