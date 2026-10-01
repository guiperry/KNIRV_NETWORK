package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

// InboundMessage is one message arriving on a bridged external channel,
// normalized for the relay to KNIRVSERVER's Expert-Advisor -> Supervisor
// pipeline (see relay.go) and for mirroring into the Matrix room (see
// appservice.go).
type InboundMessage struct {
	CustomerID string
	Channel    string // "whatsapp"
	ChatID     string // whatsmeow JID string for the chat (1:1 or group)
	SenderID   string // whatsmeow JID string for the actual sender
	SenderName string
	Text       string
	Timestamp  time.Time
}

// InboundHandler receives every bridged message, from every channel and
// customer, after WhatsAppManager has normalized it.
type InboundHandler func(ctx context.Context, msg InboundMessage)

// WhatsAppManager owns every customer's WhatsApp multi-device session. One
// bridge deployment hosts many customers' bridged WhatsApp accounts (Phase
// 4's multi-tenancy decision) — each customer does their own QR login, and
// per-customer whatsmeow.Client instances are kept in-process here, keyed by
// KNIRV customer ID rather than by phone number so a customer never needs to
// know or expose the underlying WhatsApp JID.
type WhatsAppManager struct {
	container *sqlstore.Container
	store     *bridgeStore
	logger    *zap.Logger
	onInbound InboundHandler

	mu      sync.Mutex
	clients map[string]*whatsmeow.Client // customerID -> client
}

func NewWhatsAppManager(db *sql.DB, store *bridgeStore, logger *zap.Logger, onInbound InboundHandler) *WhatsAppManager {
	container := sqlstore.NewWithDB(db, "sqlite", newZapWaLogger(logger, "whatsmeow/store"))
	return &WhatsAppManager{
		container: container,
		store:     store,
		logger:    logger,
		onInbound: onInbound,
		clients:   make(map[string]*whatsmeow.Client),
	}
}

// Init runs whatsmeow's own schema migrations against the shared database.
// Must run once before StartLogin/RestoreSessions.
func (m *WhatsAppManager) Init(ctx context.Context) error {
	if err := m.container.Upgrade(ctx); err != nil {
		return fmt.Errorf("upgrade whatsmeow store: %w", err)
	}
	return nil
}

// RestoreSessions reconnects every previously-paired customer session on
// gateway startup, using bridge_whatsapp_sessions (populated by StartLogin)
// to know which stored whatsmeow device belongs to which customer.
func (m *WhatsAppManager) RestoreSessions(ctx context.Context) error {
	sessions, err := m.store.WhatsAppSessions(ctx)
	if err != nil {
		return fmt.Errorf("load whatsapp sessions: %w", err)
	}
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("load whatsapp devices: %w", err)
	}

	byJID := make(map[string]*store.Device, len(devices))
	for _, device := range devices {
		if device.ID != nil {
			byJID[device.ID.String()] = device
		}
	}

	for customerID, jid := range sessions {
		device, ok := byJID[jid]
		if !ok {
			m.logger.Warn("Recorded WhatsApp session has no matching device — skipping",
				zap.String("customerID", customerID), zap.String("jid", jid))
			continue
		}
		client := whatsmeow.NewClient(device, newZapWaLogger(m.logger, "whatsmeow/client/"+customerID))
		m.registerHandlers(client, customerID)
		if err := client.Connect(); err != nil {
			m.logger.Warn("Failed to reconnect WhatsApp session",
				zap.String("customerID", customerID), zap.Error(err))
			continue
		}
		m.mu.Lock()
		m.clients[customerID] = client
		m.mu.Unlock()
		m.logger.Info("Restored WhatsApp session", zap.String("customerID", customerID), zap.String("jid", jid))
	}
	return nil
}

