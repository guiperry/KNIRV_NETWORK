//go:build !cgo

package api

import "fmt"

func parseTreeSitterNative(_ string, _ string) (*TreeNode, error) {
	return nil, fmt.Errorf("tree-sitter requires a cgo-enabled KNIRVENGINE build")
}
