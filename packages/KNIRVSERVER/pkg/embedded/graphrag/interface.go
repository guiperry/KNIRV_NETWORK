package graphrag

import "context"

type Interface interface {
	Query(ctx context.Context, kbID string, query *GraphQuery) (*GraphResult, error)
	BuildIndex(ctx context.Context, kbID string, strategy string) (*IndexStatus, error)
	GetIndexStatus(ctx context.Context, kbID string) (*IndexStatus, error)
	IndexDocument(ctx context.Context, docID string, content []byte) error
	IndexDocumentWithResult(ctx context.Context, docID string, content []byte) ([]byte, error)
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)
	Close() error
}
