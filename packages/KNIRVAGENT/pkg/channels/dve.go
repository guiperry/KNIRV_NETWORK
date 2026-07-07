package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/logger"
)

type DVEChannel struct {
	*BaseChannel
	dveID     string
	serverURL string
	client    *http.Client
}

func NewDVEChannel(bus *bus.MessageBus) (*DVEChannel, error) {
	dveID := os.Getenv("DVE_ID")
	serverURL := os.Getenv("KNIRV_SERVER_URL")

	if dveID == "" {
		return nil, fmt.Errorf("DVE_ID environment variable not set")
	}
	if serverURL == "" {
		serverURL = "http://host.docker.internal:8080" // Default fallback
	}

	return &DVEChannel{
		BaseChannel: NewBaseChannel("dve", nil, bus, nil),
		dveID:       dveID,
		serverURL:   serverURL,
		client:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *DVEChannel) Start(ctx context.Context) error {
	c.setRunning(true)
	logger.InfoCF("channels", "DVE channel started", map[string]interface{}{
		"dve_id":     c.dveID,
		"server_url": c.serverURL,
	})
	return nil
}

func (c *DVEChannel) Stop(ctx context.Context) error {
	c.setRunning(false)
	return nil
}

func (c *DVEChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	url := fmt.Sprintf("%s/api/v1/dve/%s/agent/response", c.serverURL, c.dveID)
	
	payload := map[string]string{
		"message": msg.Content,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send response to KNIRV-SERVER: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KNIRV-SERVER returned status %d", resp.StatusCode)
	}

	return nil
}
