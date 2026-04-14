package knirvshell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBinaryDir(t *testing.T) {
	t.Run("returns default when no env vars set", func(t *testing.T) {
		os.Unsetenv("KNIRV_SHELL_BINARY_DIR")
		os.Unsetenv("XDG_DATA_HOME")

		dir, err := defaultBinaryDir()
		require.NoError(t, err)
		assert.True(t, strings.Contains(dir, ".local/share/knirvserver/bin"))
	})

	t.Run("respects KNIRV_SHELL_BINARY_DIR env var", func(t *testing.T) {
		os.Setenv("KNIRV_SHELL_BINARY_DIR", "/custom/bin")
		defer os.Unsetenv("KNIRV_SHELL_BINARY_DIR")

		dir, err := defaultBinaryDir()
		require.NoError(t, err)
		assert.Equal(t, "/custom/bin", dir)
	})

	t.Run("respects XDG_DATA_HOME when KNIRV_SHELL_BINARY_DIR not set", func(t *testing.T) {
		os.Unsetenv("KNIRV_SHELL_BINARY_DIR")
		os.Setenv("XDG_DATA_HOME", "/custom/data")
		defer os.Unsetenv("XDG_DATA_HOME")

		dir, err := defaultBinaryDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/custom/data", "knirvserver", "bin"), dir)
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates directory recursively", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "nested", "dir", "test")

		err := ensureDir(testPath)
		require.NoError(t, err)

		info, err := os.Stat(testPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := ensureDir(tmpDir)
		require.NoError(t, err)

		info, err := os.Stat(tmpDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestWriteFileAtomically(t *testing.T) {
	t.Run("writes file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "test.bin")
		testData := []byte("test data content")

		err := writeFileAtomically(testPath, testData, 0755)
		require.NoError(t, err)

		data, err := os.ReadFile(testPath)
		require.NoError(t, err)
		assert.Equal(t, testData, data)

		info, err := os.Stat(testPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})
}

func TestExtractEmbeddedBinary(t *testing.T) {
	t.Run("extracts to default directory when empty destDir", func(t *testing.T) {
		if !IsEmbeddedBinaryAvailable() {
			t.Skip("No embedded binary available")
		}

		tmpDir := t.TempDir()

		path, err := ExtractEmbeddedBinary(tmpDir)
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, !info.IsDir())
		assert.Equal(t, "knirvshell", filepath.Base(path))
	})

	t.Run("fails when embedded binary is empty", func(t *testing.T) {
		embeddedBinary = []byte{}

		path, err := ExtractEmbeddedBinary(t.TempDir())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embedded knirvshell binary is empty")
		assert.Empty(t, path)

		embeddedBinary = []byte{1, 2, 3}
	})
}

func TestGetBinaryPath(t *testing.T) {
	t.Run("returns embedded path when available", func(t *testing.T) {
		if !IsEmbeddedBinaryAvailable() {
			t.Skip("No embedded binary available")
		}

		os.Unsetenv("KNIRV_SHELL_PATH")
		os.Unsetenv("KNIRV_SHELL_BINARY_DIR")

		path, err := GetBinaryPath()
		require.NoError(t, err)
		assert.Equal(t, "knirvshell", filepath.Base(path))
	})

	t.Run("respects KNIRV_SHELL_PATH env var", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "knirvshell")
		err := os.WriteFile(testPath, []byte("mock"), 0755)
		require.NoError(t, err)

		os.Setenv("KNIRV_SHELL_PATH", testPath)
		defer os.Unsetenv("KNIRV_SHELL_PATH")

		path, err := GetBinaryPath()
		require.NoError(t, err)
		assert.Equal(t, testPath, path)
	})
}

func TestIsEmbeddedBinaryAvailable(t *testing.T) {
	t.Run("returns true when binary is embedded", func(t *testing.T) {
		if !IsEmbeddedBinaryAvailable() {
			t.Skip("No embedded binary available")
		}
		assert.True(t, IsEmbeddedBinaryAvailable())
	})
}

func TestResolveBinaryPath(t *testing.T) {
	t.Run("returns embedded path when available", func(t *testing.T) {
		if !IsEmbeddedBinaryAvailable() {
			t.Skip("No embedded binary available")
		}

		os.Unsetenv("KNIRV_SHELL_PATH")
		os.Unsetenv("KNIRV_SHELL_BINARY_DIR")

		path, err := ResolveBinaryPath()
		require.NoError(t, err)
		assert.Equal(t, "knirvshell", filepath.Base(path))
	})

	t.Run("respects KNIRV_SHELL_PATH env var", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "knirvshell")
		err := os.WriteFile(testPath, []byte("mock"), 0755)
		require.NoError(t, err)

		os.Setenv("KNIRV_SHELL_PATH", testPath)
		defer os.Unsetenv("KNIRV_SHELL_PATH")

		path, err := ResolveBinaryPath()
		require.NoError(t, err)
		assert.Equal(t, testPath, path)
	})
}

func TestNewKNIRVSHELLService(t *testing.T) {
	t.Run("creates service with valid binary path", func(t *testing.T) {
		if !IsEmbeddedBinaryAvailable() {
			t.Skip("No embedded binary available")
		}

		service, err := NewKNIRVSHELLService()
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotEmpty(t, service.binaryPath)
		assert.Equal(t, "knirvshell", filepath.Base(service.binaryPath))
	})
}

