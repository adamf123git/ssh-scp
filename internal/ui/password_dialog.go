package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PasswordRequestMsg is sent when SSH authentication needs user input
// (e.g. a password or verification code). The prompt originates from the
// server's keyboard-interactive challenge or from the SSH password callback.
type PasswordRequestMsg struct {
	Prompt   string
	Hostname string
	Username string
}

// PasswordResponseMsg is sent when the user submits or cancels the dialog.
type PasswordResponseMsg struct {
	Password  string
	Cancelled bool
}

// PasswordDialogModel manages the interactive password/passphrase dialog
// that is overlaid on other screens when the server requests credentials.
type PasswordDialogModel struct {
	prompt  string
	input   textinput.Model
	visible bool
}

// NewPasswordDialogModel creates a new, initially hidden password dialog.
func NewPasswordDialogModel() PasswordDialogModel {
	t := textinput.New()
	t.EchoMode = textinput.EchoPassword
	t.EchoCharacter = '•'
	t.CharLimit = 256
	t.Width = 40
	return PasswordDialogModel{input: t}
}

// Show makes the dialog visible with the given prompt and focuses the input.
func (m *PasswordDialogModel) Show(prompt string) {
	m.prompt = prompt
	m.input.SetValue("")
	m.input.Focus()
	m.visible = true
}

// Hide closes the dialog.
func (m *PasswordDialogModel) Hide() {
	m.visible = false
	m.input.Blur()
}

// Visible reports whether the dialog is currently shown.
func (m PasswordDialogModel) Visible() bool {
	return m.visible
}

// Update processes key events while the dialog is visible.
func (m PasswordDialogModel) Update(msg tea.Msg) (PasswordDialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			password := m.input.Value()
			m.visible = false
			m.input.Blur()
			return m, func() tea.Msg {
				return PasswordResponseMsg{Password: password}
			}
		case tea.KeyEsc:
			m.visible = false
			m.input.Blur()
			return m, func() tea.Msg {
				return PasswordResponseMsg{Cancelled: true}
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the dialog as a centered overlay box.
func (m PasswordDialogModel) View(width, height int) string {
	if !m.visible {
		return ""
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		dialogPromptStyle.Render(m.prompt),
		"",
		m.input.View(),
		"",
		dialogHintStyle.Render("Enter: submit • Esc: cancel"),
	)

	box := dialogBoxStyle.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 3).
			Width(54)

	dialogPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	dialogHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true)
)
