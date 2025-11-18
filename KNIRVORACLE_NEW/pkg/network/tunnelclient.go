package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config"
)

// TunnelClientImpl manages the connection to a central relay server for NAT traversal
type TunnelClientImpl struct {
	config          *config.TunnelClientConfig
	nodeID          string
	chainID         string
	internalIP      string
	internalP2PPort uint
	nodeType        string
	conn            net.Conn
	isConnected     bool
	reconnectCount  int
	mutex           sync.Mutex
	stopChan        chan struct{}
	pingTicker      *time.Ticker
}

// IdentifyMessage is sent to the tunnel server to identify this node
type IdentifyMessage struct {
	Action          string `json:"action"`
	PeerID          string `json:"devId"`
	ChainID         string `json:"chainId"`
	InternalIP      string `json:"internalIp"`
	InternalP2PPort uint   `json:"internalP2pPort"`
	Type            string `json:"type"`
}

// PingMessage is sent periodically to keep the connection alive
type PingMessage struct {
	Action string `json:"action"`
}

// ServerMessage represents messages received from the tunnel server
type ServerMessage struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Action  string `json:"action,omitempty"`
}

// NewTunnelClient creates a new tunnel client
func NewTunnelClient(cfg *config.TunnelClientConfig, nodeID, chainID, internalIP string, internalP2PPort uint, nodeType string) *TunnelClientImpl {
	return &TunnelClientImpl{
		config:          cfg,
		nodeID:          nodeID,
		chainID:         chainID,
		internalIP:      internalIP,
		internalP2PPort: internalP2PPort,
		nodeType:        nodeType,
		isConnected:     false,
		reconnectCount:  0,
		stopChan:        make(chan struct{}),
	}
}

// Connect establishes a connection to the tunnel server
func (c *TunnelClientImpl) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isConnected {
		return nil // Already connected
	}

	// Format the address properly for both IPv4 and IPv6
	var serverAddr string
	if strings.Contains(c.config.ServerAddress, ":") {
		// IPv6 address needs to be enclosed in square brackets
		serverAddr = fmt.Sprintf("[%s]:%d", c.config.ServerAddress, c.config.ControlPort)
	} else {
		// IPv4 address
		serverAddr = fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ControlPort)
	}
	log.Printf("Connecting to tunnel server at %s...", serverAddr)

	var err error
	c.conn, err = net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to tunnel server: %w", err)
	}

	// Send identification message
	identifyMsg := IdentifyMessage{
		Action:          "IDENTIFY",
		PeerID:          c.nodeID,
		ChainID:         c.chainID,
		InternalIP:      c.internalIP,
		InternalP2PPort: c.internalP2PPort,
		Type:            c.nodeType,
	}

	identifyBytes, err := json.Marshal(identifyMsg)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to marshal identify message: %w", err)
	}

	_, err = c.conn.Write(append(identifyBytes, '\n'))
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send identify message: %w", err)
	}

	// Start reading responses in a goroutine
	go c.readResponses()

	c.isConnected = true
	c.reconnectCount = 0
	log.Printf("Connected to tunnel server and sent identification")

	// Start ping ticker
	c.startPingTicker()

	return nil
}

// readResponses reads and processes messages from the tunnel server
func (c *TunnelClientImpl) readResponses() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		msg := scanner.Text()
		var serverMsg ServerMessage
		if err := json.Unmarshal([]byte(msg), &serverMsg); err != nil {
			log.Printf("Error parsing server message: %v", err)
			continue
		}

		log.Printf("Received from tunnel server: %+v", serverMsg)

		// Handle specific message types
		if serverMsg.Action == "PONG" {
			log.Printf("Received PONG from tunnel server")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from tunnel server: %v", err)
	}

	// Connection closed, attempt reconnect
	c.mutex.Lock()
	c.isConnected = false
	c.mutex.Unlock()

	c.attemptReconnect()
}

// attemptReconnect tries to reconnect to the tunnel server
func (c *TunnelClientImpl) attemptReconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Stop the ping ticker
	if c.pingTicker != nil {
		c.pingTicker.Stop()
		c.pingTicker = nil
	}

	// Check if we should stop
	select {
	case <-c.stopChan:
		return
	default:
		// Continue with reconnect
	}

	c.reconnectCount++
	reconnectDelay := time.Duration(c.config.ReconnectDelay) * time.Second

	log.Printf("Connection to tunnel server lost. Attempting reconnect %d in %v...",
		c.reconnectCount, reconnectDelay)

	// Schedule reconnect
	time.AfterFunc(reconnectDelay, func() {
		err := c.Connect()
		if err != nil {
			log.Printf("Failed to reconnect to tunnel server: %v", err)
			c.attemptReconnect() // Try again
		}
	})
}

// startPingTicker starts a ticker to send periodic pings
func (c *TunnelClientImpl) startPingTicker() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}

	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	c.pingTicker = time.NewTicker(pingInterval)

	go func() {
		for {
			select {
			case <-c.pingTicker.C:
				c.sendPing()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// sendPing sends a ping message to keep the connection alive
func (c *TunnelClientImpl) sendPing() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isConnected || c.conn == nil {
		return
	}

	pingMsg := PingMessage{
		Action: "PING",
	}

	pingBytes, err := json.Marshal(pingMsg)
	if err != nil {
		log.Printf("Failed to marshal ping message: %v", err)
		return
	}

	_, err = c.conn.Write(append(pingBytes, '\n'))
	if err != nil {
		log.Printf("Failed to send ping: %v", err)
		// Connection might be broken, close it to trigger reconnect
		c.conn.Close()
		c.isConnected = false
		return
	}

	log.Printf("Sent PING to tunnel server")
}

// Disconnect closes the connection to the tunnel server
func (c *TunnelClientImpl) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Signal to stop reconnect attempts and ping ticker
	close(c.stopChan)

	if c.pingTicker != nil {
		c.pingTicker.Stop()
		c.pingTicker = nil
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.isConnected = false
	log.Printf("Disconnected from tunnel server")
}

// IsConnected returns whether the client is currently connected
func (c *TunnelClientImpl) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.isConnected
}
