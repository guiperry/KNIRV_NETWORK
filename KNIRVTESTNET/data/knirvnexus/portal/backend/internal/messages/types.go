package messages

import "time"

// BlockchainEventMsg represents a blockchain event message
type BlockchainEventMsg struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NetworkUpdateMsg represents a network update message
type NetworkUpdateMsg struct {
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	PeerCount     int                    `json:"peer_count"`
	Latency       time.Duration          `json:"latency"`
	UploadSpeed   float64                `json:"upload_speed"`
	DownloadSpeed float64                `json:"download_speed"`
}

// LogMsg represents a log message
type LogMsg struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Component string                 `json:"component"`
	Fields    map[string]interface{} `json:"fields"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}
