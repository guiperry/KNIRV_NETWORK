package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// relayMessage is the JSON envelope POSTed to KNIRVSERVER's backend socket
// for every bridged inbound message. Per the design doc, this "rides the
// same direction traffic already flows in" (the existing backend Unix
// socket, not a new port). The backend_server handler
// (POST /api/v1/bridge/messages, KNIRV_CORP/packages/server/backend_server/
// internal/web/bridge_message_handlers.go) records the message and reports
// whether a live CLI Supervisor session exists for the customer — it does
// not yet dispatch into that session, since that requires the signed relay
// pipeline bound to the customer's own signing key (see that handler's doc
// comment). The message is always mirrored into the Matrix room regardless
// of what the backend reports, so nothing is silently lost either way.
type relayMessage struct {
	CustomerID string    `json:"customer_id"`
	Channel    string    `json:"channel"`
	ChatID     string    `json:"chat_id"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Text       string    `json:"text"`
	Timestamp  time.Time `json:"timestamp"`
}

// BackendRelay delivers bridged messages into KNIRVSERVER's Expert-Advisor
// -> Supervisor pipeline (Phase 3) over the same backend Unix socket every
// other gateway-to-backend call already uses, and accepts the resulting
// responses back out via the bridge's own /api/bridge/relay/outbound
// endpoint (handlers.go).
type BackendRelay struct {
	client     *http.Client
	token      string
	configured bool
}

// NewBackendRelay builds a relay client dialing the backend's Unix socket
// directly, attaching internalAuthToken as X-KNIRV-Internal-Token on every
// request — the same shared service-to-service secret
// requireBridgeInternalToken checks on the backend side.
func NewBackendRelay(socketPath, internalAuthToken string) *BackendRelay {
	if strings.TrimSpace(socketPath) == "" {
		return &BackendRelay{configured: false}
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &BackendRelay{
		client:     &http.Client{Transport: transport, Timeout: 15 * time.Second},
		token:      internalAuthToken,
		configured: true,
	}
}

// relayAck is backend_server's response body: whether the message was
// persisted, whether a live CLI Supervisor session existed for the customer,
// whether the backend's signed cli_supervisor relay
// (bridge_message_handlers.go's dispatchAndCapture) actually delivered the
// text as PTY input, and — when it did — whatever the agent streamed back
// during the capture window (ReplyText, already ANSI-stripped backend-side;
// empty if nothing arrived before the capture window closed).
type relayAck struct {
	Stored              bool   `json:"stored"`
	SupervisorConnected bool   `json:"supervisor_connected"`
	Delivered           bool   `json:"delivered"`
	SupervisorMessage   string `json:"supervisor_message"`
	ReplyText           string `json:"reply_text"`
}

// PostInbound forwards one bridged message to the backend. A non-2xx
// response or a dial failure is returned to the caller to log, not panic on
// — the bridge's job is to relay best-effort, matching this codebase's
// established posture for optional/cross-service calls (e.g. the CLI
// AdvisorClient's "never block the real session" convention).
func (r *BackendRelay) PostInbound(ctx context.Context, msg InboundMessage) (relayAck, error) {
	if !r.configured {
		return relayAck{}, fmt.Errorf("backend relay not configured (BACKEND_SOCKET_PATH unset)")
	}
	body, err := json.Marshal(relayMessage{
		CustomerID: msg.CustomerID,
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
		SenderID:   msg.SenderID,
		SenderName: msg.SenderName,
		Text:       msg.Text,
		Timestamp:  msg.Timestamp,
	})
	if err != nil {
		return relayAck{}, fmt.Errorf("marshal relay message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://knirvserver/api/v1/bridge/messages", bytes.NewReader(body))
	if err != nil {
		return relayAck{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.token != "" {
		req.Header.Set("X-KNIRV-Internal-Token", r.token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return relayAck{}, fmt.Errorf("backend unreachable: %w", err)
	}
	defer resp.Body.Close()
	var ack relayAck
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&ack)
	if resp.StatusCode >= 300 {
		return ack, fmt.Errorf("backend returned %d for bridged message", resp.StatusCode)
	}
	return ack, nil
}
