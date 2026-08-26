//go:build cgo

package api

import (
	"fmt"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	goGrammar "github.com/tree-sitter/tree-sitter-go/bindings/go"
	javascriptGrammar "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	pythonGrammar "github.com/tree-sitter/tree-sitter-python/bindings/go"
	typescriptGrammar "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// parseTreeSitterNative uses grammar-generated C parsers through the official
// Go bindings, preserving real syntax-node types and positions.
func parseTreeSitterNative(source, language string) (*TreeNode, error) {
	grammar, err := treeSitterLanguage(language)
	if err != nil {
		return nil, err
	}
	parser := treesitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(grammar); err != nil {
		return nil, fmt.Errorf("set tree-sitter language: %w", err)
	}
	tree := parser.Parse([]byte(source), nil)
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter did not produce a syntax tree")
	}
	defer tree.Close()
	return treeSitterNode(tree.RootNode(), source), nil
}

func treeSitterLanguage(name string) (*treesitter.Language, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "go", "golang":
		return treesitter.NewLanguage(goGrammar.Language()), nil
	case "javascript", "js", "jsx":
		return treesitter.NewLanguage(javascriptGrammar.Language()), nil
	case "typescript", "ts":
		return treesitter.NewLanguage(typescriptGrammar.LanguageTypescript()), nil
	case "python", "py":
		return treesitter.NewLanguage(pythonGrammar.Language()), nil
	default:
		return nil, fmt.Errorf("unsupported tree-sitter language %q (supported: go, javascript, typescript, python)", name)
	}
}

func treeSitterNode(node *treesitter.Node, source string) *TreeNode {
	start, end := node.StartPosition(), node.EndPosition()
	result := &TreeNode{Type: node.Kind(), StartRow: int(start.Row), StartCol: int(start.Column), EndRow: int(end.Row), EndCol: int(end.Column)}
	if node.ChildCount() == 0 {
		startByte, endByte := int(node.StartByte()), int(node.EndByte())
		if startByte >= 0 && endByte >= startByte && endByte <= len(source) {
			result.Text = source[startByte:endByte]
		}
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		result.Children = append(result.Children, *treeSitterNode(node.Child(i), source))
	}
	return result
}
