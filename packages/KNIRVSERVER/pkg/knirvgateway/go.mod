module github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY

go 1.24.0

toolchain go1.24.11

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/sessions v1.2.2
	github.com/joho/godotenv v1.5.1
	github.com/pion/turn/v2 v2.1.6
	github.com/rs/cors v1.10.1
	go.uber.org/zap v1.26.0
)

require github.com/stripe/stripe-go/v79 v79.12.0

require (
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/stun v0.6.1 // indirect
	github.com/pion/transport/v2 v2.2.1 // indirect
)

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/securecookie v1.1.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.36.0
	golang.org/x/sys v0.36.0 // indirect
)
