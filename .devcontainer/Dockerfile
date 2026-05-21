FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Pinned tooling versions — keep in sync with .github/workflows/ci.yml.
ARG GOLANGCI_LINT_VERSION=v2.0.2
ARG GORELEASER_VERSION=v2.5.1
ARG GOTESTSUM_VERSION=v1.12.0
ARG DELVE_VERSION=v1.23.1

USER vscode

RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} \
    && go install github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION} \
    && go install gotest.tools/gotestsum@${GOTESTSUM_VERSION} \
    && go install github.com/go-delve/delve/cmd/dlv@${DELVE_VERSION} \
    && go install golang.org/x/tools/cmd/goimports@latest

USER root
