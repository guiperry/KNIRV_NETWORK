package inference

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gollm_types "github.com/guiperry/gollm_cerebras/types"
)

func TestNewSimpleWindowMemory(t *testing.T) {
	t.Run("creates new memory with default model", func(t *testing.T) {
		defaultModel := "gpt-4"
		memory := NewSimpleWindowMemory(defaultModel)
		
		require.NotNil(t, memory)
		assert.Equal(t, defaultModel, memory.defaultModelName)
		assert.Empty(t, memory.messages)
		assert.NotNil(t, memory.messages) // Should be initialized, not nil
	})

	t.Run("creates new memory with empty model name", func(t *testing.T) {
		memory := NewSimpleWindowMemory("")
		
		require.NotNil(t, memory)
		assert.Equal(t, "", memory.defaultModelName)
		assert.Empty(t, memory.messages)
	})
}

func TestSimpleWindowMemory_AddMessage(t *testing.T) {
	memory := NewSimpleWindowMemory("gpt-4")

	t.Run("adds single message", func(t *testing.T) {
		message := gollm_types.MemoryMessage{
			Role:    "user",
			Content: "Hello, world!",
		}

		memory.AddMessage(message)
		
		history := memory.GetHistory()
		require.Len(t, history, 1)
		assert.Equal(t, message.Role, history[0].Role)
		assert.Equal(t, message.Content, history[0].Content)
	})

	t.Run("adds multiple messages in order", func(t *testing.T) {
		memory.Clear() // Start fresh
		
		messages := []gollm_types.MemoryMessage{
			{Role: "user", Content: "First message"},
			{Role: "assistant", Content: "Second message"},
			{Role: "user", Content: "Third message"},
		}

		for _, msg := range messages {
			memory.AddMessage(msg)
		}

		history := memory.GetHistory()
		require.Len(t, history, 3)
		
		for i, expectedMsg := range messages {
			assert.Equal(t, expectedMsg.Role, history[i].Role)
			assert.Equal(t, expectedMsg.Content, history[i].Content)
		}
	})

	t.Run("handles concurrent access", func(t *testing.T) {
		memory.Clear() // Start fresh
		
		var wg sync.WaitGroup
		numGoroutines := 10
		messagesPerGoroutine := 5

		// Add messages concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					message := gollm_types.MemoryMessage{
						Role:    "user",
						Content: "Message from goroutine",
					}
					memory.AddMessage(message)
				}
			}(i)
		}

		wg.Wait()

		history := memory.GetHistory()
		expectedTotal := numGoroutines * messagesPerGoroutine
		assert.Len(t, history, expectedTotal)
	})
}

func TestSimpleWindowMemory_GetMessagesForContext(t *testing.T) {
	memory := NewSimpleWindowMemory("gpt-4")

	// Add test messages
	testMessages := []gollm_types.MemoryMessage{
		{Role: "user", Content: "Short"},      // ~1 token
		{Role: "assistant", Content: "Medium length message"}, // ~3 tokens
		{Role: "user", Content: "This is a longer message with more content"}, // ~9 tokens
		{Role: "assistant", Content: "Final response"}, // ~2 tokens
	}

	for _, msg := range testMessages {
		memory.AddMessage(msg)
	}

	t.Run("returns all messages when token limit is high", func(t *testing.T) {
		messages := memory.GetMessagesForContext(1000, "gpt-4")
		assert.Len(t, messages, 4)
		
		// Should maintain chronological order
		assert.Equal(t, "Short", messages[0].Content)
		assert.Equal(t, "Final response", messages[3].Content)
	})

	t.Run("returns subset when token limit is low", func(t *testing.T) {
		// With a low token limit, should return only the most recent messages
		messages := memory.GetMessagesForContext(5, "gpt-4")
		
		// Should get the most recent messages that fit
		assert.True(t, len(messages) > 0)
		assert.True(t, len(messages) <= 4)
		
		// Last message should be the most recent
		lastMessage := messages[len(messages)-1]
		assert.Equal(t, "Final response", lastMessage.Content)
	})

	t.Run("returns empty slice when token limit is zero", func(t *testing.T) {
		messages := memory.GetMessagesForContext(0, "gpt-4")
		assert.Empty(t, messages)
	})

	t.Run("uses default model when model name is empty", func(t *testing.T) {
		messages := memory.GetMessagesForContext(1000, "")
		assert.Len(t, messages, 4) // Should still work with default model
	})

	t.Run("uses specified model for token estimation", func(t *testing.T) {
		messages := memory.GetMessagesForContext(1000, "different-model")
		assert.Len(t, messages, 4) // Should work with different model
	})

	t.Run("handles empty memory", func(t *testing.T) {
		emptyMemory := NewSimpleWindowMemory("gpt-4")
		messages := emptyMemory.GetMessagesForContext(1000, "gpt-4")
		assert.Empty(t, messages)
	})
}

