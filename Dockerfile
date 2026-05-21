# syntax=docker/dockerfile:1.7

# Stage 1: build the static binary inside a Go toolchain image.
FROM golang:1.24-alpine AS build

WORKDIR /src

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/pooldigital-mock \
    ./cmd/pooldigital-mock

# Stage 2: distroless runtime image — no shell, no package manager.
FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.title="pooldigital-mock"
LABEL org.opencontainers.image.description="Combined ProCon.IP + Violet pool-controller mocks"
LABEL org.opencontainers.image.source="https://github.com/yannicschroeer/pooldigital-mock"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=build /out/pooldigital-mock /pooldigital-mock

EXPOSE 8080 8180

USER nonroot:nonroot

ENTRYPOINT ["/pooldigital-mock"]
