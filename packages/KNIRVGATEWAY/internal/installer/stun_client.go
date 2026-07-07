package installer

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

type STUNClient struct {
	servers []string
	logger  *zap.Logger
}

type STUNResult struct {
	ExternalIP   string `json:"external_ip"`
	ExternalPort int    `json:"external_port"`
	Server      string `json:"server"`
	Timestamp   int64  `json:"timestamp"`
}

func NewSTUNClient(servers []string, logger *zap.Logger) *STUNClient {
	return &STUNClient{
		servers: servers,
		logger:  logger,
	}
}

func (c *STUNClient) DiscoverExternalAddress() (*STUNResult, error) {
	if len(c.servers) == 0 {
		c.servers = []string{"stun.l.google.com:19302"}
	}

	var lastErr error
	for _, server := range c.servers {
		result, err := c.queryServer(server)
		if err != nil {
			c.logger.Warn("STUN server query failed",
				zap.String("server", server),
				zap.Error(err))
			lastErr = err
			continue
		}
		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all STUN servers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no STUN servers configured")
}

func (c *STUNClient) queryServer(server string) (*STUNResult, error) {
	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer conn.Close()

	stunMsg := []byte{0x00, 0x01, 0x00, 0x00}
	_, err = conn.Write(stunMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read STUN response: %w", err)
	}

	var response struct {
		Type          string `json:"type"`
		RemoteAddress string `json:"remoteAddress"`
		RemotePort    int    `json:"remotePort"`
		Timestamp    int64  `json:"timestamp"`
	}

	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		c.logger.Debug("Failed to parse STUN response, using raw address",
			zap.Error(err))
		return &STUNResult{
			ExternalIP:   remoteAddr.IP.String(),
			ExternalPort: remoteAddr.Port,
			Server:      server,
			Timestamp:   time.Now().Unix(),
		}, nil
	}

	return &STUNResult{
		ExternalIP:   response.RemoteAddress,
		ExternalPort: response.RemotePort,
		Server:      server,
		Timestamp:   response.Timestamp,
	}, nil
}

func (c *STUNClient) DiscoverWithLocalPort(localPort int) (*STUNResult, error) {
	if len(c.servers) == 0 {
		c.servers = []string{"stun.l.google.com:19302"}
	}

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on local port: %w", err)
	}
	defer conn.Close()

	var lastErr error
	for _, server := range c.servers {
		result, err := c.queryServerWithConn(conn, server)
		if err != nil {
			c.logger.Warn("STUN server query failed",
				zap.String("server", server),
				zap.Error(err))
			lastErr = err
			continue
		}
		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all STUN servers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no STUN servers configured")
}

func (c *STUNClient) queryServerWithConn(conn *net.UDPConn, server string) (*STUNResult, error) {
	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %w", err)
	}

	stunMsg := []byte{0x00, 0x01, 0x00, 0x00}
	_, err = conn.WriteToUDP(stunMsg, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read STUN response: %w", err)
	}

	var response struct {
		Type          string `json:"type"`
		RemoteAddress string `json:"remoteAddress"`
		RemotePort    int    `json:"remotePort"`
		Timestamp    int64  `json:"timestamp"`
	}

	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		c.logger.Debug("Failed to parse STUN response, using raw address",
			zap.Error(err))
		return &STUNResult{
			ExternalIP:   remoteAddr.IP.String(),
			ExternalPort: remoteAddr.Port,
			Server:      server,
			Timestamp:   time.Now().Unix(),
		}, nil
	}

	return &STUNResult{
		ExternalIP:   response.RemoteAddress,
		ExternalPort: response.RemotePort,
		Server:      server,
		Timestamp:   response.Timestamp,
	}, nil
}