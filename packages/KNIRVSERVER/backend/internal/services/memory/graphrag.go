package memory

import graphragffi "knirv-server/pkg/embedded/graphrag"

type (
	GraphRAGInterface = graphragffi.Interface
	GraphRAGQuery     = graphragffi.GraphQuery
	GraphRAGResult    = graphragffi.GraphResult
)

type GraphRAGClient struct {
	*graphragffi.Client
}

func NewGraphRAGClient() *GraphRAGClient {
	return &GraphRAGClient{Client: graphragffi.NewClient(nil)}
}
