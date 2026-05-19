// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDB creates a temporary BuntDB instance for testing.
func newTestDB(t *testing.T) *database.BuntDBManager {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.NewBuntDB(filepath.Join(tmpDir, "agent_test.db"))
	require.NoError(t, err, "failed to create test database")
	t.Cleanup(func() { db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNewAgentService(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)

	require.NotNil(t, svc)
	assert.NotNil(t, svc.agents)
	assert.NotNil(t, svc.tasks)
	assert.False(t, svc.running)
}

func TestNewAgentService_NilDB(t *testing.T) {
	// Should construct without panicking even when all deps are nil.
	svc := NewAgentService(nil)
	require.NotNil(t, svc)
	assert.False(t, svc.running)
}

// ---------------------------------------------------------------------------
// Start / Stop lifecycle
// ---------------------------------------------------------------------------

func TestAgentService_Start(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)

	err := svc.Start()
	require.NoError(t, err)
	assert.True(t, svc.running)
}

func TestAgentService_StartTwice(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)

	require.NoError(t, svc.Start())

	// Second Start must return an error.
	err := svc.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestAgentService_Stop(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)

	require.NoError(t, svc.Start())
	require.NoError(t, svc.Stop())
	assert.False(t, svc.running)
}

func TestAgentService_StopWithoutStart(t *testing.T) {
	// Stop before Start must not panic and must succeed.
	svc := NewAgentService(nil)
	err := svc.Stop()
	assert.NoError(t, err)
	assert.False(t, svc.running)
}

func TestAgentService_StartStopCycle(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)

	require.NoError(t, svc.Start())
	require.NoError(t, svc.Stop())

	// After Stop it should be possible to Start again.
	require.NoError(t, svc.Start())
	assert.True(t, svc.running)
	require.NoError(t, svc.Stop())
}

// ---------------------------------------------------------------------------
// GetAgentStatus
// ---------------------------------------------------------------------------

func TestAgentService_GetAgentStatus_NoAgent(t *testing.T) {
	svc := NewAgentService(nil)

	status, err := svc.GetAgentStatus("dve-unknown")
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "dve-unknown", status.DVEID)
	assert.False(t, status.Running)
	// Task counts must be zero when no agent exists.
	assert.Equal(t, 0, status.ActiveTasks)
	assert.Equal(t, 0, status.CompletedTasks)
}

func TestAgentService_GetAgentStatus_CountsTasks(t *testing.T) {
	svc := NewAgentService(nil)

	dveID := "dve-counts"

	// Manually wire up internal state (bypass container manager).
	svc.agents[dveID] = &AgentStatus{
		DVEID:       dveID,
		ContainerID: "ctr-1",
		Running:     true,
	}

	// Add tasks with various statuses.
	svc.tasks["t1"] = &AgentTask{ID: "t1", DVEID: dveID, Status: "pending"}
	svc.tasks["t2"] = &AgentTask{ID: "t2", DVEID: dveID, Status: "running"}
	svc.tasks["t3"] = &AgentTask{ID: "t3", DVEID: dveID, Status: "completed"}
	svc.tasks["t4"] = &AgentTask{ID: "t4", DVEID: dveID, Status: "failed"}
	// Tasks belonging to a different DVE must not be counted.
	svc.tasks["t5"] = &AgentTask{ID: "t5", DVEID: "other-dve", Status: "pending"}

	status, err := svc.GetAgentStatus(dveID)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.True(t, status.Running)
	// "pending" + "running" = 2 active
	assert.Equal(t, 2, status.ActiveTasks)
	// "completed" = 1
	assert.Equal(t, 1, status.CompletedTasks)
}

// ---------------------------------------------------------------------------
// SubmitTask
// ---------------------------------------------------------------------------

func TestAgentService_SubmitTask_NoAgent(t *testing.T) {
	svc := NewAgentService(nil)
	req := &AgentTaskRequest{
		Title:       "test task",
		Description: "desc",
		Type:        "research",
	}

	// SubmitTask no longer checks for a running agent — it records the task
	// for the KNIRVAGENT supervisor.
	task, err := svc.SubmitTask("dve-missing", req)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "pending", task.Status)
}

