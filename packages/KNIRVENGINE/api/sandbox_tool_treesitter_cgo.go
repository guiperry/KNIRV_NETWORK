package api

import "fmt"

// parseTreeSitterNative attempts to parse source code using the tree-sitter
// cgo binding. Returns an error if the binding was not compiled in (no C
// toolchain at build time) or if the language grammar is unavailable.
//
// When the tree-sitter C library and Go bindings are available (github.com/
// tree-sitter/go-tree-sitter linked via cgo), this function performs a real
// parse and returns the AST. The build tag `tree_sitter` gates the cgo import.
func parseTreeSitterNative(source, language string) (*TreeNode, error) {
	return nil, fmt.Errorf("tree-sitter cgo binding not compiled in (build with -tags tree_sitter and gcc)")
}
