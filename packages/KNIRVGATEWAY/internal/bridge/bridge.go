package bridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers as "sqlite" — matches this repo's CGO_ENABLED=0 Docker builds
)

// Service is the bridge's top-level entry point: it wires the Tuwunel
// homeserver subprocess, the Matrix appservice, the WhatsApp multi-tenant
// manager, and the backend relay together, and is what internal/server
// mounts routes on and drives the lifecycle of. Constructing a Service never
// starts a subprocess or dials anything — Start does that, deliberately
// deferred until after everything else in KNIRVGATEWAY's own startup
// sequence has already come up (see cmd/gateway/main.go).
type Service struct {
	cfg    Config
	logger *zap.Logger

	db         *sql.DB
	store      *bridgeStore
	homeserver *Homeserver
	appsvc     *AppserviceBridge
	whatsapp   *WhatsAppManager
	relay      *BackendRelay
}

// NewService builds every in-process object the bridge needs. When
// cfg.Enabled is false it returns a Service whose Handler reports itself
// disabled and whose Start/Stop are no-ops — safe to always construct and
// wire into the router regardless of configuration.
func NewService(cfg Config, logger *zap.Logger) (*Service, error) {
	if !cfg.Enabled {
		return &Service{cfg: cfg, logger: logger}, nil
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("bridge config: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		return nil, fmt.Errorf("create bridge data dir: %w", err)
	}

	db, err := sql.Open("sqlite", "file:"+cfg.databasePath())
	if err != nil {
		return nil, fmt.Errorf("open bridge database: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	store, err := newBridgeStore(context.Background(), db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	appsvc, err := NewAppserviceBridge(cfg, logger, store)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("construct appservice: %w", err)
	}

	svc := &Service{
		cfg:        cfg,
		logger:     logger,
		db:         db,
		store:      store,
		homeserver: NewHomeserver(cfg, logger),
		appsvc:     appsvc,
		relay:      NewBackendRelay(cfg.BackendSocketPath, cfg.InternalAuthToken),
	}
	svc.whatsapp = NewWhatsAppManager(db, store, logger, svc.handleInbound)
	return svc, nil
}

// Enabled reports whether the bridge was turned on (BRIDGE_ENABLED).
func (s *Service) Enabled() bool { return s.cfg.Enabled }

// Handler returns the HTTP handler for /api/bridge/*, safe to register on
// the gateway's router regardless of whether the bridge is enabled.
func (s *Service) Handler() http.Handler {
	return newBridgeHandlers(s)
}

// Start brings the bridge fully online: the homeserver subprocess, the
// appservice's own HTTP listener, and every previously-paired WhatsApp
// session. Every step here is non-fatal to the caller — an unvendored
// Tuwunel binary or an unreachable WhatsApp session degrades the bridge to
// inert rather than failing KNIRVGATEWAY's own startup, matching this
// codebase's posture for every other optional subsystem (TURN, oracle,
// nginx).
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.logger.Info("Messaging bridge disabled (BRIDGE_ENABLED=false) — skipping")
		return nil
	}

	if err := s.whatsapp.Init(ctx); err != nil {
		return fmt.Errorf("init whatsapp store: %w", err)
	}

	if err := s.homeserver.Start(ctx); err != nil {
		if errors.Is(err, ErrHomeserverNotVendored) {
			s.logger.Warn("Tuwunel homeserver not started: no real binary vendored — "+
				"the bridge will accept WhatsApp pairing and relay to the backend, but "+
				"cannot mirror conversations into Matrix until a real binary is vendored "+
				"(see internal/bridge/bin/README.md)", zap.Error(err))
		} else {
			s.logger.Warn("Tuwunel homeserver failed to start — continuing without it", zap.Error(err))
		}
	} else if err := s.homeserver.WaitReady(ctx, 15*time.Second); err != nil {
		s.logger.Warn("Tuwunel did not report ready in time", zap.Error(err))
	}

	// The appservice's own HTTP listener is independent of whether Tuwunel
	// itself came up — it can sit idle waiting for a homeserver that starts
	// later (e.g. after an operator vendors the real binary and restarts).
	s.appsvc.Start(ctx)

	if err := s.whatsapp.RestoreSessions(ctx); err != nil {
		s.logger.Warn("Failed to restore WhatsApp sessions", zap.Error(err))
	}

	s.logger.Info("Messaging bridge started",
		zap.Bool("homeserverRunning", s.homeserver.IsRunning()),
		zap.Int("restoredSessions", len(s.whatsapp.Sessions())))
	return nil
}

