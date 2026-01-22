package types

import (
	"sync"
	"time"
)

type Peer struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Status    bool   `json:"status"`
	LastPing  int64  `json:"last_ping"`
	LastTx    int64  `json:"last_tx"`
	LastBlock int64  `json:"last_block"`
	LastCons  int64  `json:"last_cons"`
	LastSync  int64  `json:"last_sync"`
	LastFetch int64  `json:"last_fetch"`
}

type PeerManager struct {
	Peers      map[string]Peer `json:"peers"`
	Address    string          `json:"address"`
	mu         sync.Mutex
}

func NewPeerManager(address string) *PeerManager {
	return &PeerManager{
		Peers:   make(map[string]Peer),
		Address: address,
	}
}

func (pm *PeerManager) AddPeer(p Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.Peers[p.ID] = p
}

func (pm *PeerManager) GetPeers() map[string]Peer {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.Peers
}

func (pm *PeerManager) UpdatePeers(peersList map[string]bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for id, status := range peersList {
		if peer, exists := pm.Peers[id]; exists {
			peer.Status = status
			peer.LastPing = time.Now().Unix()
			pm.Peers[id] = peer
		} else {
			pm.Peers[id] = Peer{
				ID:      id,
				Address: id, // Using ID as Address for new peers
				Status:  status,
				LastPing: time.Now().Unix(),
			}
		}
	}
}

func (pm *PeerManager) RemovePeer(id string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.Peers, id)
}

func (pm *PeerManager) GetPeer(id string) (Peer, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	peer, ok := pm.Peers[id]
	return peer, ok
}