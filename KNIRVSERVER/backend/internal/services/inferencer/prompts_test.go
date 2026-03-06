package inferencer

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBuildPrompt(t *testing.T) {
	base := "Base prompt"
	context := []string{"ctx1", "ctx2"}
	userInput := "user input"
	prompt := BuildPrompt(base, context, userInput)
	expected := "Base prompt\n\nContext:\n- ctx1\n- ctx2\n\nUser Input: user input"
	assert.Equal(t, expected, prompt)
}

func TestValidatePrompt(t *testing.T) {
	// Valid
	err := ValidatePrompt("valid prompt")
	assert.NoError(t, err)
	// Empty
	err = ValidatePrompt("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
	// Too long
	longPrompt := string(make([]byte, 10001))
	err = ValidatePrompt(longPrompt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}