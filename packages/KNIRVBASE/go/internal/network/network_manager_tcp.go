//go:build legacy_tcp

package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/knirvcorp/knirvbase/internal/types"
)

// startTransport brings up the legacy custom TCP P2P transport. This is only
// compiled when the binary is built with the `legacy_tcp` build tag. New code
// should use the p2pconsensus package instead.
func (n *NetworkManager) startTransport() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("legacy_tcp: failed to resolve TCP address: %v", err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Printf("legacy_tcp: failed to start listener: %v", err)
		return
	}
	listener.SetDeadline(time.Now().Add(1 * time.Second))

	n.mu.Lock()
	n.listener = listener
	n.mu.Unlock()

	log.Printf("legacy_tcp: P2P node listening: %s on %s", n.peerID, listener.Addr().String())
	go n.acceptConnections()
}

func (n *NetworkManager) acceptConnections() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			continue
		default:
			n.listener.SetDeadline(time.Now().Add(500 * time.Millisecond))
			conn, err := n.listener.Accept()
			if err != nil {
				if n.ctx.Err() != nil {
					return
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
					log.Printf("legacy_tcp: accept error: %v", err)
				}
				continue
			}
			n.listener.SetDeadline(time.Now().Add(1 * time.Second))
			n.handleConnection(conn)
		}
	}
}

func (n *NetworkManager) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), 1024)
	if !scanner.Scan() {
		return
	}

	handshake := strings.TrimSpace(scanner.Text())
	parts := strings.Split(handshake, ":")
	if len(parts) != 2 || parts[0] != "KNIRV" {
		return
	}
	peerID := parts[1]
	fmt.Fprintf(conn, "KNIRV:%s\n", n.peerID)

	n.mu.Lock()
	n.connections[peerID] = conn
	n.peers[peerID] = &types.PeerInfo{
		PeerID:   peerID,
		Addrs:    []string{conn.RemoteAddr().String()},
		LastSeen: time.Now(),
	}
	n.mu.Unlock()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg types.ProtocolMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("legacy_tcp: failed to decode message: %v", err)
			continue
		}
		n.handleMessage(msg)
	}
}

func (n *NetworkManager) connectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("legacy_tcp: failed to connect to %s: %v", address, err)
		return
	}
	fmt.Fprintf(conn, "KNIRV:%s\n", n.peerID)

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		conn.Close()
		return
	}
	response := strings.TrimSpace(scanner.Text())
	parts := strings.Split(response, ":")
	if len(parts) != 2 || parts[0] != "KNIRV" {
		conn.Close()
		return
	}
	peerID := parts[1]

	n.mu.Lock()
	n.connections[peerID] = conn
	n.peers[peerID] = &types.PeerInfo{
		PeerID:   peerID,
		Addrs:    []string{address},
		LastSeen: time.Now(),
	}
	n.mu.Unlock()

	log.Printf("legacy_tcp: connected to peer %s at %s", peerID, address)
	go func() {
		defer conn.Close()
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var msg types.ProtocolMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				log.Printf("legacy_tcp: failed to decode message: %v", err)
				continue
			}
			n.handleMessage(msg)
		}
	}()
}