// StartLogin begins QR pairing for a customer with no existing session. The
// returned channel yields the same items whatsmeow.Client.GetQRChannel does
// ("code" events with a fresh QR string, then a terminal "success"/"error"/
// "timeout" event); the caller (handlers.go) turns "code" events into
// whatever KNIRVCONTROLLER's existing QR-pairing UI renders — deliberately
// reusing that UX per Phase 4's "real interface synergy, not a coincidence"
// design note, rather than inventing a new pairing flow.
func (m *WhatsAppManager) StartLogin(ctx context.Context, customerID string) (<-chan whatsmeow.QRChannelItem, error) {
	m.mu.Lock()
	if _, exists := m.clients[customerID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("customer %s already has an active WhatsApp session", customerID)
	}
	m.mu.Unlock()

	device := m.container.NewDevice()
	client := whatsmeow.NewClient(device, newZapWaLogger(m.logger, "whatsmeow/client/"+customerID))

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("open QR channel: %w", err)
	}
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("connect for pairing: %w", err)
	}

	out := make(chan whatsmeow.QRChannelItem, 8)
	go func() {
		defer close(out)
		for item := range qrChan {
			out <- item
			if item.Event == "success" {
				jid := client.Store.ID
				if jid == nil {
					m.logger.Error("WhatsApp pairing reported success with no device JID",
						zap.String("customerID", customerID))
					continue
				}
				if err := m.store.PutWhatsAppSession(context.Background(), customerID, jid.String()); err != nil {
					m.logger.Error("Failed to persist WhatsApp session mapping",
						zap.String("customerID", customerID), zap.Error(err))
				}
				m.registerHandlers(client, customerID)
				m.mu.Lock()
				m.clients[customerID] = client
				m.mu.Unlock()
				m.logger.Info("WhatsApp pairing succeeded",
					zap.String("customerID", customerID), zap.String("jid", jid.String()))
			} else if item.Event != "code" {
				m.logger.Warn("WhatsApp pairing ended without success",
					zap.String("customerID", customerID), zap.String("event", item.Event))
			}
		}
	}()
	return out, nil
}

func (m *WhatsAppManager) registerHandlers(client *whatsmeow.Client, customerID string) {
	client.AddEventHandler(func(rawEvt any) {
		switch evt := rawEvt.(type) {
		case *events.Message:
			m.handleMessage(customerID, evt)
		case *events.LoggedOut:
			m.logger.Warn("WhatsApp session logged out remotely", zap.String("customerID", customerID))
			m.mu.Lock()
			delete(m.clients, customerID)
			m.mu.Unlock()
		}
	})
}

func (m *WhatsAppManager) handleMessage(customerID string, evt *events.Message) {
	if m.onInbound == nil || evt.Message == nil {
		return
	}
	text := extractText(evt.Message)
	if strings.TrimSpace(text) == "" {
		return // media-only/reaction/receipt messages: nothing for the Expert Advisor to act on yet
	}
	m.onInbound(context.Background(), InboundMessage{
		CustomerID: customerID,
		Channel:    "whatsapp",
		ChatID:     evt.Info.Chat.String(),
		SenderID:   evt.Info.Sender.String(),
		SenderName: evt.Info.PushName,
		Text:       text,
		Timestamp:  evt.Info.Timestamp,
	})
}

func extractText(msg *waE2E.Message) string {
	if msg.GetConversation() != "" {
		return msg.GetConversation()
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}

// SendText delivers an outbound response from the Expert-Advisor/Supervisor
// pipeline back out over the customer's WhatsApp session (relay.go's
// outbound direction).
func (m *WhatsAppManager) SendText(ctx context.Context, customerID, chatJID, text string) error {
	m.mu.Lock()
	client, ok := m.clients[customerID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no active WhatsApp session for customer %s", customerID)
	}
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("parse chat JID: %w", err)
	}
	_, err = client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &text,
	})
	return err
}

// Sessions reports every currently connected customer session, for the
// bridge's own status endpoint.
func (m *WhatsAppManager) Sessions() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]bool, len(m.clients))
	for customerID, client := range m.clients {
		result[customerID] = client.IsLoggedIn()
	}
	return result
}

// Close disconnects every active WhatsApp client. Session state itself
// stays in the shared database for the next RestoreSessions.
func (m *WhatsAppManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		client.Disconnect()
	}
	m.clients = make(map[string]*whatsmeow.Client)
}
