FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Pinned tooling versions — keep in sync with .github/workflows/ci.yml.
ARG GOLANGCI_LINT_VERSION=v2.0.2
ARG GORELEASER_VERSION=v2.5.1
ARG GOTESTSUM_VERSION=v1.12.0
ARG DELVE_VERSION=v1.23.1
# goimports must be pinned to a version compatible with the base image's Go
# minor: newer tools.cmd/goimports drag in golang.org/x/tools that bumps the
# Go floor. v0.24.0 is the last release compatible with Go 1.22+.
ARG GOIMPORTS_VERSION=v0.24.0

USER vscode

RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} \
    && go install github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION} \
    && go install gotest.tools/gotestsum@${GOTESTSUM_VERSION} \
    && go install github.com/go-delve/delve/cmd/dlv@${DELVE_VERSION} \
    && go install golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}

USER root
