package api

import (
	"fmt"
	"io"
)

// startFridaServer starts the device-side agent in the sandbox namespace.
// It is deliberately separate from the interactive bridge, which is started
// per attachment and can be stopped without disrupting the target process.
func (s *SandboxSession) startFridaServer() error {
	s.mutex.RLock()
	alreadyStarted := s.fridaServerCmd != nil && s.fridaServerCmd.Process != nil && s.fridaServerCmd.ProcessState == nil
	s.mutex.RUnlock()
	if alreadyStarted {
		return nil
	}
	binary := resolveSandboxTool("frida-server")
	cmd, stdin, stdout, stderr, err := s.startToolProcess(binary, true, "-l", "127.0.0.1:27042")
	if err != nil {
		return fmt.Errorf("start frida-server: %w", err)
	}
	_ = stdin.Close()
	go func() { _, _ = io.Copy(io.Discard, stdout) }()
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	s.mutex.Lock()
	s.fridaServerCmd = cmd
	s.mutex.Unlock()
	go func() {
		err := cmd.Wait()
		s.mutex.Lock()
		if s.fridaServerCmd == cmd {
			s.fridaServerCmd = nil
		}
		s.mutex.Unlock()
		if err != nil {
			s.appendLog("[sandbox] frida-server exited: " + err.Error())
		}
	}()
	return nil
}

func (s *SandboxSession) ensureFridaServer() error {
	if err := ensureToolAvailable("frida-server"); err != nil {
		return err
	}
	return s.startFridaServer()
}
