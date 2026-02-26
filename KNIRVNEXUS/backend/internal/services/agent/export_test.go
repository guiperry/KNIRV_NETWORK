// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

// export_test.go exposes internal AgentService state for white-box testing only.
// This file is compiled exclusively when running tests (Go toolchain convention).

package agent

// AgentsForTest returns the internal agents map so tests in the `web` package
// (and other external test packages) can inject pre-configured AgentStatus
// entries without going through the real container manager.
func (s *AgentService) AgentsForTest() map[string]*AgentStatus {
	return s.agents
}

// TasksForTest returns the internal tasks map so tests can inject or inspect
// AgentTask entries without exercising the full submission pipeline.
func (s *AgentService) TasksForTest() map[string]*AgentTask {
	return s.tasks
}
