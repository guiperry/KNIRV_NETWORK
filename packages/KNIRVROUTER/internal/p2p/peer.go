// p2p/peer.go
package p2p

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

// Peer represents a connection to another node
type Peer struct {
	id         string
	addr       string
	conn       net.Conn
	server     *Server
	sendChan   chan Message
	quit       chan struct{}
	wg         sync.WaitGroup
	lastActive time.Time
	mutex      sync.RWMutex
}

// NewPeer creates a new peer
func NewPeer(id, addr string, conn net.Conn, server *Server) *Peer {
	return &Peer{
		id:         id,
		addr:       addr,
		conn:       conn,
		server:     server,
		sendChan:   make(chan Message, 100),
		quit:       make(chan struct{}),
		lastActive: time.Now(),
	}
}

// start begins reading and writing messages
func (p *Peer) start() {
	p.wg.Add(2)
	go p.readLoop()
	go p.writeLoop()
	
	// Start ping timer
	go p.pingLoop()
}

// readLoop continuously reads messages from the peer
func (p *Peer) readLoop() {
	defer p.wg.Done()
	
	reader := bufio.NewReader(p.conn)
	decoder := json.NewDecoder(reader)
	
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			select {
			case <-p.quit:
				// Normal shutdown
				return
			default:
				log.Printf("Error reading from peer %s: %v", p.addr, err)
				p.disconnect()
				return
			}
		}
		
		// Update last active time
		p.mutex.Lock()
		p.lastActive = time.Now()
		p.mutex.Unlock()
		
		// Handle the message
		p.server.handleIncomingMessage(p, msg)
	}
}

// writeLoop continuously sends messages to the peer
func (p *Peer) writeLoop() {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.quit:
			return
		case msg := <-p.sendChan:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshaling message: %v", err)
				continue
			}
			
			// Add newline as message delimiter
			data = append(data, '\n')
			
			// Set write deadline
			p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			
			_, err = p.conn.Write(data)
			if err != nil {
				log.Printf("Error writing to peer %s: %v", p.addr, err)
				p.disconnect()
				return
			}
		}
	}
}

// pingLoop periodically sends ping messages to keep the connection alive
func (p *Peer) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			// Check if peer has been inactive for too long
			p.mutex.RLock()
			inactive := time.Since(p.lastActive) > 90*time.Second
			p.mutex.RUnlock()
			
			if inactive {
				log.Printf("Peer %s inactive for too long, disconnecting", p.addr)
				p.disconnect()
				return
			}
			
			// Send ping
			p.sendMessage(Message{
				Type:    MessageTypePing,
				Payload: []byte("{}"),
			})
		}
	}
}

// sendMessage queues a message to be sent to the peer
func (p *Peer) sendMessage(msg Message) {
	select {
	case p.sendChan <- msg:
		// Message queued successfully
	case <-p.quit:
		// Peer is disconnecting
	default:
		// Channel is full, disconnect peer
		log.Printf("Send channel full for peer %s, disconnecting", p.addr)
		p.disconnect()
	}
}

// disconnect closes the connection to the peer
func (p *Peer) disconnect() {
	// Only close once
	select {
	case <-p.quit:
		return
	default:
		close(p.quit)
	}
	
	p.conn.Close()
	p.server.removePeer(p)
	
	// Wait for goroutines to finish
	p.wg.Wait()
}