// Stop tears down every subprocess and connection the bridge owns.
func (s *Service) Stop(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}
	s.whatsapp.Close()
	s.appsvc.Stop()
	if err := s.homeserver.Stop(ctx); err != nil {
		s.logger.Warn("Error stopping tuwunel", zap.Error(err))
	}
	return s.db.Close()
}

// handleInbound is WhatsAppManager's onInbound callback: it mirrors the
// message into the bridged Matrix room (audit trail) and relays it into
// KNIRVSERVER's Expert-Advisor -> Supervisor pipeline (the actual response
// path), independently and non-fatally — a failure in one never blocks the
// other.
func (s *Service) handleInbound(ctx context.Context, msg InboundMessage) {
	roomID, ghost, err := s.appsvc.EnsureRoom(ctx, msg)
	if err != nil {
		s.logger.Warn("Failed to ensure bridged Matrix room — continuing without Matrix mirroring",
			zap.String("customerID", msg.CustomerID), zap.Error(err))
	} else if err := s.appsvc.PostFromGhost(ctx, ghost, roomID, msg.Text); err != nil {
		s.logger.Warn("Failed to mirror inbound message into Matrix room", zap.Error(err))
	}

	if ack, err := s.relay.PostInbound(ctx, msg); err != nil {
		s.logger.Warn("Failed to relay inbound bridged message to backend",
			zap.String("customerID", msg.CustomerID), zap.Error(err))
	} else if !ack.SupervisorConnected {
		s.logger.Debug("Backend recorded bridged message — no live Supervisor session for this customer yet",
			zap.String("customerID", msg.CustomerID))
	}
}

// deliverOutbound is the outbound half: a response from the
// Expert-Advisor/Supervisor pipeline (handlers.go's relay/outbound
// endpoint), sent back out over WhatsApp and mirrored into the Matrix room
// as the bot user.
func (s *Service) deliverOutbound(ctx context.Context, channel, customerID, chatID, text string) error {
	if channel != "whatsapp" {
		return fmt.Errorf("unsupported bridge channel %q", channel)
	}

	sendErr := s.whatsapp.SendText(ctx, customerID, chatID, text)

	if roomID, ok, err := s.appsvc.RoomForConversation(ctx, channel, customerID, chatID); err != nil {
		s.logger.Warn("Failed to look up bridged room for outbound message", zap.Error(err))
	} else if ok {
		if err := s.appsvc.PostFromBot(ctx, roomID, text); err != nil {
			s.logger.Warn("Failed to mirror outbound message into Matrix room", zap.Error(err))
		}
	}

	return sendErr
}

// Status is a snapshot for the bridge's own status endpoint.
type Status struct {
	Enabled            bool            `json:"enabled"`
	HomeserverVendored bool            `json:"homeserver_vendored"`
	HomeserverRunning  bool            `json:"homeserver_running"`
	WhatsAppSessions   map[string]bool `json:"whatsapp_sessions"`
	Conversations      int             `json:"conversations"`
}

func (s *Service) status(ctx context.Context) Status {
	st := Status{Enabled: s.cfg.Enabled}
	if !s.cfg.Enabled {
		return st
	}
	st.HomeserverRunning = s.homeserver.IsRunning()
	st.HomeserverVendored = s.homeserver.IsVendored()
	st.WhatsAppSessions = s.whatsapp.Sessions()
	if convs, err := s.store.List(ctx); err == nil {
		st.Conversations = len(convs)
	}
	return st
}
