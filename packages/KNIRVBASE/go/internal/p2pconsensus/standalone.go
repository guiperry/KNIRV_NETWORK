package p2pconsensus

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

const consensusProtocol = protocol.ID("/knirvbase/consensus/1.0.0")

// StandaloneConsensus is the direct libp2p implementation used when the
// gateway is absent. One topic is created per NetworkID and all messages carry
// the same ID so peers cannot accidentally mix networks.
type StandaloneConsensus struct {
	config  ConsensusConfig
	host    host.Host
	dht     *kaddht.IpfsDHT
	pubsub  *pubsub.PubSub
	topic   *pubsub.Topic
	sub     *pubsub.Subscription
	peers   map[string]*PeerInfo
	handler EventHandler
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	running bool
}

func NewStandaloneConsensus(cfg ConsensusConfig, handlers ...EventHandler) *StandaloneConsensus {
	var handler EventHandler
	if len(handlers) > 0 {
		handler = handlers[0]
	}
	return &StandaloneConsensus{config: cfg, peers: make(map[string]*PeerInfo), handler: handler}
}

func (s *StandaloneConsensus) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("already running")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.ctx, s.cancel = context.WithCancel(ctx)

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate peer identity: %w", err)
	}
	listen := fmt.Sprintf("/ip4/%s/tcp/%d", s.config.ListenAddr, s.config.Port)
	if s.config.ListenAddr == "" {
		listen = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", s.config.Port)
	}
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(listen),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return fmt.Errorf("create libp2p host: %w", err)
	}
	dht, err := kaddht.New(s.ctx, h, kaddht.Mode(kaddht.ModeServer), kaddht.ProtocolPrefix(protocol.ID("/knirvbase/"+s.config.NetworkID)))
	if err != nil {
		_ = h.Close()
		return fmt.Errorf("create DHT: %w", err)
	}
	ps, err := pubsub.NewGossipSub(s.ctx, h)
	if err != nil {
		_ = dht.Close()
		_ = h.Close()
		return fmt.Errorf("create GossipSub: %w", err)
	}
	topic, err := ps.Join(Topic(s.config.NetworkID, s.config.NetworkSecret))
	if err != nil {
		_ = dht.Close()
		_ = h.Close()
		return fmt.Errorf("join operations topic: %w", err)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		_ = dht.Close()
		_ = h.Close()
		return fmt.Errorf("subscribe operations topic: %w", err)
	}
	s.host, s.dht, s.pubsub, s.topic, s.sub = h, dht, ps, topic, sub
	s.running = true
	h.SetStreamHandler(consensusProtocol, s.handleStream)
	go s.receiveLoop()
	go s.discoveryLoop()
	for _, addr := range s.config.BootstrapPeers {
		if info, err := peer.AddrInfoFromString(addr); err == nil {
			go func(pi peer.AddrInfo) {
				if err := h.Connect(s.ctx, pi); err == nil {
					s.recordPeer(pi)
				}
			}(*info)
		}
	}
	log.Printf("[p2pconsensus] standalone peer %s listening on %v", h.ID(), h.Addrs())
	return nil
}

func (s *StandaloneConsensus) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	cancel, sub, topic, dht, h := s.cancel, s.sub, s.topic, s.dht, s.host
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if sub != nil {
		sub.Cancel()
	}
	if topic != nil {
		_ = topic.Close()
	}
	if dht != nil {
		_ = dht.Close()
	}
	if h != nil {
		return h.Close()
	}
	return nil
}

func (s *StandaloneConsensus) Status() ConsensusStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ConsensusStatus{Mode: "standalone", NetworkID: s.config.NetworkID, PeerCount: len(s.peers), Running: s.running}
}

func (s *StandaloneConsensus) Peers() []PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]PeerInfo, 0, len(s.peers))
	for _, p := range s.peers {
		result = append(result, *p)
	}
	return result
}

func (s *StandaloneConsensus) BroadcastOperation(ctx context.Context, op OperationEnvelope) error {
	s.mu.RLock()
	topic, running := s.topic, s.running
	s.mu.RUnlock()
	if !running || topic == nil {
		return fmt.Errorf("standalone consensus is not running")
	}
	for len(s.pubsub.ListPeers(topic.String())) == 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	if op.NetworkID == "" {
		op.NetworkID = s.config.NetworkID
	}
	if op.NetworkID != s.config.NetworkID {
		return fmt.Errorf("operation network_id %q does not match %q", op.NetworkID, s.config.NetworkID)
	}
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	return topic.Publish(ctx, data)
}

