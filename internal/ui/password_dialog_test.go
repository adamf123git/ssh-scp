package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewPasswordDialogModel(t *testing.T) {
	m := NewPasswordDialogModel()
	if m.Visible() {
		t.Error("new dialog should be hidden")
	}
}

func TestPasswordDialogShowHide(t *testing.T) {
	m := NewPasswordDialogModel()
	m.Show("Enter password:")
	if !m.Visible() {
		t.Error("dialog should be visible after Show")
	}
	if m.prompt != "Enter password:" {
		t.Errorf("prompt = %q", m.prompt)
	}
	m.Hide()
	if m.Visible() {
		t.Error("dialog should be hidden after Hide")
	}
}

func TestPasswordDialogEnterSubmits(t *testing.T) {
	m := NewPasswordDialogModel()
	m.Show("Password:")

	// Type some characters
	for _, r := range "secret" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Enter
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Visible() {
		t.Error("dialog should be hidden after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}
	msg := cmd()
	resp, ok := msg.(PasswordResponseMsg)
	if !ok {
		t.Fatalf("expected PasswordResponseMsg, got %T", msg)
	}
	if resp.Cancelled {
		t.Error("should not be cancelled")
	}
	if resp.Password != "secret" {
		t.Errorf("password = %q, want %q", resp.Password, "secret")
	}
}

func TestPasswordDialogEscCancels(t *testing.T) {
	m := NewPasswordDialogModel()
	m.Show("Password:")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.Visible() {
		t.Error("dialog should be hidden after Esc")
	}
	if cmd == nil {
		t.Fatal("Esc should produce a command")
	}
	msg := cmd()
	resp, ok := msg.(PasswordResponseMsg)
	if !ok {
		t.Fatalf("expected PasswordResponseMsg, got %T", msg)
	}
	if !resp.Cancelled {
		t.Error("should be cancelled")
	}
}

func TestPasswordDialogViewHidden(t *testing.T) {
	m := NewPasswordDialogModel()
	v := m.View(80, 40)
	if v != "" {
		t.Errorf("hidden dialog view should be empty, got %q", v)
	}
}

func TestPasswordDialogViewVisible(t *testing.T) {
	m := NewPasswordDialogModel()
	m.Show("Enter code:")
	v := m.View(80, 40)
	if v == "" {
		t.Error("visible dialog view should not be empty")
	}
}
