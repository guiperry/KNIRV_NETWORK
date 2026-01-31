FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev
COPY go.mod go.sum ./
COPY KNIRVBASE ./KNIRVBASE
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o knirvbase ./cmd/node

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/knirvbase .
COPY config.yaml ./config.yaml
EXPOSE 8080 8081 9090
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --spider http://localhost:8081/health || exit 1
CMD ["./knirvbase"]