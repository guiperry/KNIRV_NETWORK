package p2p

// Block and Transaction are transport DTOs used at the KNIRVGATEWAY P2P
// boundary. Validation remains in internal/blockchain after decoding, which
// avoids maintaining a second consensus implementation in this package.
type Block struct {
	BlockNumber     uint64          `json:"block_number"`
	Timestamp       int64           `json:"timestamp"`
	Transactions    []*Transaction  `json:"transactions"`
	InvalidTxHashes map[string]bool `json:"invalid_tx_hashes"`
	PrevHash        string          `json:"prev_hash"`
	Hash            string          `json:"hash"`
}

type Transaction struct {
	TransactionHash string `json:"transaction_hash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           uint64 `json:"value"`
	Data            []byte `json:"data"`
	Timestamp       int64  `json:"timestamp"`
	Fee             uint64 `json:"fee"`
	PublicKey       string `json:"public_key,omitempty"`
	Signature       []byte `json:"signature,omitempty"`
	BodyBytes       string `json:"body_bytes,omitempty"`
	AuthInfoBytes   string `json:"auth_info_bytes,omitempty"`
	ChainID         string `json:"chain_id,omitempty"`
	AccountNumber   uint64 `json:"account_number,omitempty"`
	Sequence        uint64 `json:"sequence,omitempty"`
}
