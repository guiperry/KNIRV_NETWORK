package bridge

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
)

// botLocalpart is the Matrix localpart of the bridge's own bot user
// (@knirvbridge:<domain>), the identity that creates and audits bridged
// rooms. Individual WhatsApp contacts get their own ghost-puppet users under
// the reserved namespace below, not this account.
const botLocalpart = "knirvbridge"

// ghostNamespacePrefix reserves the whole @whatsapp_* user-ID namespace for
// this appservice, per Phase 4's "the appservice model is explicitly
// namespace-based" design note — one bridge instance, one reserved prefix,
// many bridged customer accounts underneath it.
const ghostNamespacePrefix = "whatsapp_"

// AppserviceBridge wraps a mautrix-go AppService: the Matrix side of the
// bridge, talking to the local, non-federating Tuwunel homeserver over a
// Unix socket in both directions (Host for inbound transactions from
// Tuwunel, HomeserverURL for the bot/ghost clients' outbound calls to it).
type AppserviceBridge struct {
	cfg    Config
	logger *zap.Logger
	store  *bridgeStore

	as *appservice.AppService
}

// loadOrCreateRegistration returns the appservice's Matrix registration,
// generating and persisting a new one (random as_token/hs_token, per
// appservice.CreateRegistration) the first time the bridge ever starts, and
// reusing it on every subsequent start so Tuwunel's own copy (loaded from
// the same file via its appservice_dir config key) never falls out of sync.
func loadOrCreateRegistration(cfg Config) (*appservice.Registration, error) {
	if _, err := os.Stat(cfg.registrationPath()); err == nil {
		return appservice.LoadRegistration(cfg.registrationPath())
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat registration file: %w", err)
	}

	reg := appservice.CreateRegistration()
	reg.ID = "knirv-bridge"
	reg.URL = "unix://" + cfg.appserviceSocketPath()
	reg.SenderLocalpart = botLocalpart
	reg.EphemeralEvents = true
	ghostRegex, err := regexp.Compile(regexp.QuoteMeta("@"+ghostNamespacePrefix) + `.*:` + regexp.QuoteMeta(cfg.HomeserverDomain))
	if err != nil {
		return nil, fmt.Errorf("compile ghost namespace regex: %w", err)
	}
	reg.Namespaces.UserIDs.Register(ghostRegex, true)

	if err := os.MkdirAll(cfg.appservicesDir(), 0750); err != nil {
		return nil, fmt.Errorf("create appservices dir: %w", err)
	}
	if err := reg.Save(cfg.registrationPath()); err != nil {
		return nil, fmt.Errorf("save registration: %w", err)
	}
	return reg, nil
}

// NewAppserviceBridge constructs the appservice but does not yet listen or
// dial anything — Start does that, once the homeserver has had a chance to
// come up.
func NewAppserviceBridge(cfg Config, logger *zap.Logger, store *bridgeStore) (*AppserviceBridge, error) {
	reg, err := loadOrCreateRegistration(cfg)
	if err != nil {
		return nil, fmt.Errorf("load or create appservice registration: %w", err)
	}

	as, err := appservice.CreateFull(appservice.CreateOpts{
		Registration:     reg,
		HomeserverDomain: cfg.HomeserverDomain,
		HomeserverURL:    "unix://" + cfg.homeserverClientSocketPath(),
		HostConfig: appservice.HostConfig{
			Hostname: cfg.appserviceSocketPath(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create appservice: %w", err)
	}
	as.Log = zapAsZerolog(logger)

	return &AppserviceBridge{cfg: cfg, logger: logger, store: store, as: as}, nil
}

// Start launches the appservice's own HTTP listener (for transactions
// Tuwunel pushes in) and the loop that consumes events Tuwunel forwards.
func (b *AppserviceBridge) Start(ctx context.Context) {
	go b.as.Start()
	go b.consumeEvents(ctx)
}

func (b *AppserviceBridge) Stop() {
	b.as.Stop()
}

// consumeEvents drains events the homeserver pushes to the appservice.
// Today this is diagnostic-only: the bridge's primary inbound direction is
// WhatsApp -> relay (see whatsapp.go/relay.go), not a human typing directly
// into the Matrix room. A future channel that *is* Matrix-native, or a human
// operator replying from a Matrix client attached to a bridged room, would
// extend this switch.
func (b *AppserviceBridge) consumeEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-b.as.Events:
			if !ok {
				return
			}
			if evt.Sender == b.as.BotIntent().UserID || strings.HasPrefix(string(evt.Sender), "@"+ghostNamespacePrefix) {
				continue // ignore the bridge's own echoes
			}
			b.logger.Debug("Appservice received Matrix event",
				zap.String("type", evt.Type.String()), zap.String("room", evt.RoomID.String()))
		}
	}
}

