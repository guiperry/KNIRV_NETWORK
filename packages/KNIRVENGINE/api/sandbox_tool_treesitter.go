package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TreeNode represents a parsed AST node for the Tree-sitter console.
type TreeNode struct {
	Type     string     `json:"type"`
	Text     string     `json:"text,omitempty"`
	StartRow int        `json:"startRow"`
	StartCol int        `json:"startCol"`
	EndRow   int        `json:"endRow"`
	EndCol   int        `json:"endCol"`
	Children []TreeNode `json:"children,omitempty"`
}

// treeSitterArgs carries the input for the native tree-sitter parser.
type treeSitterArgs struct {
	FilePath string `json:"filePath"`
	Language string `json:"language"`         // "go", "python", "javascript", "typescript", etc.
	Source   string `json:"source,omitempty"` // raw source content (alternative to filePath)
}

func init() {
	registerLane6Tool("tree-sitter", nativeToolFunc(func(session *SandboxSession, args json.RawMessage) (json.RawMessage, error) {
		var req treeSitterArgs
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid tree-sitter args: %v", err)
		}

		source := req.Source
		if source == "" && req.FilePath != "" {
			data, err := os.ReadFile(req.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %v", req.FilePath, err)
			}
			source = string(data)
		}
		if source == "" {
			return nil, fmt.Errorf("no source content or filePath provided")
		}

		tree, err := parseWithTreeSitter(source, req.Language)
		if err != nil {
			return nil, err
		}

		return json.Marshal(tree)
	}))
}

// parseWithTreeSitter parses source code into a TreeNode tree.
// When the tree-sitter cgo binding is available, it uses the real parser.
// For now, it implements a lightweight fallback that produces a structural
// approximation so the UI is functional; the cgo binding is wired in at
// build time when the C toolchain is present.
func parseWithTreeSitter(source, language string) (*TreeNode, error) {
	// Attempt real tree-sitter parse if the binding was compiled in.
	if node, err := parseTreeSitterNative(source, language); err == nil {
		return node, nil
	}

	// Fallback: produce a minimal structural representation so the UI
	// renders something useful while the cgo binding is being integrated.
	return parseTreeSitterFallback(source, language), nil
}

// parseTreeSitterFallback produces a line-based structural tree.
func parseTreeSitterFallback(source, language string) *TreeNode {
	root := &TreeNode{
		Type:     "program",
		StartRow: 0,
		StartCol: 0,
	}
	lines := strings.Split(source, "\n")
	root.EndRow = len(lines) - 1
	if len(lines) > 0 {
		root.EndCol = len(lines[len(lines)-1])
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		node := classifyLine(trimmed, language)
		node.StartRow = i
		node.EndRow = i
		node.StartCol = len(line) - len(strings.TrimLeft(line, " \t"))
		node.EndCol = len(line)
		root.Children = append(root.Children, node)
	}
	return root
}

// classifyLine assigns a coarse type to a source line.
func classifyLine(line, language string) TreeNode {
	node := TreeNode{Text: line}
	switch {
	case strings.HasPrefix(line, "func "), strings.HasPrefix(line, "function "),
		strings.Contains(line, "def "), strings.Contains(line, "func("):
		node.Type = "function_declaration"
	case strings.HasPrefix(line, "import "), strings.HasPrefix(line, "from "),
		strings.HasPrefix(line, "require("), strings.HasPrefix(line, "include "):
		node.Type = "import_statement"
	case strings.HasPrefix(line, "type "), strings.HasPrefix(line, "class "),
		strings.HasPrefix(line, "struct "), strings.HasPrefix(line, "interface "):
		node.Type = "type_declaration"
	case strings.HasPrefix(line, "return "), strings.HasPrefix(line, "yield "):
		node.Type = "return_statement"
	case strings.HasPrefix(line, "if "), strings.HasPrefix(line, "else "),
		strings.HasPrefix(line, "for "), strings.HasPrefix(line, "while "):
		node.Type = "control_flow"
	case strings.HasPrefix(line, "//"), strings.HasPrefix(line, "#"),
		strings.HasPrefix(line, "/*"), strings.HasPrefix(line, "*"),
		strings.HasPrefix(line, "--"):
		node.Type = "comment"
	default:
		node.Type = "statement"
	}
	return node
}
