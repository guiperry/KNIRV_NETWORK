package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiperry/KNIRVCHAIN-CLI/ui"
)

// FormField represents a form field
type FormField struct {
	Label       string
	Input       textinput.Model
	Description string
	Required    bool
	Error       string
	Validator   func(string) error
}

// NewFormField creates a new form field
func NewFormField(label, placeholder, value string, required bool) FormField {
	input := textinput.New()
	input.Placeholder = placeholder
	input.SetValue(value)
	input.Width = 40

	return FormField{
		Label:     label,
		Input:     input,
		Required:  required,
		Error:     "",
		Validator: nil,
	}
}

// NewPasswordField creates a new password field
func NewPasswordField(label, placeholder, value string, required bool) FormField {
	field := NewFormField(label, placeholder, value, required)
	field.Input.EchoMode = textinput.EchoPassword
	field.Input.EchoCharacter = '•'
	return field
}

// SetValidator sets the field validator
func (f *FormField) SetValidator(validator func(string) error) {
	f.Validator = validator
}

// SetDescription sets the field description
func (f *FormField) SetDescription(description string) {
	f.Description = description
}

// Validate validates the field
func (f *FormField) Validate() bool {
	f.Error = ""

	// Check if required
	if f.Required && f.Input.Value() == "" {
		f.Error = "This field is required"
		return false
	}

	// Run validator if provided
	if f.Validator != nil {
		if err := f.Validator(f.Input.Value()); err != nil {
			f.Error = err.Error()
			return false
		}
	}

	return true
}

// Form represents a form
type Form struct {
	Fields      []FormField
	FocusIndex  int
	styles      ui.Styles
	width       int
	submitLabel string
	cancelLabel string
}

// NewForm creates a new form
func NewForm(styles ui.Styles, width int) Form {
	return Form{
		Fields:      []FormField{},
		FocusIndex:  0,
		styles:      styles,
		width:       width,
		submitLabel: "Submit",
		cancelLabel: "Cancel",
	}
}

// AddField adds a field to the form
func (f *Form) AddField(field FormField) {
	f.Fields = append(f.Fields, field)
	if len(f.Fields) == 1 {
		f.Fields[0].Input.Focus()
	}
}

// SetSubmitLabel sets the submit button label
func (f *Form) SetSubmitLabel(label string) {
	f.submitLabel = label
}

// SetCancelLabel sets the cancel button label
func (f *Form) SetCancelLabel(label string) {
	f.cancelLabel = label
}

// Focus focuses the form
func (f *Form) Focus() {
	if len(f.Fields) > 0 {
		f.FocusIndex = 0
		f.Fields[0].Input.Focus()
	}
}

// Blur blurs the form
func (f *Form) Blur() {
	if len(f.Fields) > 0 {
		f.Fields[f.FocusIndex].Input.Blur()
	}
}

// NextField focuses the next field
func (f *Form) NextField() {
	if len(f.Fields) == 0 {
		return
	}

	f.Fields[f.FocusIndex].Input.Blur()
	f.FocusIndex = (f.FocusIndex + 1) % len(f.Fields)
	f.Fields[f.FocusIndex].Input.Focus()
}

// PrevField focuses the previous field
func (f *Form) PrevField() {
	if len(f.Fields) == 0 {
		return
	}

	f.Fields[f.FocusIndex].Input.Blur()
	f.FocusIndex = (f.FocusIndex - 1 + len(f.Fields)) % len(f.Fields)
	f.Fields[f.FocusIndex].Input.Focus()
}

// Validate validates the form
func (f *Form) Validate() bool {
	valid := true
	for i := range f.Fields {
		if !f.Fields[i].Validate() {
			valid = false
		}
	}
	return valid
}

// GetValues returns the form values
func (f *Form) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range f.Fields {
		values[field.Label] = field.Input.Value()
	}
	return values
}

// Update handles user input
func (f *Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			f.NextField()
		case "shift+tab":
			f.PrevField()
		}
	}

	// Update the focused field
	if len(f.Fields) > 0 {
		newInput, cmd := f.Fields[f.FocusIndex].Input.Update(msg)
		f.Fields[f.FocusIndex].Input = newInput
		cmds = append(cmds, cmd)
	}

	return *f, tea.Batch(cmds...)
}

