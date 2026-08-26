package main

import "testing"

func TestShouldRefuseRootLaunch(t *testing.T) {
	if !shouldRefuseRootLaunch(0) {
		t.Fatal("root launch must be refused so Electron stays sandboxed")
	}
	if shouldRefuseRootLaunch(1000) {
		t.Fatal("normal desktop user launch must be allowed")
	}
}