func TestAgentService_SubmitTask_AgentNotRunning(t *testing.T) {
	svc := NewAgentService(nil)
	dveID := "dve-stopped"
	svc.agents[dveID] = &AgentStatus{DVEID: dveID, Running: false}

	// SubmitTask no longer checks agent running state — it records the task
	// for the KNIRVAGENT supervisor.
	task, err := svc.SubmitTask(dveID, &AgentTaskRequest{Title: "t"})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "pending", task.Status)
}

func TestAgentService_SubmitTask_DefaultsApplied(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)
	dveID := "dve-defaults"

	svc.agents[dveID] = &AgentStatus{
		DVEID:       dveID,
		ContainerID: "ctr-1",
		Running:     true,
	}

	req := &AgentTaskRequest{
		Title: "My Task",
		// Type and Priority are intentionally left zero so defaults kick in.
	}

	task, err := svc.SubmitTask(dveID, req)
	require.NoError(t, err)
	require.NotNil(t, task)

	// Default type must be "research".
	assert.Equal(t, "research", task.Type)
	// Default priority must be 1.
	assert.Equal(t, 1, task.Priority)
	// Status starts as "pending".
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, dveID, task.DVEID)
	assert.NotEmpty(t, task.ID)
}

// ---------------------------------------------------------------------------
// GetTasks
// ---------------------------------------------------------------------------

func TestAgentService_GetTasks_EmptyForUnknownDVE(t *testing.T) {
	svc := NewAgentService(nil)

	tasks, err := svc.GetTasks("dve-none")
	require.NoError(t, err)
	// Should return nil/empty slice, not an error.
	assert.Empty(t, tasks)
}

func TestAgentService_GetTasks_FiltersByDVE(t *testing.T) {
	svc := NewAgentService(nil)

	svc.tasks["ta"] = &AgentTask{ID: "ta", DVEID: "dve-a", Status: "pending"}
	svc.tasks["tb"] = &AgentTask{ID: "tb", DVEID: "dve-a", Status: "completed"}
	svc.tasks["tc"] = &AgentTask{ID: "tc", DVEID: "dve-b", Status: "pending"}

	tasksA, err := svc.GetTasks("dve-a")
	require.NoError(t, err)
	assert.Len(t, tasksA, 2)

	tasksB, err := svc.GetTasks("dve-b")
	require.NoError(t, err)
	assert.Len(t, tasksB, 1)
}

// ---------------------------------------------------------------------------
// GetTask
// ---------------------------------------------------------------------------

func TestAgentService_GetTask_NotFound(t *testing.T) {
	svc := NewAgentService(nil)

	task, err := svc.GetTask("non-existent-id")
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "task not found")
}

func TestAgentService_GetTask_Found(t *testing.T) {
	svc := NewAgentService(nil)

	expected := &AgentTask{
		ID:    "task-42",
		DVEID: "dve-x",
		Title: "Run benchmarks",
	}
	svc.tasks[expected.ID] = expected

	got, err := svc.GetTask("task-42")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected.ID, got.ID)
	assert.Equal(t, expected.Title, got.Title)
}

// ---------------------------------------------------------------------------
// Task persistence in DB
// ---------------------------------------------------------------------------

func TestAgentService_TaskPersistence_SaveAndLoad(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgentService(db)
	dveID := "dve-persist"

	svc.agents[dveID] = &AgentStatus{
		DVEID:       dveID,
		ContainerID: "ctr-persist",
		Running:     true,
	}

	req := &AgentTaskRequest{
		Title:       "Persist Task",
		Description: "Test persistence",
		Type:        "analysis",
		Priority:    3,
	}

	task, err := svc.SubmitTask(dveID, req)
	require.NoError(t, err)
	require.NotNil(t, task)
	taskID := task.ID

	// Read directly from DB to verify the record was written.
	var stored AgentTask
	err = db.GetJSON("agent:task:"+taskID, &stored)
	require.NoError(t, err, "task should be persisted to DB immediately after SubmitTask")

	assert.Equal(t, taskID, stored.ID)
	assert.Equal(t, dveID, stored.DVEID)
	assert.Equal(t, "Persist Task", stored.Title)
	assert.Equal(t, "analysis", stored.Type)
	assert.Equal(t, 3, stored.Priority)
}

