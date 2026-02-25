package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var helpContent = `
  Key Bindings

  Ctrl+←/→  Switch between local and remote panels
  Tab       Switch between local and remote panels
  Ctrl+U    Upload selected local file to remote
  Ctrl+D    Download selected remote file to local
  Ctrl+T    Switch to next tab
  Ctrl+N    New connection tab
  Ctrl+W    Close current tab
  Enter     Navigate into directory
  Backspace Go up one directory
  T         Transfer selected file (contextual)
  ?         Toggle this help overlay
  Ctrl+C    Quit
`

var helpStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#7D56F4")).
	Padding(1, 3).
	Bold(false)

// RenderHelp returns the help overlay view.
func RenderHelp(width, height int) string {
	box := helpStyle.Render(helpContent)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
