package ibc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type MessageHandler func(context.Context, IBCMessage) error

// Handler is the central IBC coordinator for the KNIRV network
type Handler struct {
	channels        map[string]*IBCChannel
	connections     map[string]*IBCConnection
	clients         map[string]*IBCClient
	messageQueue    chan IBCMessage
	packetQueue     chan *PendingPacket
	chainChannels   map[types.ChainID]string // Maps chain ID to channel ID
	sequenceNumber  uint64
	logger          *zap.Logger
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	httpClient      *http.Client
	messageHandlers map[MessageType]MessageHandler
	startOnce       sync.Once
	stopOnce        sync.Once
	wg              sync.WaitGroup
}

// NewHandler creates a new IBC handler
func NewHandler(logger *zap.Logger) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Handler{
		channels:        make(map[string]*IBCChannel),
		connections:     make(map[string]*IBCConnection),
		clients:         make(map[string]*IBCClient),
		messageQueue:    make(chan IBCMessage, 1000),
		packetQueue:     make(chan *PendingPacket, 1000),
		chainChannels:   make(map[types.ChainID]string),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
		messageHandlers: make(map[MessageType]MessageHandler),
	}
}

// Start starts the IBC handler
func (h *Handler) Start() error {
	h.startOnce.Do(func() {
		h.wg.Add(2)
		go func() { defer h.wg.Done(); h.processMessages() }()
		go func() { defer h.wg.Done(); h.processPackets() }()
	})

	h.logger.Info("IBC handler started")
	return nil
}

// Stop stops the IBC handler
func (h *Handler) Stop() error {
	h.stopOnce.Do(func() {
		h.cancel()
		h.wg.Wait()
	})

	h.logger.Info("IBC handler stopped")
	return nil
}

func (h *Handler) RegisterMessageHandler(messageType MessageType, handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("message handler is required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.messageHandlers[messageType]; exists {
		return fmt.Errorf("handler for %s already registered", messageType.String())
	}
	h.messageHandlers[messageType] = handler
	return nil
}

// RegisterChannel registers a new IBC channel
func (h *Handler) RegisterChannel(channelID, connectionID, portID string, chainID types.ChainID, ordering ChannelOrdering) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.channels[channelID]; exists {
		return fmt.Errorf("channel %s already exists", channelID)
	}

	channel := &IBCChannel{
		ChannelID:    channelID,
		ConnectionID: connectionID,
		PortID:       portID,
		State:        ChannelStateInit,
		Version:      "ics20-1",
		Ordering:     ordering,
		ChainID:      chainID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.channels[channelID] = channel
	h.chainChannels[chainID] = channelID

	h.logger.Info("Registered IBC channel",
		zap.String("channel_id", channelID),
		zap.String("chain", chainID.String()),
	)

	return nil
}

// RegisterConnection registers a new IBC connection
func (h *Handler) RegisterConnection(connID, clientID, endpoint string, chainID types.ChainID, transport TransportType) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.connections[connID]; exists {
		return fmt.Errorf("connection %s already exists", connID)
	}

	connection := &IBCConnection{
		ConnectionID: connID,
		ClientID:     clientID,
		State:        ConnectionStateInit,
		DelayPeriod:  0,
		ChainID:      chainID,
		Endpoint:     endpoint,
		Transport:    transport,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.connections[connID] = connection

	h.logger.Info("Registered IBC connection",
		zap.String("connection_id", connID),
		zap.String("endpoint", endpoint),
		zap.String("chain", chainID.String()),
	)

	return nil
}

