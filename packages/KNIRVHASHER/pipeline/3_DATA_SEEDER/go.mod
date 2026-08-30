module github.com/lab/hasher/data-seeder

go 1.24.11

require (
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40
	github.com/xitongsys/parquet-go v1.6.2
	github.com/xitongsys/parquet-go-source v0.0.0-20241021075129-b732d2ac9c9b
	// Keep the gRPC runtime compatible with Arrow's pre-module-split genproto
	// dependency while supporting the generated ASIC client (requires >=1.32).
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	knirvhasher v0.0.0
)

require (
	github.com/apache/thrift v0.20.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto v0.0.0-20220401170504-314d38edb7de // indirect
)

replace knirvhasher => ../..

// Arrow v0 uses the monolithic genproto module. Excluding the newer split
// module prevents both modules from supplying the same RPC packages.
exclude google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57
