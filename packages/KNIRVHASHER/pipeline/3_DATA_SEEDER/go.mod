module github.com/lab/hasher/data-seeder

go 1.24.11

require (
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40
	github.com/xitongsys/parquet-go v1.6.2
	github.com/xitongsys/parquet-go-source v0.0.0-20241021075129-b732d2ac9c9b
	knirvhasher v0.0.0
)

require (
	github.com/apache/thrift v0.20.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
)

replace knirvhasher => ../..