// RegisterClient registers a new light client
func (h *Handler) RegisterClient(clientID, clientType string, chainID types.ChainID, trustingPeriod, unbondingPeriod uint64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[clientID]; exists {
		return fmt.Errorf("client %s already exists", clientID)
	}

	client := &IBCClient{
		ClientID:        clientID,
		ClientType:      clientType,
		ChainID:         chainID,
		State:           ClientStateActive,
		LatestHeight:    0,
		TrustingPeriod:  trustingPeriod,
		UnbondingPeriod: unbondingPeriod,
		MaxClockDrift:   10, // 10 seconds
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	h.clients[clientID] = client

	h.logger.Info("Registered light client",
		zap.String("client_id", clientID),
		zap.String("chain", chainID.String()),
	)

	return nil
}

// OpenChannel opens a channel (transitions to Open state)
func (h *Handler) OpenChannel(channelID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel, exists := h.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	channel.State = ChannelStateOpen
	channel.UpdatedAt = time.Now()

	h.logger.Info("Opened IBC channel", zap.String("channel_id", channelID))

	return nil
}

// OpenConnection opens a connection (transitions to Open state)
func (h *Handler) OpenConnection(connectionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	connection, exists := h.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	connection.State = ConnectionStateOpen
	connection.UpdatedAt = time.Now()

	h.logger.Info("Opened IBC connection", zap.String("connection_id", connectionID))

	return nil
}

// GetChannel retrieves a channel by ID
func (h *Handler) GetChannel(channelID string) (*IBCChannel, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channel, exists := h.channels[channelID]
	if !exists {
		return nil, types.ErrChannelNotFound
	}

	return channel, nil
}

// GetChannelForChain retrieves the channel for a specific chain
func (h *Handler) GetChannelForChain(chainID types.ChainID) (*IBCChannel, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channelID, exists := h.chainChannels[chainID]
	if !exists {
		return nil, fmt.Errorf("no channel found for chain %s", chainID.String())
	}

	return h.channels[channelID], nil
}

// GetConnection retrieves a connection by ID
func (h *Handler) GetConnection(connectionID string) (*IBCConnection, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connection, exists := h.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	return connection, nil
}

// GetClient retrieves a client by ID
func (h *Handler) GetClient(clientID string) (*IBCClient, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}

	return client, nil
}

// ListChannels returns all registered channels
func (h *Handler) ListChannels() []*IBCChannel {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channels := make([]*IBCChannel, 0, len(h.channels))
	for _, channel := range h.channels {
		channels = append(channels, channel)
	}

	return channels
}

// ListConnections returns all registered connections
func (h *Handler) ListConnections() []*IBCConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]*IBCConnection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}

	return connections
}

// ListClients returns all registered clients
func (h *Handler) ListClients() []*IBCClient {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*IBCClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}

// SendMessage sends an IBC message to the message queue
func (h *Handler) SendMessage(msg IBCMessage) error {
	select {
	case h.messageQueue <- msg:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("handler stopped")
	default:
		return fmt.Errorf("message queue full")
	}
}

// SendPacket sends an IBC packet
func (h *Handler) SendPacket(packet *IBCPacket) error {
	h.mu.Lock()
	h.sequenceNumber++
	packet.Sequence = h.sequenceNumber
	h.mu.Unlock()

	pendingPacket := &PendingPacket{
		Packet:     packet,
		Retries:    0,
		MaxRetries: 3,
		LastRetry:  time.Now(),
	}

	select {
	case h.packetQueue <- pendingPacket:
		h.logger.Info("Sent IBC packet",
			zap.Uint64("sequence", packet.Sequence),
			zap.String("dest_chain", packet.DestChainID.String()),
		)
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("handler stopped")
	default:
		return fmt.Errorf("packet queue full")
	}
}

// processMessages processes messages from the message queue
func (h *Handler) processMessages() {
	for {
		select {
		case msg := <-h.messageQueue:
			h.handleMessage(msg)
		case <-h.ctx.Done():
			return
		}
	}
}

// processPackets processes packets from the packet queue
func (h *Handler) processPackets() {
	for {
		select {
		case pending := <-h.packetQueue:
			h.handlePacket(pending)
		case <-h.ctx.Done():
			return
		}
	}
}

// handleMessage handles an IBC message
func (h *Handler) handleMessage(msg IBCMessage) {
	h.logger.Debug("Processing IBC message",
		zap.String("type", msg.Type.String()),
		zap.String("sender", msg.Sender),
	)

	h.mu.RLock()
	handler := h.messageHandlers[msg.Type]
	h.mu.RUnlock()
	if handler == nil {
		h.logger.Error("No IBC message handler registered", zap.String("type", msg.Type.String()))
		return
	}
	if err := handler(h.ctx, msg); err != nil {
		h.logger.Error("IBC message handler failed", zap.String("type", msg.Type.String()), zap.Error(err))
	}
}

// handlePacket handles an IBC packet
func (h *Handler) handlePacket(pending *PendingPacket) {
	packet := pending.Packet

	h.logger.Debug("Processing IBC packet",
		zap.Uint64("sequence", packet.Sequence),
		zap.String("dest_chain", packet.DestChainID.String()),
	)

	// Get channel for destination chain
	channel, err := h.GetChannelForChain(packet.DestChainID)
	if err != nil {
		h.logger.Error("Channel not found for destination chain",
			zap.String("chain", packet.DestChainID.String()),
			zap.Error(err),
		)
		h.retryPacket(pending)
		return
	}

	// Get connection
	connection, err := h.GetConnection(channel.ConnectionID)
	if err != nil {
		h.logger.Error("Connection not found",
			zap.String("connection_id", channel.ConnectionID),
			zap.Error(err),
		)
		h.retryPacket(pending)
		return
	}

	// Transmit packet based on transport type
	err = h.transmitPacket(connection, packet)
	if err != nil {
		h.logger.Error("Failed to transmit packet",
			zap.Uint64("sequence", packet.Sequence),
			zap.Error(err),
		)
		h.retryPacket(pending)
		return
	}

	h.logger.Info("Successfully transmitted packet",
		zap.Uint64("sequence", packet.Sequence),
		zap.String("dest_chain", packet.DestChainID.String()),
	)
}