func (s *StandaloneConsensus) RequestSync(ctx context.Context, req SyncRequest) error {
	if req.NetworkID == "" {
		req.NetworkID = s.config.NetworkID
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}{Type: "sync_request", Payload: payload})
	if err != nil {
		return err
	}
	s.mu.RLock()
	h, running := s.host, s.running
	s.mu.RUnlock()
	if !running || h == nil {
		return fmt.Errorf("standalone consensus is not running")
	}
	for _, p := range h.Network().Peers() {
		stream, err := h.NewStream(ctx, p, consensusProtocol)
		if err != nil {
			continue
		}
		if _, err := stream.Write(append(data, '\n')); err == nil {
			var resp SyncResponse
			if json.NewDecoder(stream).Decode(&resp) == nil && s.handler != nil {
				if handler, ok := s.handler.(SyncResponseHandler); ok {
					_ = handler.OnSyncResponseReceived(resp)
				}
			}
		}
		_ = stream.Close()
	}
	return nil
}

func (s *StandaloneConsensus) handleStream(stream network.Stream) {
	defer stream.Close()
	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
		return
	}
	if s.handler == nil {
		return
	}
	if msg.Type == "sync_request" {
		var req SyncRequest
		if json.Unmarshal(msg.Payload, &req) == nil {
			if resp, err := s.handler.OnSyncRequestReceived(req); err == nil {
				_ = json.NewEncoder(stream).Encode(resp)
			}
		}
	}
}

func (s *StandaloneConsensus) receiveLoop() {
	for {
		s.mu.RLock()
		sub := s.sub
		ctx := s.ctx
		s.mu.RUnlock()
		if sub == nil || ctx == nil {
			return
		}
		msg, err := sub.Next(ctx)
		if err != nil {
			return
		}
		if s.host != nil && msg.GetFrom() == s.host.ID() {
			continue
		}
		var op OperationEnvelope
		if json.Unmarshal(msg.Data, &op) == nil && op.NetworkID == s.config.NetworkID {
			pi := PeerInfo{ID: msg.GetFrom().String(), LastSeen: time.Now()}
			s.mu.Lock()
			s.peers[pi.ID] = &pi
			s.mu.Unlock()
			if s.handler != nil {
				_ = s.handler.OnOperationReceived(op)
			}
			continue
		}
		var wire WireMessage
		if json.Unmarshal(msg.Data, &wire) != nil || wire.Type != MsgSyncRequest || wire.NetworkID != s.config.NetworkID || s.handler == nil {
			continue
		}
		var req SyncRequest
		if json.Unmarshal(wire.Payload, &req) != nil {
			continue
		}
		resp, err := s.handler.OnSyncRequestReceived(req)
		if err == nil && resp != nil {
			for _, responseOp := range resp.Operations {
				_ = s.BroadcastOperation(ctx, responseOp)
			}
		}
	}
}

func (s *StandaloneConsensus) discoveryLoop() {
	s.mu.RLock()
	ctx, dht, h := s.ctx, s.dht, s.host
	s.mu.RUnlock()
	if ctx == nil || dht == nil || h == nil {
		return
	}
	rd := routing.NewRoutingDiscovery(dht)
	_, _ = rd.Advertise(ctx, "knirvbase/"+s.config.NetworkID)
	for {
		peers, err := rd.FindPeers(ctx, "knirvbase/"+s.config.NetworkID)
		if err == nil {
			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				if err := h.Connect(ctx, p); err == nil {
					s.recordPeer(p)
					if s.handler != nil {
						_ = s.handler.OnPeerDiscovered(PeerInfo{ID: p.ID.String(), Addresses: addrs(p.Addrs), LastSeen: time.Now()})
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (s *StandaloneConsensus) recordPeer(pi peer.AddrInfo) {
	s.mu.Lock()
	s.peers[pi.ID.String()] = &PeerInfo{ID: pi.ID.String(), Addresses: addrs(pi.Addrs), LastSeen: time.Now()}
	s.mu.Unlock()
}
func addrs(in []multiaddr.Multiaddr) []string {
	out := make([]string, 0, len(in))
	for _, a := range in {
		out = append(out, a.String())
	}
	return out
}