func TestCommandRequestSerialization(t *testing.T) {
	t.Run("marshals and unmarshals correctly", func(t *testing.T) {
		req := &CommandRequest{
			Command:  "help",
			NodeID:   "node-123",
			Args:     []string{"--verbose"},
			Env:      map[string]string{"DEBUG": "true"},
			Timeout:  30,
			UserID:   "user-456",
			Username: "testuser",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CommandRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, req.Command, decoded.Command)
		assert.Equal(t, req.NodeID, decoded.NodeID)
		assert.Equal(t, req.Args, decoded.Args)
		assert.Equal(t, req.Env, decoded.Env)
		assert.Equal(t, req.Timeout, decoded.Timeout)
		assert.Equal(t, req.UserID, decoded.UserID)
		assert.Equal(t, req.Username, decoded.Username)
	})
}

func TestWalletInfoSerialization(t *testing.T) {
	t.Run("marshals and unmarshals correctly", func(t *testing.T) {
		info := &WalletInfo{
			Address:  "xion1abc123",
			Balance:  1000000,
			Currency: "uxion",
		}

		data, err := json.Marshal(info)
		require.NoError(t, err)

		var decoded WalletInfo
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, info.Address, decoded.Address)
		assert.Equal(t, info.Balance, decoded.Balance)
		assert.Equal(t, info.Currency, decoded.Currency)
	})
}

func TestTokenTransactionSerialization(t *testing.T) {
	t.Run("marshals and unmarshals correctly", func(t *testing.T) {
		tx := &TokenTransaction{
			TxHash: "tx123abc",
			From:   "xion1sender",
			To:     "xion1receiver",
			Amount: 500,
			Type:   "transfer",
			Status: "pending",
		}

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		var decoded TokenTransaction
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, tx.TxHash, decoded.TxHash)
		assert.Equal(t, tx.From, decoded.From)
		assert.Equal(t, tx.To, decoded.To)
		assert.Equal(t, tx.Amount, decoded.Amount)
		assert.Equal(t, tx.Type, decoded.Type)
		assert.Equal(t, tx.Status, decoded.Status)
	})
}

func TestBadgeInfoSerialization(t *testing.T) {
	t.Run("marshals and unmarshals correctly", func(t *testing.T) {
		badge := &BadgeInfo{
			ID:          "badge-123",
			Name:        "Test Badge",
			BadgeType:   "achievement",
			Description: "A test badge",
			ImageData:   "base64data...",
			Metadata:    map[string]interface{}{"level": 5},
			Minted:      true,
			AgentID:     "agent-456",
		}

		data, err := json.Marshal(badge)
		require.NoError(t, err)

		var decoded BadgeInfo
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, badge.ID, decoded.ID)
		assert.Equal(t, badge.Name, decoded.Name)
		assert.Equal(t, badge.BadgeType, decoded.BadgeType)
		assert.Equal(t, badge.Description, decoded.Description)
		assert.Equal(t, badge.Minted, decoded.Minted)
		assert.Equal(t, badge.AgentID, decoded.AgentID)
	})
}

func TestBadgeCreateRequestSerialization(t *testing.T) {
	t.Run("marshals and unmarshals correctly", func(t *testing.T) {
		req := &BadgeCreateRequest{
			Name:        "New Badge",
			BadgeType:   "skill",
			Description: "A skill badge",
			ImageData:   "base64...",
			Metadata:    map[string]interface{}{"difficulty": "hard"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded BadgeCreateRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, req.Name, decoded.Name)
		assert.Equal(t, req.BadgeType, decoded.BadgeType)
		assert.Equal(t, req.Description, decoded.Description)
		assert.Equal(t, req.ImageData, decoded.ImageData)
	})
}

func TestTerminalSessionStructure(t *testing.T) {
	t.Run("session can be created with all fields", func(t *testing.T) {
		session := &TerminalSession{
			ID:        "session-123",
			NodeID:    "node-456",
			UserID:    "user-789",
			Username:  "testuser",
			Command:   "status",
			Status:    "running",
			StartTime: time.Now(),
			Output:    []string{"line1", "line2"},
			ExitCode:  0,
		}

		assert.Equal(t, "session-123", session.ID)
		assert.Equal(t, "node-456", session.NodeID)
		assert.Equal(t, "user-789", session.UserID)
		assert.Equal(t, "testuser", session.Username)
		assert.Equal(t, "status", session.Command)
		assert.Equal(t, "running", session.Status)
		assert.NotZero(t, session.StartTime)
		assert.Equal(t, []string{"line1", "line2"}, session.Output)
		assert.Equal(t, 0, session.ExitCode)
	})
}

func TestCommandResultStructure(t *testing.T) {
	t.Run("result can be created with all fields", func(t *testing.T) {
		result := &CommandResult{
			SessionID: "session-123",
			Command:   "help",
			Output:    []string{"output1", "output2"},
			ExitCode:  0,
			Status:    "completed",
			StartTime: "2024-01-01T00:00:00Z",
			EndTime:   "2024-01-01T00:01:00Z",
		}

		assert.Equal(t, "session-123", result.SessionID)
		assert.Equal(t, "help", result.Command)
		assert.Equal(t, []string{"output1", "output2"}, result.Output)
		assert.Equal(t, 0, result.ExitCode)
		assert.Equal(t, "completed", result.Status)
		assert.NotEmpty(t, result.StartTime)
		assert.NotEmpty(t, result.EndTime)
	})
}
