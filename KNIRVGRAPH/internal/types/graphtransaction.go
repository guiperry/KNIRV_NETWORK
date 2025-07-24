package types

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
)

type GraphTransaction struct {
    ID          string                 `json:"id"`
    Type        GraphTxType            `json:"type"`
    NodeID      string                 `json:"node_id,omitempty"`
    EdgeID      string                 `json:"edge_id,omitempty"`
    From        string                 `json:"from"`
    To          string                 `json:"to"`
    Amount      uint64                 `json:"amount"`
    Fee         uint64                 `json:"fee"`
    Data        []byte                 `json:"data"`
    Timestamp   time.Time              `json:"timestamp"`
    Signature   string                 `json:"signature"`
    Nonce       uint64                 `json:"nonce"`
    GraphData   GraphTxData            `json:"graph_data"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type GraphTxType int

const (
    CreateNodeTx GraphTxType = iota
    CreateEdgeTx
    UpdateNodeTx
    DeleteNodeTx
    DeleteEdgeTx
    TransferTx
    ValidatorTx
    GraphQueryTx
)

type GraphTxData struct {
    NodeData     *GraphNode             `json:"node_data,omitempty"`
    EdgeData     *Edge                  `json:"edge_data,omitempty"`
    QueryData    *GraphQuery            `json:"query_data,omitempty"`
    Parents      []string               `json:"parents,omitempty"`
    Children     []string               `json:"children,omitempty"`
    Weight       float64                `json:"weight,omitempty"`
    Properties   map[string]interface{} `json:"properties,omitempty"`
}

type GraphQuery struct {
    Type        QueryType              `json:"type"`
    StartNode   string                 `json:"start_node"`
    EndNode     string                 `json:"end_node,omitempty"`
    MaxDepth    int                    `json:"max_depth"`
    Filters     map[string]interface{} `json:"filters"`
    OrderBy     string                 `json:"order_by"`
    Limit       int                    `json:"limit"`
}

type QueryType int

const (
    FindPath QueryType = iota
    FindNeighbors
    TraverseGraph
    FindShortestPath
    FindAllPaths
    GetSubgraph
)

func (gtx *GraphTransaction) Hash() string {
    data, _ := json.Marshal(gtx)
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

func (gtx *GraphTransaction) Serialize() ([]byte, error) {
    return json.Marshal(gtx)
}

func (gtx *GraphTransaction) Verify() bool {
    // Implement signature verification logic for graph transactions
    // This would include validating graph-specific constraints
    return true // Placeholder
}

func NewGraphTransaction(txType GraphTxType, from, to string, amount, fee uint64, data []byte) *GraphTransaction {
    return &GraphTransaction{
        ID:        generateGraphTxID(),
        Type:      txType,
        From:      from,
        To:        to,
        Amount:    amount,
        Fee:       fee,
        Data:      data,
        Timestamp: time.Now(),
        Nonce:     0,
        GraphData: GraphTxData{},
        Metadata:  make(map[string]interface{}),
    }
}

func generateGraphTxID() string {
    hash := sha256.Sum256([]byte(time.Now().String()))
    return hex.EncodeToString(hash[:])[:16]
}

// Graph-specific transaction creation helpers
func NewCreateNodeTransaction(from string, nodeData *GraphNode, fee uint64) *GraphTransaction {
    tx := NewGraphTransaction(CreateNodeTx, from, "", 0, fee, []byte{})
    tx.NodeID = nodeData.ID
    tx.GraphData.NodeData = nodeData
    return tx
}

func NewCreateEdgeTransaction(from string, edgeData *Edge, fee uint64) *GraphTransaction {
    tx := NewGraphTransaction(CreateEdgeTx, from, "", 0, fee, []byte{})
    tx.EdgeID = edgeData.ID
    tx.GraphData.EdgeData = edgeData
    return tx
}

func NewGraphQueryTransaction(from string, query *GraphQuery, fee uint64) *GraphTransaction {
    tx := NewGraphTransaction(GraphQueryTx, from, "", 0, fee, []byte{})
    tx.GraphData.QueryData = query
    return tx
}