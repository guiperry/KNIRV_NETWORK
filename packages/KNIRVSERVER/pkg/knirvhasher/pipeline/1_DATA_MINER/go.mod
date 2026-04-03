module data-miner

go 1.23

toolchain go1.24.1

require (
	github.com/am-sokolov/go-spacy v0.0.0-20250919212123-1d3a142ac336
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40
	github.com/apache/arrow/go/v18 v18.0.0-20241007013041-ab95a4d25142
	github.com/vbauerster/mpb/v8 v8.7.4
	go.etcd.io/bbolt v1.4.0
)

replace github.com/am-sokolov/go-spacy => ./spacy

require (
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
)

require (
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/joho/godotenv v1.5.1
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.29.0 // indirect
)
