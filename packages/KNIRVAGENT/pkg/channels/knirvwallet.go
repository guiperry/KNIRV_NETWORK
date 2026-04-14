package channels

import (
	"context"
	"fmt"

	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/config"
	"github.com/knirvcorp/knirvagent/pkg/logger"
)

type KNIRVWalletChannel struct {
	*BaseChannel
	config config.KNIRVWalletConfig
}

func NewKNIRVWalletChannel(cfg config.KNIRVWalletConfig, bus *bus.MessageBus) (*KNIRVWalletChannel, error) {
	base := NewBaseChannel("knirvwallet", cfg, bus, cfg.AllowFrom)
	return &KNIRVWalletChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *KNIRVWalletChannel) Start(ctx context.Context) error {
	logger.InfoC("knirvwallet", "Starting KNIRVWallet placeholder channel")
	c.setRunning(true)
	return nil
}

func (c *KNIRVWalletChannel) Stop(ctx context.Context) error {
	logger.InfoC("knirvwallet", "Stopping KNIRVWallet placeholder channel")
	c.setRunning(false)
	return nil
}

func (c *KNIRVWalletChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("knirvwallet channel not running")
	}

	logger.InfoCF("knirvwallet", "Sending message to KNIRVWallet (placeholder)", map[string]interface{}{
		"chat_id":     msg.ChatID,
		"content_len": len(msg.Content),
	})

	// Placeholder: In a real implementation, this would send to a WebSocket or API
	return nil
}

// Receive is a placeholder for external messages coming from the wallet
func (c *KNIRVWalletChannel) Receive(senderID, chatID, content string) {
	logger.InfoCF("knirvwallet", "Received message from KNIRVWallet (placeholder)", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   chatID,
	})
	c.HandleMessage(senderID, chatID, content, nil, nil)
}
