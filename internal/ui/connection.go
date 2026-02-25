package ui

import (
	"fmt"
	"log"
	"strings"

	"ssh-scp/internal/config"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConnectMsg is sent when the user initiates a connection.
type ConnectMsg struct {
	Conn config.Connection
}

type connectionField int

const (
	fieldHost connectionField = iota
	fieldPort
	fieldUser
	fieldKey
	fieldJump
	fieldCount
)

// connectionPane tracks which section has keyboard focus on the connection screen.
type connectionPane int

const (
	paneForm connectionPane = iota
	paneList
)

// ConnectionModel is the connection screen.
type ConnectionModel struct {
	inputs     []textinput.Model
	focused    connectionField
	connList   list.Model
	hasItems   bool
	activePane connectionPane
	cfg        *config.Config
	sshHosts   []config.SSHHost
	width      int
	height     int
	err        string
}

// connItem is a list item representing either a recent connection or an SSH config host.
type connItem struct {
	conn   config.Connection
	source string // "recent" or "ssh-config"
}

func (c connItem) Title() string {
	title := fmt.Sprintf("%s@%s:%s", c.conn.Username, c.conn.Host, c.conn.Port)
	if c.conn.Username == "" {
		title = fmt.Sprintf("%s:%s", c.conn.Host, c.conn.Port)
	}
	return title
}
func (c connItem) Description() string {
	tag := "recent"
	if c.source == "ssh-config" {
		tag = "~/.ssh/config"
	}
	name := c.conn.Name
	if name == "" {
		name = c.conn.Host
	}
	return fmt.Sprintf("[%s] %s", tag, name)
}
func (c connItem) FilterValue() string {
	return c.conn.Host + " " + c.conn.Name
}

// NewConnectionModel creates a new connection screen model.
func NewConnectionModel(cfg *config.Config) ConnectionModel {
	return NewConnectionModelWithSSH(cfg, config.LoadSSHConfig())
}

// SetError sets an error message to display on the connection screen.
func (m *ConnectionModel) SetError(msg string) {
	m.err = msg
}

// NewConnectionModelWithSSH creates a connection screen with explicit SSH config hosts.
func NewConnectionModelWithSSH(cfg *config.Config, sshHosts []config.SSHHost) ConnectionModel {
	inputs := make([]textinput.Model, fieldCount)
	labels := []string{"Host", "Port", "Username", "SSH Key Path", "Jump Host"}
	for i := range inputs {
		t := textinput.New()
		t.Placeholder = labels[i]
		t.CharLimit = 256
		inputs[i] = t
	}
	inputs[fieldPort].SetValue("22")
	inputs[fieldJump].Placeholder = "user@host:port (optional)"
	inputs[fieldHost].Focus()

	// Build combined list: SSH config hosts first, then recent connections.
	var items []list.Item
	for _, h := range sshHosts {
		items = append(items, connItem{conn: h.ToConnection(), source: "ssh-config"})
	}
	for _, c := range cfg.RecentConnections {
		items = append(items, connItem{conn: c, source: "recent"})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFFFF")).
		BorderForeground(lipgloss.Color("#7D56F4"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#AAAAAA")).
		BorderForeground(lipgloss.Color("#7D56F4"))

	l := list.New(items, delegate, 44, 14)
	l.Title = "Connections"
	l.SetShowStatusBar(false)

	return ConnectionModel{
		inputs:     inputs,
		focused:    fieldHost,
		connList:   l,
		hasItems:   len(items) > 0,
		activePane: paneForm,
		cfg:        cfg,
		sshHosts:   sshHosts,
	}
}

func (m ConnectionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConnectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.connList.SetWidth(msg.Width / 3)
		listH := msg.Height - 6
		if listH < 4 {
			listH = 4
		}
		m.connList.SetHeight(listH)
		return m, nil
	case tea.KeyMsg:
		log.Printf("[ConnectionModel] key: type=%d string=%q runes=%v alt=%v pane=%d focused=%d",
			msg.Type, msg.String(), msg.Runes, msg.Alt, m.activePane, m.focused)

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			log.Printf("[ConnectionModel] Enter pressed, pane=%d host=%q user=%q",
				m.activePane, m.inputs[fieldHost].Value(), m.inputs[fieldUser].Value())
			if m.activePane == paneList {
				// Populate form from selected list item and connect.
				if item, ok := m.connList.SelectedItem().(connItem); ok {
					log.Printf("[ConnectionModel] list item selected: %s@%s", item.conn.Username, item.conn.Host)
					m.fillForm(item.conn)
					m.activePane = paneForm
					m.inputs[m.focused].Focus()
					cmd := m.submitForm()
					log.Printf("[ConnectionModel] submitForm returned cmd=%v err=%q", cmd != nil, m.err)
					return m, cmd
				}
				return m, nil
			}
			cmd := m.submitForm()
			log.Printf("[ConnectionModel] submitForm returned cmd=%v err=%q", cmd != nil, m.err)
			return m, cmd

		case tea.KeyTab, tea.KeyDown:
			if m.activePane == paneForm {
				m.inputs[m.focused].Blur()
				m.focused = (m.focused + 1) % fieldCount
				m.inputs[m.focused].Focus()
				return m, nil
			}
		case tea.KeyShiftTab, tea.KeyUp:
			if m.activePane == paneForm {
				m.inputs[m.focused].Blur()
				m.focused = (m.focused - 1 + fieldCount) % fieldCount
				m.inputs[m.focused].Focus()
				return m, nil
			}

		case tea.KeyCtrlRight:
			if m.hasItems && m.activePane == paneForm {
				m.inputs[m.focused].Blur()
				m.activePane = paneList
			}
			return m, nil

		case tea.KeyCtrlLeft:
			if m.activePane == paneList {
				m.activePane = paneForm
				m.inputs[m.focused].Focus()
			}
			return m, nil
		}
	}

	if m.activePane == paneList {
		var cmd tea.Cmd
		m.connList, cmd = m.connList.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// fillForm populates the input fields from a connection.
func (m *ConnectionModel) fillForm(c config.Connection) {
	m.inputs[fieldHost].SetValue(c.Host)
	m.inputs[fieldPort].SetValue(c.Port)
	m.inputs[fieldUser].SetValue(c.Username)
	m.inputs[fieldKey].SetValue(c.KeyPath)
	m.inputs[fieldJump].SetValue(c.ProxyJump)
}

// submitForm validates and submits the form.
func (m *ConnectionModel) submitForm() tea.Cmd {
	host := m.inputs[fieldHost].Value()
	port := m.inputs[fieldPort].Value()
	user := m.inputs[fieldUser].Value()
	key := m.inputs[fieldKey].Value()
	jump := m.inputs[fieldJump].Value()

	if host == "" || user == "" {
		m.err = "Host and username are required"
		return nil
	}
	if port == "" {
		port = "22"
	}

	conn := config.Connection{
		Name:      fmt.Sprintf("%s@%s", user, host),
		Host:      host,
		Port:      port,
		Username:  user,
		KeyPath:   key,
		ProxyJump: jump,
	}
	m.cfg.AddRecent(conn)
	if err := config.Save(m.cfg); err != nil {
		m.err = "Failed to save config: " + err.Error()
	}

	return func() tea.Msg { return ConnectMsg{Conn: conn} }
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Width(12)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(1, 2).
			Width(50)

	focusedInputBoxStyle = inputBoxStyle.
				BorderForeground(lipgloss.Color("#7D56F4"))

	dimBoxStyle = inputBoxStyle.
			BorderForeground(lipgloss.Color("#333333"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
)

func (m ConnectionModel) View() string {
	labels := []string{"Host:", "Port:", "Username:", "SSH Key:", "Jump Host:"}
	var rows []string
	for i, inp := range m.inputs {
		label := labelStyle.Render(labels[i])
		row := lipgloss.JoinHorizontal(lipgloss.Center, label, inp.View())
		rows = append(rows, row)
	}

	form := strings.Join(rows, "\n")
	var boxStyle lipgloss.Style
	if m.activePane == paneForm {
		boxStyle = focusedInputBoxStyle
	} else {
		boxStyle = dimBoxStyle
	}
	box := boxStyle.Render(form)

	title := titleStyle.Render("SSH TUI - New Connection")
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(
		"Tab/↑↓: navigate • Enter: connect • Ctrl + ←/→: switch pane • Ctrl + C: quit",
	)

	var errMsg string
	if m.err != "" {
		errMsg = "\n" + errorStyle.Render("⚠  "+m.err)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", box, "", hint+errMsg)

	if m.hasItems {
		listView := m.connList.View()
		var listBoxStyle lipgloss.Style
		if m.activePane == paneList {
			listBoxStyle = focusedInputBoxStyle
		} else {
			listBoxStyle = dimBoxStyle
		}
		listBox := listBoxStyle.Render(listView)
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, "  ", listBox)
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
