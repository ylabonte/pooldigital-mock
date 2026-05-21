BIN := pooldigital-mock
PKG := ./...
COVER := coverage.out
COVER_MIN ?= 85

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/pooldigital-mock

.PHONY: run
run: ## Run the binary
	go run ./cmd/pooldigital-mock

.PHONY: test
test: ## Run tests with race detector
	go test -race $(PKG)

.PHONY: cover
cover: ## Run tests with coverage and enforce floor
	go test -race -coverprofile=$(COVER) -covermode=atomic $(PKG)
	@go tool cover -func=$(COVER) | tail -n 1
	@total=$$(go tool cover -func=$(COVER) | awk '/^total:/ {print $$3}' | tr -d '%'); \
	awk -v t="$$total" -v m="$(COVER_MIN)" 'BEGIN { exit (t+0 < m+0) ? 1 : 0 }' || \
		(echo "coverage $$total% < $(COVER_MIN)%" && exit 1)

.PHONY: cover-html
cover-html: cover ## Render coverage HTML report
	go tool cover -html=$(COVER) -o coverage.html
	@echo "open coverage.html"

.PHONY: vet
vet: ## go vet
	go vet $(PKG)

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## gofmt -s -w
	gofmt -s -w .

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: docker
docker: ## Build the production Docker image
	docker build -t $(BIN):dev .

.PHONY: docker-run
docker-run: docker ## Run the production Docker image with both ports forwarded
	docker run --rm -p 8080:8080 -p 8180:8180 $(BIN):dev

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BIN) $(COVER) coverage.html
	rm -rf dist