// View renders the form
func (f Form) View() string {
	var sb strings.Builder

	for i, field := range f.Fields {
		// Label
		labelStyle := f.styles.Label
		if i == f.FocusIndex {
			labelStyle = labelStyle.Copy().Foreground(f.styles.Theme.Primary)
		}
		sb.WriteString(labelStyle.Render(field.Label))
		if field.Required {
			sb.WriteString(f.styles.Error.Render(" *"))
		}
		sb.WriteString("\n")

		// Description
		if field.Description != "" {
			sb.WriteString(f.styles.Subtle.Render(field.Description))
			sb.WriteString("\n")
		}

		// Input
		var inputStyle lipgloss.Style
		if field.Error != "" {
			inputStyle = f.styles.InputError
		} else if i == f.FocusIndex {
			inputStyle = f.styles.InputFocused
		} else {
			inputStyle = f.styles.Input
		}
		sb.WriteString(inputStyle.Render(field.Input.View()))
		sb.WriteString("\n")

		// Error
		if field.Error != "" {
			sb.WriteString(f.styles.Error.Render(field.Error))
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	// Buttons
	submitStyle := f.styles.ButtonPrimary
	cancelStyle := f.styles.Button
	sb.WriteString(submitStyle.Render(f.submitLabel))
	sb.WriteString(" ")
	sb.WriteString(cancelStyle.Render(f.cancelLabel))

	return sb.String()
}

// SelectOption represents a select option
type SelectOption struct {
	Value string
	Label string
}

// Select represents a select field
type Select struct {
	Label       string
	Options     []SelectOption
	Selected    int
	Description string
	Required    bool
	Error       string
	Focused     bool
	styles      ui.Styles
}

// NewSelect creates a new select field
func NewSelect(styles ui.Styles, label string, options []SelectOption, required bool) Select {
	return Select{
		Label:    label,
		Options:  options,
		Selected: 0,
		Required: required,
		Error:    "",
		Focused:  false,
		styles:   styles,
	}
}

// SetDescription sets the select description
func (s *Select) SetDescription(description string) {
	s.Description = description
}

// Focus focuses the select
func (s *Select) Focus() {
	s.Focused = true
}

// Blur blurs the select
func (s *Select) Blur() {
	s.Focused = false
}

// Next selects the next option
func (s *Select) Next() {
	s.Selected = (s.Selected + 1) % len(s.Options)
}

// Prev selects the previous option
func (s *Select) Prev() {
	s.Selected = (s.Selected - 1 + len(s.Options)) % len(s.Options)
}

// GetValue returns the selected value
func (s *Select) GetValue() string {
	if len(s.Options) == 0 {
		return ""
	}
	return s.Options[s.Selected].Value
}

// GetLabel returns the selected label
func (s *Select) GetLabel() string {
	if len(s.Options) == 0 {
		return ""
	}
	return s.Options[s.Selected].Label
}

// SetValue sets the selected value
func (s *Select) SetValue(value string) {
	for i, option := range s.Options {
		if option.Value == value {
			s.Selected = i
			return
		}
	}
}

// Validate validates the select
func (s *Select) Validate() bool {
	s.Error = ""

	// Check if required
	if s.Required && len(s.Options) == 0 {
		s.Error = "This field is required"
		return false
	}

	return true
}

// Update handles user input
func (s *Select) Update(msg tea.Msg) (Select, tea.Cmd) {
	if !s.Focused {
		return *s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "left":
			s.Prev()
		case "down", "right":
			s.Next()
		}
	}

	return *s, nil
}

// View renders the select
func (s Select) View() string {
	var sb strings.Builder

	// Label
	labelStyle := s.styles.Label
	if s.Focused {
		labelStyle = labelStyle.Copy().Foreground(s.styles.Theme.Primary)
	}
	sb.WriteString(labelStyle.Render(s.Label))
	if s.Required {
		sb.WriteString(s.styles.Error.Render(" *"))
	}
	sb.WriteString("\n")

	// Description
	if s.Description != "" {
		sb.WriteString(s.styles.Subtle.Render(s.Description))
		sb.WriteString("\n")
	}

	// Options
	for i, option := range s.Options {
		if i == s.Selected {
			if s.Focused {
				sb.WriteString(s.styles.ListItemSelected.Render(fmt.Sprintf("● %s", option.Label)))
			} else {
				sb.WriteString(s.styles.Bold.Render(fmt.Sprintf("● %s", option.Label)))
			}
		} else {
			sb.WriteString(s.styles.ListItem.Render(fmt.Sprintf("○ %s", option.Label)))
		}
		sb.WriteString("\n")
	}

	// Error
	if s.Error != "" {
		sb.WriteString(s.styles.Error.Render(s.Error))
		sb.WriteString("\n")
	}

	return sb.String()
}
