# syntax=docker/dockerfile:1
#
# KNIRVGATEWAY — Debian bookworm image.
#
# The build context MUST be KNIRV_NETWORK/packages (not this directory): go.mod
# `replace`s the SDK from ../KNIRVSDK/go-package.
#
#   cd KNIRV_NETWORK/packages
#   docker build -f KNIRVGATEWAY/Dockerfile -t knirv-gateway .
#
# The SDK's Rust core runs inside the gateway as a WebAssembly (WASI) module under
# wazero, so the gateway builds with CGO_ENABLED=0: no C toolchain, no native
# library. The Go SDK does not embed that module: it loads it on first use from
# the user data directory, falling back to a download from releases.knirv.com.
# Stage 1 compiles the module from source; stage 2 pins the SDK to the digest of
# what stage 1 built (stamp.go), and the runtime image ships the file in its data
# directory, so this image never needs the network for the core and never trusts a
# digest it did not build.
#
# The runtime base is Debian bookworm by default (KNIRVSERVER's base). To run on
# Kali instead:
#   docker build --build-arg RUNTIME_BASE=kalilinux/kali-rolling ...

ARG GO_VERSION=1.26
ARG RUST_VERSION=1
ARG RUNTIME_BASE=debian:bookworm-slim

# ---- Stage 1: the SDK's Rust core, as a WASI module ----
FROM rust:${RUST_VERSION}-bookworm AS sdk-core
RUN rustup target add wasm32-wasip1
WORKDIR /src
COPY KNIRVSDK/core KNIRVSDK/core
COPY KNIRVSDK/bindings/wasi KNIRVSDK/bindings/wasi
RUN cargo build --release --locked --target wasm32-wasip1 --manifest-path KNIRVSDK/bindings/wasi/Cargo.toml

# ---- Stage 2: the gateway binary (pure Go, no CGO) ----
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY KNIRVSDK/go-package KNIRVSDK/go-package
# Pin the Go SDK to the module stage 1 built (replaces the release digest in stamp.go).
COPY --from=sdk-core /src/KNIRVSDK/bindings/wasi/target/wasm32-wasip1/release/knirv_core.wasm /opt/knirv-sdk/knirv_core.wasm
RUN sha=$(sha256sum /opt/knirv-sdk/knirv_core.wasm | cut -d' ' -f1) \
 && sed -i "s/Digest = \"[0-9a-f]*\"/Digest = \"$sha\"/" KNIRVSDK/go-package/internal/core/stamp.go \
 && version=$(sed -n 's/.*Version = "\(.*\)"/\1/p' KNIRVSDK/go-package/internal/core/stamp.go) \
 && mkdir -p /out/sdk/$version && cp /opt/knirv-sdk/knirv_core.wasm /out/sdk/$version/knirv_core.wasm
COPY KNIRVGATEWAY/go.mod KNIRVGATEWAY/go.sum KNIRVGATEWAY/
RUN cd KNIRVGATEWAY && go mod download
COPY KNIRVGATEWAY KNIRVGATEWAY
RUN cd KNIRVGATEWAY && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/knirvgateway ./cmd/gateway

# ---- Stage 3: runtime ----
FROM ${RUNTIME_BASE}
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 1000 knirv \
 && useradd --uid 1000 --gid knirv --create-home --shell /usr/sbin/nologin knirv

WORKDIR /app
COPY --from=build /out/knirvgateway /app/knirvgateway
# The SDK core this binary is pinned to (see stage 2); no download at run time.
COPY --from=build /out/sdk /opt/knirv/sdk
RUN mkdir -p /app/data /app/logs && chown -R knirv:knirv /app
USER knirv

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -fsS http://localhost:8080/health || exit 1

ENV KNIRV_SDK_DATA_DIR=/opt/knirv/sdk \
    PORT=8080 \
    GATEWAY_MODE=persistent \
    KNIRV_CHAIN_ID=testnet

CMD ["/app/knirvgateway"]
