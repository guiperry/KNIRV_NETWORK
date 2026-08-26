package api

import (
	"encoding/json"
	"fmt"
	"os"
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

// parseWithTreeSitter parses source code into a TreeNode tree. There is no
// heuristic fallback: presenting a line classifier as an AST makes syntax
// analysis look successful when it was not.
func parseWithTreeSitter(source, language string) (*TreeNode, error) {
	return parseTreeSitterNative(source, language)
}