func TestAgentService_TaskPersistence_LoadOnStart(t *testing.T) {
	db := newTestDB(t)

	// Write a task record directly to the DB so we can verify loadTasksFromDB.
	taskID := "pre-existing-task"
	now := time.Now()
	preExisting := &AgentTask{
		ID:        taskID,
		DVEID:     "dve-reload",
		Title:     "Pre-existing",
		Status:    "completed",
		CreatedAt: now,
		UpdatedAt: now,
	}
	data, err := json.Marshal(preExisting)
	require.NoError(t, err)

	// Write the JSON string directly under the agent:task: prefix.
	err = db.SetValue("agent:task:"+taskID, string(data))
	require.NoError(t, err)

	// A fresh service loading from this DB should restore the task.
	svc2 := NewAgentService(db)
	require.NoError(t, svc2.Start())
	defer svc2.Stop()

	got, err := svc2.GetTask(taskID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Pre-existing", got.Title)
	assert.Equal(t, "completed", got.Status)
}

func TestAgentService_saveTaskToDB_NilDB(t *testing.T) {
	// saveTaskToDB must be a no-op (not panic) when db is nil.
	svc := NewAgentService(nil)
	task := &AgentTask{ID: "t1", DVEID: "dve-1", Title: "x"}
	err := svc.saveTaskToDB(task)
	assert.NoError(t, err)
}

func TestAgentService_loadTasksFromDB_NilDB(t *testing.T) {
	// loadTasksFromDB must be a no-op (not panic) when db is nil.
	svc := NewAgentService(nil)
	err := svc.loadTasksFromDB()
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AgentTask struct
// ---------------------------------------------------------------------------

func TestAgentTask_JSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	task := AgentTask{
		ID:          "round-trip-id",
		DVEID:       "dve-rt",
		ContainerID: "ctr-rt",
		Title:       "RT Task",
		Description: "desc",
		Type:        "coding",
		Status:      "pending",
		Priority:    5,
		Input:       map[string]string{"key": "value"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded AgentTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.DVEID, decoded.DVEID)
	assert.Equal(t, task.Title, decoded.Title)
	assert.Equal(t, task.Type, decoded.Type)
	assert.Equal(t, task.Priority, decoded.Priority)
	assert.Equal(t, task.Input, decoded.Input)
}

// ---------------------------------------------------------------------------
// AgentStatus struct
// ---------------------------------------------------------------------------

func TestAgentStatus_JSONRoundtrip(t *testing.T) {
	status := AgentStatus{
		DVEID:          "dve-st",
		ContainerID:    "ctr-st",
		Running:        true,
		ViewportURL:    "/api/dve/dve-st/agent/viewport",
		ActiveTasks:    3,
		CompletedTasks: 7,
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded AgentStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, status.DVEID, decoded.DVEID)
	assert.Equal(t, status.Running, decoded.Running)
	assert.Equal(t, status.ActiveTasks, decoded.ActiveTasks)
	assert.Equal(t, status.CompletedTasks, decoded.CompletedTasks)
	assert.Equal(t, status.ViewportURL, decoded.ViewportURL)
}

func TestAgentTaskRequest_JSONRoundtrip(t *testing.T) {
	req := AgentTaskRequest{
		Title:       "Test Task",
		Description: "Test description",
		Type:        "coding",
		Priority:    2,
		Input:       map[string]string{"source": "test"},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded AgentTaskRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Title, decoded.Title)
	assert.Equal(t, req.Description, decoded.Description)
	assert.Equal(t, req.Type, decoded.Type)
	assert.Equal(t, req.Priority, decoded.Priority)
	assert.Equal(t, req.Input, decoded.Input)
}
