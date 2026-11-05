package main

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	cleanup, err := installEmbeddedCore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "install test SDK core:", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
