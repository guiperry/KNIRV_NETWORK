module github.com/guiperry/knirv/llama

go 1.24.0

require github.com/guiperry/knirv-sdk-go/models v0.0.0

// Role-frame contract shared with KNIRVINFERENCER and the daemon (KNIRVSDK nested module).
replace github.com/guiperry/knirv-sdk-go/models => ../../KNIRVSDK/go-package/models
