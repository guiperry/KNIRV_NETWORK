package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guiperry/KNIRVCHAIN-CLI/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewFormField(t *testing.T) {
	field := NewFormField("Username", "Enter username", "testuser", true)

	assert.Equal(t, "Username", field.Label)
	assert.Equal(t, "Enter username", field.Input.Placeholder)
	assert.Equal(t, "testuser", field.Input.Value())
	assert.True(t, field.Required)
	assert.Equal(t, "", field.Error)
	assert.Nil(t, field.Validator)
}

func TestNewPasswordField(t *testing.T) {
	field := NewPasswordField("Password", "Enter password", "secret", true)

	assert.Equal(t, "Password", field.Label)
	assert.Equal(t, "Enter password", field.Input.Placeholder)
	assert.Equal(t, "secret", field.Input.Value())
	assert.True(t, field.Required)
	assert.Equal(t, "", field.Error)
	assert.Nil(t, field.Validator)
	assert.Equal(t, 1, field.Input.EchoMode) // EchoPassword
	assert.Equal(t, '•', field.Input.EchoCharacter)
}

func TestSetValidator(t *testing.T) {
	field := NewFormField("Username", "Enter username", "", true)
	validator := func(value string) error {
		return nil
	}

	field.SetValidator(validator)
	assert.NotNil(t, field.Validator)
	assert.Equal(t, validator, field.Validator)
}

func TestSetDescription(t *testing.T) {
	field := NewFormField("Username", "Enter username", "", true)
	field.SetDescription("Your username for login")

	assert.Equal(t, "Your username for login", field.Description)
}

func TestValidate(t *testing.T) {
	// Test required field with empty value
	field := NewFormField("Username", "Enter username", "", true)
	valid := field.Validate()

	assert.False(t, valid)
	assert.Equal(t, "This field is required", field.Error)

	// Test required field with non-empty value
	field = NewFormField("Username", "Enter username", "testuser", true)
	valid = field.Validate()

	assert.True(t, valid)
	assert.Equal(t, "", field.Error)

	// Test optional field with empty value
	field = NewFormField("Username", "Enter username", "", false)
	valid = field.Validate()

	assert.True(t, valid)
	assert.Equal(t, "", field.Error)

	// Test field with validator that passes
	field = NewFormField("Username", "Enter username", "testuser", true)
	field.SetValidator(func(value string) error {
		return nil
	})
	valid = field.Validate()

	assert.True(t, valid)
	assert.Equal(t, "", field.Error)

	// Test field with validator that fails
	field = NewFormField("Username", "Enter username", "testuser", true)
	field.SetValidator(func(value string) error {
		return assert.AnError
	})
	valid = field.Validate()

	assert.False(t, valid)
	assert.Equal(t, assert.AnError.Error(), field.Error)
}

func TestNewForm(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	assert.Equal(t, 0, len(form.Fields))
	assert.Equal(t, 0, form.FocusIndex)
	assert.Equal(t, styles, form.styles)
	assert.Equal(t, 60, form.width)
	assert.Equal(t, "Submit", form.submitLabel)
	assert.Equal(t, "Cancel", form.cancelLabel)
}

func TestAddField(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	field1 := NewFormField("Username", "Enter username", "", true)
	form.AddField(field1)

	assert.Equal(t, 1, len(form.Fields))
	assert.Equal(t, field1, form.Fields[0])
	assert.True(t, form.Fields[0].Input.Focused())

	field2 := NewFormField("Password", "Enter password", "", true)
	form.AddField(field2)

	assert.Equal(t, 2, len(form.Fields))
	assert.Equal(t, field2, form.Fields[1])
	assert.True(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
}

func TestSetSubmitLabel(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)
	form.SetSubmitLabel("Save")

	assert.Equal(t, "Save", form.submitLabel)
}

func TestSetCancelLabel(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)
	form.SetCancelLabel("Back")

	assert.Equal(t, "Back", form.cancelLabel)
}