// ghostUserID deterministically derives a stable ghost-puppet Matrix user ID
// for one WhatsApp sender, scoped by customer so two customers' contacts can
// never collide even if they happen to message the same WhatsApp number.
func (b *AppserviceBridge) ghostUserID(customerID, senderJID string) id.UserID {
	localpart := fmt.Sprintf("%s%s_%s", ghostNamespacePrefix, sanitizeLocalpart(customerID), sanitizeLocalpart(senderJID))
	return id.NewUserID(localpart, b.cfg.HomeserverDomain)
}

func sanitizeLocalpart(s string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	return out.String()
}

// EnsureRoom returns the Matrix room mirroring one bridged chat, creating it
// (and the ghost puppet representing the external sender) the first time
// this chat is seen. The bot user always owns/creates the room, per Phase
// 4's "KNIRVGATEWAY... acts as the Matrix client/bot in each bridged room."
func (b *AppserviceBridge) EnsureRoom(ctx context.Context, msg InboundMessage) (id.RoomID, *appservice.IntentAPI, error) {
	conv, ok, err := b.store.Get(ctx, msg.Channel, msg.CustomerID, msg.ChatID)
	if err != nil {
		return "", nil, err
	}

	ghostID := b.ghostUserID(msg.CustomerID, msg.SenderID)
	ghost := b.as.Intent(ghostID)
	if err := ghost.EnsureRegistered(ctx); err != nil {
		return "", nil, fmt.Errorf("register ghost puppet: %w", err)
	}
	if displayName := strings.TrimSpace(msg.SenderName); displayName != "" {
		_ = ghost.SetDisplayName(ctx, displayName)
	}

	if ok {
		if err := ghost.EnsureJoined(ctx, id.RoomID(conv.RoomID)); err != nil {
			b.logger.Warn("Ghost puppet failed to (re)join existing bridged room",
				zap.String("room", conv.RoomID), zap.Error(err))
		}
		return id.RoomID(conv.RoomID), ghost, nil
	}

	resp, err := b.as.BotClient().CreateRoom(ctx, &mautrix.ReqCreateRoom{
		Name:   fmt.Sprintf("WhatsApp: %s", firstNonEmpty(msg.SenderName, msg.SenderID)),
		Topic:  fmt.Sprintf("Bridged WhatsApp chat %s for KNIRV customer %s", msg.ChatID, msg.CustomerID),
		Invite: []id.UserID{ghostID},
		CreationContent: map[string]interface{}{
			"m.federate": false,
		},
	})
	if err != nil {
		return "", nil, fmt.Errorf("create bridged room: %w", err)
	}
	if err := ghost.EnsureJoined(ctx, resp.RoomID); err != nil {
		b.logger.Warn("Ghost puppet failed to join newly created bridged room",
			zap.String("room", resp.RoomID.String()), zap.Error(err))
	}

	if err := b.store.Put(ctx, &Conversation{
		CustomerID: msg.CustomerID,
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
		RoomID:     resp.RoomID.String(),
		CreatedAt:  time.Now(),
	}); err != nil {
		return "", nil, fmt.Errorf("persist conversation mapping: %w", err)
	}

	return resp.RoomID, ghost, nil
}

// PostFromGhost mirrors an inbound WhatsApp message into its bridged Matrix
// room as the ghost puppet representing the original sender.
func (b *AppserviceBridge) PostFromGhost(ctx context.Context, ghost *appservice.IntentAPI, roomID id.RoomID, text string) error {
	_, err := ghost.SendText(ctx, roomID, text)
	return err
}

// PostFromBot mirrors an outbound Expert-Advisor/Supervisor response into a
// bridged room as the bridge's own bot user.
func (b *AppserviceBridge) PostFromBot(ctx context.Context, roomID id.RoomID, text string) error {
	_, err := b.as.BotIntent().SendText(ctx, roomID, text)
	return err
}

// RoomForConversation looks up a room the caller already knows the
// conversation for (relay.go's outbound path, which starts from a
// customer+chat pair rather than a live inbound event).
func (b *AppserviceBridge) RoomForConversation(ctx context.Context, channel, customerID, chatID string) (id.RoomID, bool, error) {
	conv, ok, err := b.store.Get(ctx, channel, customerID, chatID)
	if err != nil || !ok {
		return "", ok, err
	}
	return id.RoomID(conv.RoomID), true, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "unknown"
}