func TestSimpleWindowMemory_Clear(t *testing.T) {
	memory := NewSimpleWindowMemory("gpt-4")

	t.Run("clears all messages", func(t *testing.T) {
		// Add some messages
		messages := []gollm_types.MemoryMessage{
			{Role: "user", Content: "Message 1"},
			{Role: "assistant", Content: "Message 2"},
			{Role: "user", Content: "Message 3"},
		}

		for _, msg := range messages {
			memory.AddMessage(msg)
		}

		// Verify messages were added
		history := memory.GetHistory()
		assert.Len(t, history, 3)

		// Clear and verify
		memory.Clear()
		history = memory.GetHistory()
		assert.Empty(t, history)
	})

	t.Run("clear on empty memory is safe", func(t *testing.T) {
		emptyMemory := NewSimpleWindowMemory("gpt-4")
		
		// Should not panic
		assert.NotPanics(t, func() {
			emptyMemory.Clear()
		})
		
		history := emptyMemory.GetHistory()
		assert.Empty(t, history)
	})

	t.Run("handles concurrent clear operations", func(t *testing.T) {
		// Add some messages
		for i := 0; i < 10; i++ {
			memory.AddMessage(gollm_types.MemoryMessage{
				Role:    "user",
				Content: "Test message",
			})
		}

		var wg sync.WaitGroup
		numGoroutines := 5

		// Clear concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				memory.Clear()
			}()
		}

		wg.Wait()

		history := memory.GetHistory()
		assert.Empty(t, history)
	})
}

func TestSimpleWindowMemory_GetHistory(t *testing.T) {
	memory := NewSimpleWindowMemory("gpt-4")

	t.Run("returns copy of messages", func(t *testing.T) {
		originalMessage := gollm_types.MemoryMessage{
			Role:    "user",
			Content: "Original content",
		}
		memory.AddMessage(originalMessage)

		history := memory.GetHistory()
		require.Len(t, history, 1)

		// Modify the returned slice - should not affect internal state
		history[0].Content = "Modified content"

		// Verify internal state is unchanged
		internalHistory := memory.GetHistory()
		assert.Equal(t, "Original content", internalHistory[0].Content)
	})

	t.Run("returns empty slice for empty memory", func(t *testing.T) {
		emptyMemory := NewSimpleWindowMemory("gpt-4")
		history := emptyMemory.GetHistory()
		
		assert.NotNil(t, history)
		assert.Empty(t, history)
	})

	t.Run("handles concurrent read access", func(t *testing.T) {
		// Add test messages
		for i := 0; i < 5; i++ {
			memory.AddMessage(gollm_types.MemoryMessage{
				Role:    "user",
				Content: "Test message",
			})
		}

		var wg sync.WaitGroup
		numGoroutines := 10
		results := make([][]gollm_types.MemoryMessage, numGoroutines)

		// Read concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				results[index] = memory.GetHistory()
			}(i)
		}

		wg.Wait()

		// All results should be identical
		for i := 1; i < numGoroutines; i++ {
			assert.Equal(t, len(results[0]), len(results[i]))
		}
	})
}

func TestSimpleWindowMemory_InterfaceCompliance(t *testing.T) {
	t.Run("implements ConversationMemory interface", func(t *testing.T) {
		var memory ConversationMemory = NewSimpleWindowMemory("gpt-4")
		
		// Should be able to call interface methods
		assert.NotPanics(t, func() {
			memory.AddMessage(gollm_types.MemoryMessage{Role: "user", Content: "test"})
			memory.GetMessagesForContext(100, "gpt-4")
			memory.GetHistory()
			memory.Clear()
		})
	})
}
