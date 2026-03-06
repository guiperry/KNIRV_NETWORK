// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package viewport

import (
	"fmt"
	"log"
)

// VNCSession manages VNC connections to containers
type VNCSession struct {
	containerID string
	vncPort     int
	closed      bool
}

// NewVNCSession creates a new VNC session for a container
func NewVNCSession(containerID string) (*VNCSession, error) {
	session := &VNCSession{
		containerID: containerID,
		vncPort:     5900, // Default VNC port
		closed:      false,
	}

	log.Printf("VNC session created for container %s on port %d", containerID, session.vncPort)

	return session, nil
}

// Close closes the VNC session
func (vs *VNCSession) Close() error {
	if vs.closed {
		return nil
	}

	// TODO: Close actual VNC connection
	vs.closed = true

	log.Printf("VNC session closed for container %s", vs.containerID)

	return nil
}

// GetConnectionString returns the VNC connection string
func (vs *VNCSession) GetConnectionString() string {
	if vs.closed {
		return ""
	}

	return fmt.Sprintf("vnc://localhost:%d", vs.vncPort)
}

// SendFrameBuffer sends a frame buffer update
func (vs *VNCSession) SendFrameBuffer(data []byte) error {
	if vs.closed {
		return fmt.Errorf("VNC session closed")
	}

	// TODO: Implement actual frame buffer sending
	return nil
}

// GetStatus returns the VNC session status
func (vs *VNCSession) GetStatus() string {
	if vs.closed {
		return "closed"
	}
	return "connected"
}