func TestFocus(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test focus with no fields
	form.Focus()
	assert.Equal(t, 0, form.FocusIndex)

	// Test focus with fields
	field1 := NewFormField("Username", "Enter username", "", true)
	field2 := NewFormField("Password", "Enter password", "", true)
	form.AddField(field1)
	form.AddField(field2)

	form.Fields[0].Input.Blur()
	form.Fields[1].Input.Blur()
	form.FocusIndex = 1

	form.Focus()
	assert.Equal(t, 0, form.FocusIndex)
	assert.True(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
}

func TestBlur(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test blur with no fields
	form.Blur()
	assert.Equal(t, 0, form.FocusIndex)

	// Test blur with fields
	field1 := NewFormField("Username", "Enter username", "", true)
	field2 := NewFormField("Password", "Enter password", "", true)
	form.AddField(field1)
	form.AddField(field2)

	form.Fields[0].Input.Focus()
	form.FocusIndex = 0

	form.Blur()
	assert.False(t, form.Fields[0].Input.Focused())
}

func TestNextField(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test next field with no fields
	form.NextField()
	assert.Equal(t, 0, form.FocusIndex)

	// Test next field with fields
	field1 := NewFormField("Username", "Enter username", "", true)
	field2 := NewFormField("Password", "Enter password", "", true)
	field3 := NewFormField("Email", "Enter email", "", true)
	form.AddField(field1)
	form.AddField(field2)
	form.AddField(field3)

	form.Fields[0].Input.Focus()
	form.Fields[1].Input.Blur()
	form.Fields[2].Input.Blur()
	form.FocusIndex = 0

	form.NextField()
	assert.Equal(t, 1, form.FocusIndex)
	assert.False(t, form.Fields[0].Input.Focused())
	assert.True(t, form.Fields[1].Input.Focused())
	assert.False(t, form.Fields[2].Input.Focused())

	form.NextField()
	assert.Equal(t, 2, form.FocusIndex)
	assert.False(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
	assert.True(t, form.Fields[2].Input.Focused())

	form.NextField()
	assert.Equal(t, 0, form.FocusIndex)
	assert.True(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
	assert.False(t, form.Fields[2].Input.Focused())
}

func TestPrevField(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test prev field with no fields
	form.PrevField()
	assert.Equal(t, 0, form.FocusIndex)

	// Test prev field with fields
	field1 := NewFormField("Username", "Enter username", "", true)
	field2 := NewFormField("Password", "Enter password", "", true)
	field3 := NewFormField("Email", "Enter email", "", true)
	form.AddField(field1)
	form.AddField(field2)
	form.AddField(field3)

	form.Fields[0].Input.Focus()
	form.Fields[1].Input.Blur()
	form.Fields[2].Input.Blur()
	form.FocusIndex = 0

	form.PrevField()
	assert.Equal(t, 2, form.FocusIndex)
	assert.False(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
	assert.True(t, form.Fields[2].Input.Focused())

	form.PrevField()
	assert.Equal(t, 1, form.FocusIndex)
	assert.False(t, form.Fields[0].Input.Focused())
	assert.True(t, form.Fields[1].Input.Focused())
	assert.False(t, form.Fields[2].Input.Focused())

	form.PrevField()
	assert.Equal(t, 0, form.FocusIndex)
	assert.True(t, form.Fields[0].Input.Focused())
	assert.False(t, form.Fields[1].Input.Focused())
	assert.False(t, form.Fields[2].Input.Focused())
}

func TestFormValidate(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test validate with no fields
	valid := form.Validate()
	assert.True(t, valid)

	// Test validate with valid fields
	field1 := NewFormField("Username", "Enter username", "testuser", true)
	field2 := NewFormField("Password", "Enter password", "password", true)
	form.AddField(field1)
	form.AddField(field2)

	valid = form.Validate()
	assert.True(t, valid)

	// Test validate with invalid fields
	form.Fields[0].Input.SetValue("")
	valid = form.Validate()
	assert.False(t, valid)
	assert.Equal(t, "This field is required", form.Fields[0].Error)
}

func TestGetValues(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	// Test get values with no fields
	values := form.GetValues()
	assert.Equal(t, 0, len(values))

	// Test get values with fields
	field1 := NewFormField("Username", "Enter username", "testuser", true)
	field2 := NewFormField("Password", "Enter password", "password", true)
	form.AddField(field1)
	form.AddField(field2)

	values = form.GetValues()
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "testuser", values["Username"])
	assert.Equal(t, "password", values["Password"])
}

func TestFormUpdate(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	field1 := NewFormField("Username", "Enter username", "", true)
	field2 := NewFormField("Password", "Enter password", "", true)
	form.AddField(field1)
	form.AddField(field2)

	// Test tab key
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedForm, _ := form.Update(msg)

	assert.Equal(t, 1, updatedForm.FocusIndex)
	assert.False(t, updatedForm.Fields[0].Input.Focused())
	assert.True(t, updatedForm.Fields[1].Input.Focused())

	// Test shift+tab key
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	updatedForm, _ = updatedForm.Update(msg)

	assert.Equal(t, 0, updatedForm.FocusIndex)
	assert.True(t, updatedForm.Fields[0].Input.Focused())
	assert.False(t, updatedForm.Fields[1].Input.Focused())

	// Test character input
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedForm, _ = updatedForm.Update(msg)

	assert.Equal(t, "a", updatedForm.Fields[0].Input.Value())
}

func TestFormView(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	form := NewForm(styles, 60)

	field1 := NewFormField("Username", "Enter username", "", true)
	field1.SetDescription("Your username for login")
	field2 := NewFormField("Password", "Enter password", "", true)
	field2.Error = "This field is required"

	form.AddField(field1)
	form.AddField(field2)

	view := form.View()

	// Check that the view contains all the expected elements
	assert.Contains(t, view, "Username")
	assert.Contains(t, view, "*") // Required field marker
	assert.Contains(t, view, "Your username for login")
	assert.Contains(t, view, "Password")
	assert.Contains(t, view, "This field is required")
	assert.Contains(t, view, "Submit")
	assert.Contains(t, view, "Cancel")
}
