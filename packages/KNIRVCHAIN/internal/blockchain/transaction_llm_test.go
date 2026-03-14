package blockchain

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCMU(t *testing.T) {
	tests := []struct {
		name     string
		data     LLMRootingData
		network  string
		expected string
	}{
		{
			name: "Basic CMU generation",
			data: LLMRootingData{
				ModelName:   "GPT-4",
				ModelOwner:  "OpenAI",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
				MetadataCID: "Qm123456789",
			},
			network:  "mainnet",
			expected: "knirv://mainnet/",
		},
		{
			name: "Testnet CMU generation",
			data: LLMRootingData{
				ModelName:   "Claude",
				ModelOwner:  "Anthropic",
				APIEndpoint: "https://api.anthropic.com/v1/messages",
				MetadataCID: "Qm987654321",
			},
			network:  "testnet",
			expected: "knirv://testnet/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmu := GenerateCMU(tt.data, tt.network)
			assert.True(t, len(cmu) > len(tt.expected), "CMU should be longer than prefix")
			assert.Equal(t, tt.expected, cmu[:len(tt.expected)], "CMU prefix should match")
		})
	}
}

func TestNewLLMRootingTransaction(t *testing.T) {
	llmData := LLMRootingData{
		ModelName:   "GPT-4-Turbo",
		ModelOwner:  "OpenAI",
		APIEndpoint: "https://api.openai.com/v1/chat/completions",
		MetadataCID: "Qm123456789abcdef",
	}

	from := "test-owner-address"
	fee := uint64(100)

	tx, err := NewLLMRootingTransaction(from, llmData, fee)

	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, from, tx.From)
	assert.Equal(t, TransactionTypeLLMRooting, tx.Type)
	assert.Equal(t, fee, tx.Fee)
	assert.NotEmpty(t, tx.TransactionHash)
	assert.True(t, tx.Timestamp > 0)

	// Verify data was serialized correctly
	var deserializedData LLMRootingData
	err = json.Unmarshal(tx.Data, &deserializedData)
	require.NoError(t, err)
	assert.Equal(t, llmData.ModelName, deserializedData.ModelName)
	assert.Equal(t, llmData.ModelOwner, deserializedData.ModelOwner)
	assert.Equal(t, llmData.APIEndpoint, deserializedData.APIEndpoint)
	assert.Equal(t, llmData.MetadataCID, deserializedData.MetadataCID)
	assert.NotEmpty(t, deserializedData.CMU)
}

func TestLLMRootingTransactionValidation(t *testing.T) {
	tests := []struct {
		name        string
		llmData     LLMRootingData
		from        string
		expectError bool
	}{
		{
			name: "Valid LLM rooting transaction",
			llmData: LLMRootingData{
				ModelName:   "GPT-4",
				ModelOwner:  "OpenAI",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
				MetadataCID: "Qm123456789",
			},
			from:        "OpenAI",
			expectError: false,
		},
		{
			name: "Missing model name",
			llmData: LLMRootingData{
				ModelOwner:  "OpenAI",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
				MetadataCID: "Qm123456789",
			},
			from:        "OpenAI",
			expectError: true,
		},
		{
			name: "Missing model owner",
			llmData: LLMRootingData{
				ModelName:   "GPT-4",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
				MetadataCID: "Qm123456789",
			},
			from:        "OpenAI",
			expectError: true,
		},
		{
			name: "Missing API endpoint",
			llmData: LLMRootingData{
				ModelName:   "GPT-4",
				ModelOwner:  "OpenAI",
				MetadataCID: "Qm123456789",
			},
			from:        "OpenAI",
			expectError: true,
		},
		{
			name: "Missing metadata CID",
			llmData: LLMRootingData{
				ModelName:   "GPT-4",
				ModelOwner:  "OpenAI",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
			},
			from:        "OpenAI",
			expectError: true,
		},
		{
			name: "Sender doesn't match model owner",
			llmData: LLMRootingData{
				ModelName:   "GPT-4",
				ModelOwner:  "OpenAI",
				APIEndpoint: "https://api.openai.com/v1/chat/completions",
				MetadataCID: "Qm123456789",
			},
			from:        "DifferentOwner",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewLLMRootingTransaction(tt.from, tt.llmData, 100)
			require.NoError(t, err)

			valid := tx.VerifyTxn()
			if tt.expectError {
				assert.False(t, valid)
			} else {
				assert.True(t, valid)
			}
		})
	}
}

func TestCMUConsistency(t *testing.T) {
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	// Generate CMU multiple times
	cmu1 := GenerateCMU(llmData, "mainnet")
	cmu2 := GenerateCMU(llmData, "mainnet")

	// Should be identical
	assert.Equal(t, cmu1, cmu2)

	// Create transaction and verify CMU matches
	tx, err := NewLLMRootingTransaction(llmData.ModelOwner, llmData, 100)
	require.NoError(t, err)

	var txData LLMRootingData
	err = json.Unmarshal(tx.Data, &txData)
	require.NoError(t, err)

	assert.Equal(t, cmu1, txData.CMU)
}

func TestCMUFormat(t *testing.T) {
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	cmu := GenerateCMU(llmData, "mainnet")

	// Should start with knirv://mainnet/
	assert.True(t, strings.HasPrefix(cmu, "knirv://mainnet/"))

	// Should be exactly 64 characters after the prefix (SHA256 hash)
	hashPart := cmu[len("knirv://mainnet/"):]
	assert.Equal(t, 64, len(hashPart))

	// Should be valid hex
	for _, r := range hashPart {
		assert.True(t, (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'), "Invalid hex character: %c", r)
	}
}

func TestLLMRootingDataSerialization(t *testing.T) {
	original := LLMRootingData{
		ModelName:   "GPT-4-Turbo",
		ModelOwner:  "OpenAI",
		APIEndpoint: "https://api.openai.com/v1/chat/completions",
		MetadataCID: "Qm123456789abcdef",
	}

	// Serialize
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize
	var deserialized LLMRootingData
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.ModelName, deserialized.ModelName)
	assert.Equal(t, original.ModelOwner, deserialized.ModelOwner)
	assert.Equal(t, original.APIEndpoint, deserialized.APIEndpoint)
	assert.Equal(t, original.MetadataCID, deserialized.MetadataCID)
	// CMU is generated, so it should be set
	assert.NotEmpty(t, deserialized.CMU)
}
