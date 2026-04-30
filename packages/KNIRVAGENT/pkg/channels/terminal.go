package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/logger"
)

type TerminalChannel struct {
	*BaseChannel
}

func NewTerminalChannel(bus *bus.MessageBus) *TerminalChannel {
	return &TerminalChannel{
		BaseChannel: NewBaseChannel("terminal", nil, bus, nil),
	}
}

func (c *TerminalChannel) Start(ctx context.Context) error {
	c.setRunning(true)
	logger.InfoC("channels", "Terminal channel started")
	return nil
}

func (c *TerminalChannel) Stop(ctx context.Context) error {
	c.setRunning(false)
	return nil
}

func (c *TerminalChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	// Find all PTYs
	entries, err := os.ReadDir("/dev/pts")
	if err != nil {
		return fmt.Errorf("failed to read /dev/pts: %w", err)
	}

	formattedMsg := fmt.Sprintf("\r\n\x1b[32m[KNIRVAGENT] %s\x1b[0m\r\n", msg.Content)

	for _, entry := range entries {
		// Only write to numbered PTYs (e.g., /dev/pts/0)
		name := entry.Name()
		if name == "ptmx" || name == "terminal" {
			continue
		}

		path := filepath.Join("/dev/pts", name)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			// Skip PTYs we can't open
			continue
		}
		
		_, _ = f.WriteString(formattedMsg)
		_ = f.Close()
	}

	return nil
}