// transmitPacket transmits a packet to a destination chain
func (h *Handler) transmitPacket(connection *IBCConnection, packet *IBCPacket) error {
	if connection.State != ConnectionStateOpen {
		return fmt.Errorf("connection %s is not open", connection.ConnectionID)
	}
	if packet == nil {
		return fmt.Errorf("packet is required")
	}
	// Serialize packet
	packetBytes, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	switch connection.Transport {
	case TransportGRPC:
		return h.transmitGRPC(connection.Endpoint, packetBytes)
	case TransportHTTP:
		return h.transmitHTTP(connection.Endpoint, packetBytes)
	case TransportWebSocket:
		return h.transmitWebSocket(connection.Endpoint, packetBytes)
	default:
		return fmt.Errorf("unsupported transport type: %v", connection.Transport)
	}
}

// retryPacket retries a failed packet transmission
func (h *Handler) retryPacket(pending *PendingPacket) {
	pending.Retries++
	pending.LastRetry = time.Now()

	if pending.Retries >= pending.MaxRetries {
		h.logger.Error("Max retries reached for packet",
			zap.Uint64("sequence", pending.Packet.Sequence),
		)
		return
	}

	// Retry after delay
	go func() {
		timer := time.NewTimer(time.Second * time.Duration(pending.Retries))
		defer timer.Stop()
		select {
		case <-timer.C:
			select {
			case h.packetQueue <- pending:
			case <-h.ctx.Done():
			}
		case <-h.ctx.Done():
		}
	}()
}

func (h *Handler) transmitGRPC(endpoint string, data []byte) error {
	h.logger.Debug("Transmitting via gRPC", zap.String("endpoint", endpoint))
	target := strings.TrimPrefix(strings.TrimPrefix(endpoint, "grpc://"), "grpcs://")
	if target == "" {
		return fmt.Errorf("gRPC endpoint is required")
	}
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
	defer cancel()
	connection, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create gRPC client: %w", err)
	}
	defer connection.Close()
	method := strings.TrimSpace(os.Getenv("KNIRV_IBC_GRPC_METHOD"))
	if method == "" {
		method = "/knirv.oracle.ibc.v1.PacketService/RelayPacket"
	}
	if err := connection.Invoke(ctx, method, wrapperspb.Bytes(data), &emptypb.Empty{}); err != nil {
		return fmt.Errorf("invoke %s: %w", method, err)
	}
	return nil
}

func (h *Handler) transmitHTTP(endpoint string, data []byte) error {
	h.logger.Debug("Transmitting via HTTP", zap.String("endpoint", endpoint))
	request, err := http.NewRequestWithContext(h.ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create HTTP packet request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send HTTP packet: %w", err)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		return fmt.Errorf("read HTTP acknowledgement: %w", readErr)
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP relay returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (h *Handler) transmitWebSocket(endpoint string, data []byte) error {
	h.logger.Debug("Transmitting via WebSocket", zap.String("endpoint", endpoint))
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
	defer cancel()
	connection, response, err := websocket.DefaultDialer.DialContext(ctx, endpoint, http.Header{"Origin": []string{"https://gateway.knirv.network"}})
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return fmt.Errorf("dial WebSocket relay: %w", err)
	}
	defer connection.Close()
	if err := connection.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write WebSocket packet: %w", err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return fmt.Errorf("set WebSocket acknowledgement deadline: %w", err)
	}
	_, acknowledgement, err := connection.ReadMessage()
	if err != nil {
		return fmt.Errorf("read WebSocket acknowledgement: %w", err)
	}
	var ack PacketAcknowledgement
	if err := json.Unmarshal(acknowledgement, &ack); err != nil {
		return fmt.Errorf("decode WebSocket acknowledgement: %w", err)
	}
	if !ack.Success {
		return fmt.Errorf("WebSocket relay rejected packet: %s", ack.Error)
	}
	return nil
}
