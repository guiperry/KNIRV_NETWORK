module data-encoder

go 1.24.11

require (
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40
	github.com/apache/arrow/go/v14 v14.0.2
	github.com/guiperry/text-embedder v0.1.0
	github.com/joho/godotenv v1.5.1
	github.com/knirvcorp/knirvbase v1.0.7
	github.com/pkoukk/tiktoken-go v0.1.8
	gopkg.in/yaml.v3 v3.0.1
	knirvhasher v1.0.0
)

require (
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/cloudflare/circl v1.6.2 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/exp v0.0.0-20250128182459-e0ece0dbea4c // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260109210033-bd525da824e2 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
)

replace knirvhasher => ../../

replace github.com/knirvcorp/knirvbase => ../../../KNIRVBASE/go
