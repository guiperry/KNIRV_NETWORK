module github.com/guiperry/knirv/llama

go 1.24.0

require github.com/guiperry/knirv-sdk-go/models v0.0.0

require (
	github.com/guiperry/knirv-sdk-go v0.1.1 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
)

// Role-frame contract shared with KNIRVINFERENCER and the daemon (KNIRVSDK nested module).
replace github.com/guiperry/knirv-sdk-go/models => ../../KNIRVSDK/go-package/models

replace github.com/guiperry/knirv-sdk-go => ../../KNIRVSDK/go-package
