package types

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
)

type GraphNode struct {
    ID           string                 `json:"id"`
    Parents      []string               `json:"parents"`      // Multiple parent nodes
    Children     []string               `json:"children"`     // Multiple child nodes
    Data         GraphData              `json:"data"`
    Timestamp    time.Time              `json:"timestamp"`
    Hash         string                 `json:"hash"`
    Validators   []string               `json:"validators"`
    Weight       float64                `json:"weight"`
    Level        uint64                 `json:"level"`       // Graph level/depth
    Metadata     map[string]interface{} `json:"metadata"`
}

type GraphData struct {
    Transactions []Transaction          `json:"transactions"`
    StateChanges []StateChange          `json:"state_changes"`
    Edges        []Edge                 `json:"edges"`
    Payload      map[string]interface{} `json:"payload"`
}

type Edge struct {
    ID       string                 `json:"id"`
    From     string                 `json:"from"`
    To       string                 `json:"to"`
    Weight   float64                `json:"weight"`
    Type     EdgeType               `json:"type"`
    Metadata map[string]interface{} `json:"metadata"`
    Created  time.Time              `json:"created"`
}

type EdgeType int

const (
    TransactionEdge EdgeType = iota
    ValidationEdge
    ConsensusEdge
    StateEdge
    CustomEdge
)

type StateChange struct {
    Type      StateChangeType        `json:"type"`
    Key       string                 `json:"key"`
    OldValue  interface{}            `json:"old_value"`
    NewValue  interface{}            `json:"new_value"`
    Timestamp time.Time              `json:"timestamp"`
}

type StateChangeType int

const (
    AccountBalance StateChangeType = iota
    AccountNonce
    ContractState
    ValidatorSet
)

func (gn *GraphNode) CalculateHash() string {
    data := struct {
        ID        string    `json:"id"`
        Parents   []string  `json:"parents"`
        Data      GraphData `json:"data"`
        Timestamp time.Time `json:"timestamp"`
        Level     uint64    `json:"level"`
    }{
        ID:        gn.ID,
        Parents:   gn.Parents,
        Data:      gn.Data,
        Timestamp: gn.Timestamp,
        Level:     gn.Level,
    }
    
    jsonData, _ := json.Marshal(data)
    hash := sha256.Sum256(jsonData)
    return hex.EncodeToString(hash[:])
}

func (gn *GraphNode) Serialize() ([]byte, error) {
    return json.Marshal(gn)
}

func (gn *GraphNode) AddParent(parentID string) {
    for _, p := range gn.Parents {
        if p == parentID {
            return // Already exists
        }
    }
    gn.Parents = append(gn.Parents, parentID)
}

func (gn *GraphNode) AddChild(childID string) {
    for _, c := range gn.Children {
        if c == childID {
            return // Already exists
        }
    }
    gn.Children = append(gn.Children, childID)
}

func (gn *GraphNode) RemoveParent(parentID string) {
    for i, p := range gn.Parents {
        if p == parentID {
            gn.Parents = append(gn.Parents[:i], gn.Parents[i+1:]...)
            return
        }
    }
}

func (gn *GraphNode) RemoveChild(childID string) {
    for i, c := range gn.Children {
        if c == childID {
            gn.Children = append(gn.Children[:i], gn.Children[i+1:]...)
            return
        }
    }
}

func NewGraphNode(id string, parents []string, data GraphData) *GraphNode {
    node := &GraphNode{
        ID:        id,
        Parents:   parents,
        Children:  []string{},
        Data:      data,
        Timestamp: time.Now(),
        Validators: []string{},
        Weight:    1.0,
        Level:     0,
        Metadata:  make(map[string]interface{}),
    }
    
    node.Hash = node.CalculateHash()
    return node
}

func (e *Edge) Serialize() ([]byte, error) {
    return json.Marshal(e)
}

func NewEdge(from, to string, edgeType EdgeType, weight float64) *Edge {
    return &Edge{
        ID:       generateEdgeID(from, to),
        From:     from,
        To:       to,
        Weight:   weight,
        Type:     edgeType,
        Metadata: make(map[string]interface{}),
        Created:  time.Now(),
    }
}

func generateEdgeID(from, to string) string {
    data := from + "->" + to + time.Now().String()
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])[:16]
}

func generateNodeID() string {
    hash := sha256.Sum256([]byte(time.Now().String()))
    return hex.EncodeToString(hash[:])[:16]
}