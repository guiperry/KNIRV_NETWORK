//go:build !unix

package knirvbao

import "errors"

func (m *Manager) lockRoot(string) error {
	return errors.New("OpenBao manager is unsupported on this platform")
}
func (m *Manager) releaseLock() {}
