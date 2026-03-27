package turnserver

import (
	"time"
)

const (
	OriginPublic  = "chain"
	OriginPrivate = "nrn"
)

type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Value     uint64 `json:"value"`
	Data      []byte `json:"data"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Origin    string `json:"origin,omitempty"`
}

type TurnSessionData struct {
	Type       string `json:"type"`
	Timestamp  string `json:"timestamp"`
	ClientAddr string `json:"client_addr"`
	Username   string `json:"username"`
	Realm      string `json:"realm"`
	RecordedAt string `json:"recorded_at,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	BytesSent  uint64 `json:"bytes_sent,omitempty"`
	BytesRecv  uint64 `json:"bytes_recv,omitempty"`
	Duration   int64  `json:"duration,omitempty"`
}

type NRNMintRequest struct {
	Recipient string `json:"recipient"`
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
	ProofID   string `json:"proof_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type MintingStats struct {
	MinterAddress             string    `json:"minter_address"`
	MintingEnabled            bool      `json:"minting_enabled"`
	LastUpdated               time.Time `json:"last_updated"`
	TotalTurnSessions         int       `json:"total_turn_sessions"`
	TotalNRNMints             int       `json:"total_nrn_mints"`
	TotalConnectivityRewards  int       `json:"total_connectivity_rewards"`
	TotalParticipationRewards int       `json:"total_participation_rewards"`
	TransactionPoolSize       int       `json:"transaction_pool_size"`
	DataSource                string    `json:"data_source"`
}

type TurnServerConfig struct {
	UDPPort      int
	TCPPort      int
	APIPort      int
	Realm        string
	AuthSecret   string
	PublicIP     string
	MinerAddress string
}

type TurnServerStatus struct {
	Status       string    `json:"status"`
	Running      bool      `json:"running"`
	UDPPort      int       `json:"udp_port"`
	TCPPort      int       `json:"tcp_port"`
	APIPort      int       `json:"api_port"`
	Realm        string    `json:"realm"`
	SessionCount int64     `json:"session_count"`
	ActiveRelays int64     `json:"active_relays"`
	Uptime       string    `json:"uptime"`
	Timestamp    time.Time `json:"timestamp"`
}

type ConnectivityProof struct {
	ProofID      string  `json:"proof_id"`
	NodeID       string  `json:"node_id"`
	Score        float64 `json:"score"`
	RewardAmount string  `json:"reward_amount"`
	Status       string  `json:"status"`
	Timestamp    int64   `json:"timestamp"`
}
