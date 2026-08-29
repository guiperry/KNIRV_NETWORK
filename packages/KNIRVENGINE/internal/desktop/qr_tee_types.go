package desktop

import "time"

// QRLinkageInfo is a lightweight struct used by tests/mocks to represent a generated QR linkage
type QRLinkageInfo struct {
	QRCode    string    `json:"qr_code"`
	DeviceID  string    `json:"device_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// TEESession is a lightweight session representation used in tests and mocks
type TEESession struct {
	SessionID string    `json:"session_id"`
	ClientID  string    `json:"client_id"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}
