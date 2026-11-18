// Package testhooks holds crash-injection seams that exist only for tests.
//
// Production code never installs a hook, and nothing reads configuration or the
// environment to enable one. Installing a hook is also inert outside a test
// binary: testing.Testing reports false in a normal build, so SetLedgerFault
// does nothing there and the ledger's commit path never sees a fault.
package testhooks

import (
	"sync"
	"testing"
)

// LedgerFault is called at each ledger commit stage ("before_write",
// "after_write", "after_sync"). A non-nil return simulates a crash at that point.
type LedgerFault func(stage string) error

var (
	ledgerMu     sync.RWMutex
	ledgerFaults = map[string]LedgerFault{}
)

// SetLedgerFault installs hook for the ledger log at logPath and returns a
// function that removes it. Outside a test binary it installs nothing.
func SetLedgerFault(logPath string, hook LedgerFault) (restore func()) {
	if !testing.Testing() || hook == nil {
		return func() {}
	}
	ledgerMu.Lock()
	ledgerFaults[logPath] = hook
	ledgerMu.Unlock()
	return func() {
		ledgerMu.Lock()
		delete(ledgerFaults, logPath)
		ledgerMu.Unlock()
	}
}

// LedgerFaultAt runs the hook installed for logPath at one commit stage. With no
// hook installed it returns nil. The ledger calls it on every commit stage.
func LedgerFaultAt(logPath, stage string) error {
	ledgerMu.RLock()
	hook := ledgerFaults[logPath]
	ledgerMu.RUnlock()
	if hook == nil {
		return nil
	}
	return hook(stage)
}
