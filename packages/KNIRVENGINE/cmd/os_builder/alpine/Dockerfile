# syntax=docker/dockerfile:1
# KNIRVENGINE Alpine runtime. Kernel eBPF/cgroup capabilities are granted at
# docker run time; see cmd/container_deployer/compose.ebpf.yaml.
FROM node:22-alpine AS gui-builder
RUN apk add --no-cache \
    python3 make g++ pkgconfig \
    cairo-dev pango-dev jpeg-dev giflib-dev librsvg-dev
WORKDIR /src/gui
COPY gui/package.json gui/package-lock.json ./
RUN npm ci
COPY gui/ ./
RUN npm run build

FROM golang:1.23-alpine AS engine-builder
RUN apk add --no-cache build-base git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=gui-builder /src/gui/dist ./gui/dist
RUN CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o /out/knirv-engine .

FROM alpine:3.21
RUN apk add --no-cache \
    bash bubblewrap bpftrace ca-certificates curl iproute2 iptables \
    libcap procps util-linux

# File capabilities constrain the elevated binaries. Docker must retain these
# capabilities in its bounding set (the compose profile does so explicitly).
RUN setcap cap_sys_admin,cap_sys_ptrace,cap_sys_chroot+ep /usr/bin/nsenter && \
    setcap cap_bpf,cap_perfmon,cap_sys_admin+ep /usr/bin/bpftrace

RUN addgroup -S -g 1001 knirvengine && adduser -S -D -u 1001 -G knirvengine knirvengine
WORKDIR /app
COPY --from=engine-builder --chown=knirvengine:knirvengine /out/knirv-engine /app/knirv-engine
RUN mkdir -p /app/data /app/logs /home/knirvengine/.config && \
    chown -R knirvengine:knirvengine /app /home/knirvengine
USER knirvengine

ENV HOME=/home/knirvengine XDG_CONFIG_HOME=/home/knirvengine/.config
ENV KNIRV_ENV=production KNIRV_LOG_LEVEL=info
EXPOSE 8080 8081
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 CMD curl --fail http://127.0.0.1:8081/health || exit 1
ENTRYPOINT ["/app/knirv-engine"]
CMD ["--production", "--browser"]
