// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package proto

// This file contains go:generate directives for compiling .proto files
// into Go code using protoc.
//
// Usage:
//   cd backend/internal/proto
//   go generate
//
// Requirements:
//   - protoc: apt-get install protobuf-compiler
//   - protoc-gen-go: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//   - protoc-gen-go-grpc: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative hasher/hasher.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative root_key.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative bootnode_key.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative blockchain/blockchain.proto
