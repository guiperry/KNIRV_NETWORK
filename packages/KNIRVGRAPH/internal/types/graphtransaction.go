package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
)

type GraphTransaction struct {
	ID            string                 `json:"id"`
	Type          GraphTxType            `json:"type"`
	NodeID        string                 `json:"node_id,omitempty"`
	EdgeID        string                 `json:"edge_id,omitempty"`
	From          string                 `json:"from"`
	To            string                 `json:"to"`
	Amount        uint64                 `json:"amount"`
	Fee           uint64                 `json:"fee"`
	Data          []byte                 `json:"data"`
	Timestamp     time.Time              `json:"timestamp"`
	Signature     string                 `json:"signature"`
	BodyBytes     string                 `json:"body_bytes"`
	AuthInfoBytes string                 `json:"auth_info_bytes"`
	Signatures    []string               `json:"signatures"`
	PublicKey     string                 `json:"public_key"`
	Address       string                 `json:"address"`
	ChainID       string                 `json:"chain_id"`
	AccountNumber uint64                 `json:"account_number"`
	Nonce         uint64                 `json:"nonce"`
	GraphData     GraphTxData            `json:"graph_data"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type GraphTxType int

const (
	CreateNodeTx GraphTxType = iota
	CreateEdgeTx
	UpdateNodeTx
	DeleteNodeTx
	DeleteEdgeTx
	GraphQueryTx
)

type GraphTxData struct {
	NodeData   *GraphNode             `json:"node_data,omitempty"`
	EdgeData   *Edge                  `json:"edge_data,omitempty"`
	QueryData  *GraphQuery            `json:"query_data,omitempty"`
	Parents    []string               `json:"parents,omitempty"`
	Children   []string               `json:"children,omitempty"`
	Weight     float64                `json:"weight,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type GraphQuery struct {
	Type      QueryType              `json:"type"`
	StartNode string                 `json:"start_node"`
	EndNode   string                 `json:"end_node,omitempty"`
	MaxDepth  int                    `json:"max_depth"`
	Filters   map[string]interface{} `json:"filters"`
	OrderBy   string                 `json:"order_by"`
	Limit     int                    `json:"limit"`
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
	signed, err := gtx.signedTransaction()
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(knirvsigning.MarshalTxRaw(signed.BodyBytes, signed.AuthInfoBytes, signed.Signatures))
	return hex.EncodeToString(hash[:])
}

func (gtx *GraphTransaction) Serialize() ([]byte, error) {
	return json.Marshal(gtx)
}

func (gtx *GraphTransaction) Verify() bool {
	return gtx.VerifyError() == nil
}

func (gtx *GraphTransaction) VerifyError() error {
	if gtx == nil {
		return fmt.Errorf("graph transaction is required")
	}
	if strings.TrimSpace(gtx.ChainID) == "" {
		return fmt.Errorf("chain_id is required")
	}
	signed, err := gtx.signedTransaction()
	if err != nil {
		return err
	}
	if err := knirvsigning.VerifyTransaction(signed, gtx.ChainID, gtx.AccountNumber); err != nil {
		return fmt.Errorf("verify SIGN_MODE_DIRECT transaction: %w", err)
	}
	sequence, err := knirvsigning.ParseSequence(signed.AuthInfoBytes)
	if err != nil || sequence != gtx.Nonce {
		return fmt.Errorf("signed sequence does not match graph transaction nonce")
	}
	action, err := knirvsigning.ParseActionBody(signed.BodyBytes)
	if err != nil {
		return err
	}
	expectedPayload, err := gtx.actionPayload()
	if err != nil {
		return err
	}
	if action.Action != fmt.Sprintf("graph.%d", gtx.Type) || action.Sender != gtx.From || action.Recipient != gtx.To || action.Amount != gtx.Amount || !bytesEqual(action.Payload, expectedPayload) {
		return fmt.Errorf("signed action does not match graph transaction")
	}
	if action.TimestampUnix != gtx.Timestamp.Unix() {
		return fmt.Errorf("signed action timestamp does not match graph transaction")
	}
	if signed.Address != gtx.From || (gtx.Address != "" && gtx.Address != gtx.From) {
		return fmt.Errorf("graph transaction sender does not match signer")
	}
	hash := gtx.Hash()
	if hash == "" {
		return fmt.Errorf("could not calculate transaction hash")
	}
	if gtx.ID != "" && !strings.EqualFold(gtx.ID, hash) {
		return fmt.Errorf("transaction ID does not match canonical hash")
	}
	return nil
}

func (gtx *GraphTransaction) signedTransaction() (knirvsigning.SignedTransaction, error) {
	wire := knirvsigning.WireTransaction{
		BodyBytes: gtx.BodyBytes, AuthInfoBytes: gtx.AuthInfoBytes, Signatures: gtx.Signatures,
		PublicKey: gtx.PublicKey, Address: gtx.Address,
	}
	if wire.Address == "" {
		wire.Address = gtx.From
	}
	return knirvsigning.ParseWire(wire)
}

func (gtx *GraphTransaction) actionPayload() ([]byte, error) {
	return json.Marshal(struct {
		NodeID    string                 `json:"node_id,omitempty"`
		EdgeID    string                 `json:"edge_id,omitempty"`
		Data      []byte                 `json:"data,omitempty"`
		Nonce     uint64                 `json:"nonce"`
		GraphData GraphTxData            `json:"graph_data"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}{gtx.NodeID, gtx.EdgeID, gtx.Data, gtx.Nonce, gtx.GraphData, gtx.Metadata})
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (gtx *GraphTransaction) CanonicalSignRequest(feeDenom, feeAmount string, gasLimit uint64) (knirvsigning.SignRequest, error) {
	payload, err := gtx.actionPayload()
	if err != nil {
		return knirvsigning.SignRequest{}, err
	}
	return knirvsigning.SignRequest{
		Action:  knirvsigning.Action{Action: fmt.Sprintf("graph.%d", gtx.Type), Sender: gtx.From, Recipient: gtx.To, Amount: gtx.Amount, Payload: payload, TimestampUnix: gtx.Timestamp.Unix()},
		ChainID: gtx.ChainID, AccountNumber: gtx.AccountNumber, Sequence: gtx.Nonce,
		Fee: knirvsigning.Fee{Denom: feeDenom, Amount: feeAmount, GasLimit: gasLimit},
	}, nil
}

func NewGraphTransaction(txType GraphTxType, from, to string, amount, fee uint64, data []byte) *GraphTransaction {
	return &GraphTransaction{
		ID:        "",
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
