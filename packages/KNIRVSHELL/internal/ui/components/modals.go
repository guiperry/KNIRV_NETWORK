package components

import (
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Modal represents a modal dialog
type Modal struct {
	Title       string
	Content     string
	Width       int
	Height      int
	styles      ui.Styles
	buttons     []string
	selectedBtn int
	visible     bool
}

// NewModal creates a new modal
func NewModal(styles ui.Styles, title, content string, width, height int) Modal {
	return Modal{
		Title:       title,
		Content:     content,
		Width:       width,
		Height:      height,
		styles:      styles,
		buttons:     []string{"OK"},
		selectedBtn: 0,
		visible:     false,
	}
}

// SetButtons sets the modal buttons
func (m *Modal) SetButtons(buttons []string) {
	m.buttons = buttons
	m.selectedBtn = 0
}

// Show shows the modal
func (m *Modal) Show() {
	m.visible = true
}

// Hide hides the modal
func (m *Modal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is visible
func (m *Modal) IsVisible() bool {
	return m.visible
}

// NextButton selects the next button
func (m *Modal) NextButton() {
	m.selectedBtn = (m.selectedBtn + 1) % len(m.buttons)
}

// PrevButton selects the previous button
func (m *Modal) PrevButton() {
	m.selectedBtn = (m.selectedBtn - 1 + len(m.buttons)) % len(m.buttons)
}

// SelectedButton returns the selected button
func (m *Modal) SelectedButton() string {
	if len(m.buttons) == 0 {
		return ""
	}
	return m.buttons[m.selectedBtn]
}

// Update handles user input
func (m *Modal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.PrevButton()
		case "right":
			m.NextButton()
		case "enter":
			m.Hide()
			return *m, nil
		case "esc":
			m.Hide()
			return *m, nil
		}
	}

	return *m, nil
}

// View renders the modal
func (m Modal) View() string {
	if !m.visible {
		return ""
	}

	// Calculate dimensions
	width := m.Width
	if width < 20 {
		width = 20
	}

	// Create the modal content
	var sb strings.Builder

	// Title
	titleStyle := m.styles.DialogTitle.Copy().Width(width - 4)
	sb.WriteString(titleStyle.Render(m.Title))
	sb.WriteString("\n\n")

	// Content
	contentStyle := m.styles.DialogContent.Copy().Width(width - 4)
	sb.WriteString(contentStyle.Render(m.Content))
	sb.WriteString("\n\n")

	// Buttons
	for i, button := range m.buttons {
		var btnStyle lipgloss.Style
		if i == m.selectedBtn {
			btnStyle = m.styles.ButtonPrimary
		} else {
			btnStyle = m.styles.Button
		}
		sb.WriteString(btnStyle.Render(button))
		sb.WriteString(" ")
	}

	// Apply dialog style
	return m.styles.Dialog.Copy().Width(width).Render(sb.String())
}

// ConfirmModal represents a confirmation modal
type ConfirmModal struct {
	Modal
	onConfirm func()
	onCancel  func()
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(styles ui.Styles, title, content string, width, height int) ConfirmModal {
	modal := NewModal(styles, title, content, width, height)
	modal.SetButtons([]string{"Cancel", "Confirm"})

	return ConfirmModal{
		Modal:     modal,
		onConfirm: func() {},
		onCancel:  func() {},
	}
}

// SetCallbacks sets the confirmation callbacks
func (m *ConfirmModal) SetCallbacks(onConfirm, onCancel func()) {
	m.onConfirm = onConfirm
	m.onCancel = onCancel
}

// Update handles user input
func (m *ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.Hide()
			if m.selectedBtn == 1 {
				m.onConfirm()
			} else {
				m.onCancel()
			}
			return *m, nil
		case "esc":
			m.Hide()
			m.onCancel()
			return *m, nil
		}
	}

	modal, cmd := m.Modal.Update(msg)
	m.Modal = modal
	return *m, cmd
}

// AlertModal represents an alert modal
type AlertModal struct {
	Modal
	onClose func()
}

// NewAlertModal creates a new alert modal
func NewAlertModal(styles ui.Styles, title, content string, width, height int) AlertModal {
	modal := NewModal(styles, title, content, width, height)
	modal.SetButtons([]string{"OK"})

	return AlertModal{
		Modal:   modal,
		onClose: func() {},
	}
}

// SetOnClose sets the close callback
func (m *AlertModal) SetOnClose(onClose func()) {
	m.onClose = onClose
}

// Update handles user input
func (m *AlertModal) Update(msg tea.Msg) (AlertModal, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.Hide()
			m.onClose()
			return *m, nil
		}
	}

	modal, cmd := m.Modal.Update(msg)
	m.Modal = modal
	return *m, cmd
}

// InputModal represents an input modal
type InputModal struct {
	Modal
	form     Form
	onSubmit func(string)
	onCancel func()
}

// NewInputModal creates a new input modal
func NewInputModal(styles ui.Styles, title, prompt, placeholder string, width, height int) InputModal {
	modal := NewModal(styles, title, "", width, height)
	modal.SetButtons([]string{"Cancel", "Submit"})

	form := NewForm(styles, width-4)
	field := NewFormField("", placeholder, "", true)
	field.SetDescription(prompt)
	form.AddField(field)

	return InputModal{
		Modal:    modal,
		form:     form,
		onSubmit: func(string) {},
		onCancel: func() {},
	}
}

// SetCallbacks sets the input callbacks
func (m *InputModal) SetCallbacks(onSubmit func(string), onCancel func()) {
	m.onSubmit = onSubmit
	m.onCancel = onCancel
}

// Update handles user input
func (m *InputModal) Update(msg tea.Msg) (InputModal, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.form.Validate() {
				m.Hide()
				m.onSubmit(m.form.Fields[0].Input.Value())
				return *m, nil
			}
		case "esc":
			m.Hide()
			m.onCancel()
			return *m, nil
		}
	}

	form, cmd := m.form.Update(msg)
	m.form = form
	return *m, cmd
}

// View renders the input modal
func (m InputModal) View() string {
	if !m.visible {
		return ""
	}

	// Calculate dimensions
	width := m.Width
	if width < 20 {
		width = 20
	}

	// Create the modal content
	var sb strings.Builder

	// Title
	titleStyle := m.styles.DialogTitle.Copy().Width(width - 4)
	sb.WriteString(titleStyle.Render(m.Title))
	sb.WriteString("\n\n")

	// Form
	sb.WriteString(m.form.View())

	// Apply dialog style
	return m.styles.Dialog.Copy().Width(width).Render(sb.String())
}
