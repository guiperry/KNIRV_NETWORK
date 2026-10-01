package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Conversation maps one bridged external chat (e.g. a WhatsApp JID) to the
// Matrix room the bridge mirrors it into, and to the KNIRV customer account
// that owns the bridged session.
type Conversation struct {
	CustomerID string    `json:"customer_id"`
	Channel    string    `json:"channel"` // "whatsapp" today; more later per Phase 4's channel list
	ChatID     string    `json:"chat_id"` // channel-native chat identifier (a whatsmeow JID string)
	RoomID     string    `json:"room_id"` // Matrix room ID mirroring this chat
	CreatedAt  time.Time `json:"created_at"`
}

// bridgeStore persists the bridge's own tables (bridge_conversations,
// bridge_whatsapp_sessions) inside the single shared SQLite database
// (Config.databasePath) — the same *sql.DB connection whatsmeow's own
// sqlstore.Container manages its session tables in. This intentionally
// avoids standing up a second, differently-shaped store (e.g. a bespoke
// JSON file) next to a database the bridge already has to create for
// whatsmeow: one local database file for everything this package persists.
type bridgeStore struct {
	db *sql.DB
}

func newBridgeStore(ctx context.Context, db *sql.DB) (*bridgeStore, error) {
	const schema = `
CREATE TABLE IF NOT EXISTS bridge_conversations (
	channel     TEXT NOT NULL,
	customer_id TEXT NOT NULL,
	chat_id     TEXT NOT NULL,
	room_id     TEXT NOT NULL UNIQUE,
	created_at  TIMESTAMP NOT NULL,
	PRIMARY KEY (channel, customer_id, chat_id)
);
CREATE TABLE IF NOT EXISTS bridge_whatsapp_sessions (
	customer_id TEXT PRIMARY KEY,
	device_jid  TEXT NOT NULL,
	created_at  TIMESTAMP NOT NULL
);`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("create bridge tables: %w", err)
	}
	return &bridgeStore{db: db}, nil
}

// PutWhatsAppSession records which WhatsApp device JID belongs to which
// KNIRV customer, so a restart can reattach the right whatsmeow client to
// the right customer without re-pairing.
func (s *bridgeStore) PutWhatsAppSession(ctx context.Context, customerID, deviceJID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bridge_whatsapp_sessions (customer_id, device_jid, created_at)
		   VALUES (?, ?, ?)
		   ON CONFLICT (customer_id) DO UPDATE SET device_jid = excluded.device_jid`,
		customerID, deviceJID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("put whatsapp session: %w", err)
	}
	return nil
}

// WhatsAppSessions returns the customer-id -> device-JID map recorded by
// PutWhatsAppSession, used to reattach restored whatsmeow devices
// (Container.GetAllDevices) to the customer that paired them.
func (s *bridgeStore) WhatsAppSessions(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT customer_id, device_jid FROM bridge_whatsapp_sessions`)
	if err != nil {
		return nil, fmt.Errorf("list whatsapp sessions: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var customerID, deviceJID string
		if err := rows.Scan(&customerID, &deviceJID); err != nil {
			return nil, fmt.Errorf("scan whatsapp session: %w", err)
		}
		result[customerID] = deviceJID
	}
	return result, rows.Err()
}

func (s *bridgeStore) Get(ctx context.Context, channel, customerID, chatID string) (*Conversation, bool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT channel, customer_id, chat_id, room_id, created_at
		   FROM bridge_conversations WHERE channel = ? AND customer_id = ? AND chat_id = ?`,
		channel, customerID, chatID)
	conv, err := scanConversation(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("get conversation: %w", err)
	}
	return conv, true, nil
}

func (s *bridgeStore) GetByRoom(ctx context.Context, roomID string) (*Conversation, bool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT channel, customer_id, chat_id, room_id, created_at
		   FROM bridge_conversations WHERE room_id = ?`,
		roomID)
	conv, err := scanConversation(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("get conversation by room: %w", err)
	}
	return conv, true, nil
}

func (s *bridgeStore) Put(ctx context.Context, row *Conversation) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bridge_conversations (channel, customer_id, chat_id, room_id, created_at)
		   VALUES (?, ?, ?, ?, ?)
		   ON CONFLICT (channel, customer_id, chat_id)
		   DO UPDATE SET room_id = excluded.room_id`,
		row.Channel, row.CustomerID, row.ChatID, row.RoomID, row.CreatedAt.UTC())
	if err != nil {
		return fmt.Errorf("put conversation: %w", err)
	}
	return nil
}

// List returns every known conversation, primarily for the bridge's own
// status endpoint.
func (s *bridgeStore) List(ctx context.Context) ([]*Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT channel, customer_id, chat_id, room_id, created_at FROM bridge_conversations`)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var result []*Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		result = append(result, conv)
	}
	return result, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConversation(row rowScanner) (*Conversation, error) {
	conv := &Conversation{}
	if err := row.Scan(&conv.Channel, &conv.CustomerID, &conv.ChatID, &conv.RoomID, &conv.CreatedAt); err != nil {
		return nil, err
	}
	return conv, nil
}
