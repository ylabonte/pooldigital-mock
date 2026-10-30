FROM mcr.microsoft.com/devcontainers/go:2-1.27-bookworm

# Pinned tooling versions — keep in sync with .github/workflows/ci.yml.
ARG GOLANGCI_LINT_VERSION=v2.14.0
ARG GORELEASER_VERSION=v2.5.1
ARG GOTESTSUM_VERSION=v1.13.0
# Delve's release number tracks the highest Go minor it supports — older
# pins refuse to debug newer Go binaries with "Version of Delve is too old
# for Go version goX.Y.Z". Pin to the latest series that matches our Go.
ARG DELVE_VERSION=v1.27.2
# goimports comes from golang.org/x/tools, whose go directive must not exceed
# the base image's Go (GOTOOLCHAIN=local there). v0.52.0 needs Go 1.26+.
ARG GOIMPORTS_VERSION=v0.52.0

USER vscode

RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} \
    && go install github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION} \
    && go install gotest.tools/gotestsum@${GOTESTSUM_VERSION} \
    && go install github.com/go-delve/delve/cmd/dlv@${DELVE_VERSION} \
    && go install golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}

USER root
