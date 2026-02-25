package ui

import (
	"bytes"
	"io"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"
	sshclient "sshtui/internal/ssh"
)

// TerminalOutputMsg carries new output from the SSH session.
type TerminalOutputMsg struct {
	Data []byte
}

// TerminalModel manages an interactive SSH terminal.
type TerminalModel struct {
	client  *sshclient.Client
	session *ssh.Session
	stdin   io.WriteCloser
	buf     bytes.Buffer
	mu      sync.Mutex
	width   int
	height  int
	active  bool
	err     string
	program *tea.Program
}

// terminalWriter implements io.Writer and sends output as tea messages.
type terminalWriter struct {
	program *tea.Program
}

func (tw *terminalWriter) Write(p []byte) (int, error) {
	data := make([]byte, len(p))
	copy(data, p)
	tw.program.Send(TerminalOutputMsg{Data: data})
	return len(p), nil
}

// NewTerminalModel creates a new terminal model.
func NewTerminalModel(client *sshclient.Client) *TerminalModel {
	return &TerminalModel{
		client: client,
		active: true,
	}
}

// SetProgram sets the bubbletea program for sending messages.
func (m *TerminalModel) SetProgram(p *tea.Program) {
	m.program = p
}

// StartSession starts the SSH terminal session.
func (m *TerminalModel) StartSession() error {
	session, err := m.client.NewSession()
	if err != nil {
		return err
	}
	m.session = session

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return err
	}
	m.stdin = stdinPipe

	var tw io.Writer
	if m.program != nil {
		tw = &terminalWriter{program: m.program}
	} else {
		tw = &m.buf
	}

	if err := m.client.StartTerminal(session, nil, tw, tw); err != nil {
		session.Close()
		return err
	}

	go func() {
		session.Wait()
		if m.program != nil {
			m.program.Send(TerminalOutputMsg{Data: []byte("\r\n[Session closed]\r\n")})
		}
	}()

	return nil
}

// Write sends data to the SSH session stdin.
func (m *TerminalModel) Write(data []byte) error {
	if m.stdin == nil {
		return nil
	}
	_, err := m.stdin.Write(data)
	return err
}

// Resize resizes the terminal PTY.
func (m *TerminalModel) Resize(width, height int) {
	m.width = width
	m.height = height
	if m.session != nil {
		_ = m.client.ResizePty(m.session, width, height)
	}
}

// Close closes the terminal session.
func (m *TerminalModel) Close() {
	if m.stdin != nil {
		m.stdin.Close()
	}
	if m.session != nil {
		m.session.Close()
	}
}

// AppendOutput appends terminal output to the buffer.
func (m *TerminalModel) AppendOutput(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Write(data)
	if m.buf.Len() > 100*1024 {
		b := m.buf.Bytes()
		m.buf.Reset()
		m.buf.Write(b[len(b)-50*1024:])
	}
}

// BufferedOutput returns the current terminal output.
func (m *TerminalModel) BufferedOutput() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

var (
	terminalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(0, 1)

	activeTerminalStyle = terminalStyle.
		BorderForeground(lipgloss.Color("#7D56F4"))
)

// RenderTerminal returns the terminal view string.
func (m *TerminalModel) RenderTerminal(active bool, width, height int) string {
	style := terminalStyle
	if active {
		style = activeTerminalStyle
	}

	output := m.BufferedOutput()
	lines := splitOutputLines(output, height-4)
	content := joinLines(lines)

	return style.Width(width).Height(height).Render(content)
}

func splitOutputLines(s string, n int) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	total := 0
	for _, l := range lines {
		total += len(l) + 1
	}
	b := make([]byte, 0, total)
	for i, l := range lines {
		b = append(b, l...)
		if i < len(lines)-1 {
			b = append(b, '\n')
		}
	}
	return string(b)
}
