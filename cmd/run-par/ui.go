package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	commands        []*Command
	selectedIndex   int
	searchMode      bool
	searchQuery     string
	width           int
	height          int
	updates         chan CommandUpdate
	quitting        bool
	continueOnError bool
}

func newModel(commands []*Command, updates chan CommandUpdate, continueOnError bool) model {
	return model{
		commands:        commands,
		selectedIndex:   0,
		searchMode:      false,
		searchQuery:     "",
		updates:         updates,
		continueOnError: continueOnError,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForUpdate(m.updates),
	)
}

func waitForUpdate(updates chan CommandUpdate) tea.Cmd {
	return func() tea.Msg {
		return <-updates
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchMode(msg)
		}
		return m.handleNormalMode(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case CommandUpdate:
		// Update the command
		if msg.Index >= 0 && msg.Index < len(m.commands) {
			cmd := m.commands[msg.Index]
			cmd.mu.Lock()
			switch msg.Type {
			case UpdateStatus:
				cmd.Status = msg.Status
			case UpdateOutput:
				// Already updated in runner
			case UpdateComplete:
				cmd.Status = msg.Status
				cmd.ExitCode = msg.ExitCode
			}
			cmd.mu.Unlock()
		}

		// If not continue-on-error mode and a command failed, exit immediately
		if !m.continueOnError && msg.Type == UpdateComplete && msg.Status == StatusFailed {
			m.quitting = true
			return m, tea.Quit
		}

		// Check if all commands are complete
		if m.allCommandsComplete() {
			m.quitting = true
			return m, tea.Quit
		}

		return m, waitForUpdate(m.updates)
	}

	return m, nil
}

func (m model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}

	case "down", "j":
		if m.selectedIndex < len(m.commands)-1 {
			m.selectedIndex++
		}

	case "/":
		m.searchMode = true
		m.searchQuery = ""
	}

	return m, nil
}

func (m model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchQuery = ""

	case "enter":
		m.searchMode = false

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Calculate dimensions
	sidebarWidth := m.width * 4 / 15
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	logWidth := m.width - sidebarWidth - 2

	// Render sidebar
	sidebar := m.renderSidebar(sidebarWidth, m.height-2)

	// Render log panel
	logPanel := m.renderLogPanel(logWidth, m.height-2)

	// Combine sidebar and log panel
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, logPanel)

	// Render status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

func (m model) renderSidebar(width, height int) string {
	var items []string

	for i, cmd := range m.commands {
		cmd.mu.RLock()
		status := cmd.Status
		cmdName := cmd.FullCommand
		if len(cmdName) > width-10 {
			cmdName = cmdName[:width-13] + "..."
		}
		cmd.mu.RUnlock()

		icon := m.getStatusIcon(status)
		item := fmt.Sprintf("%s %s", icon, cmdName)

		style := lipgloss.NewStyle().Width(width - 2).Padding(0, 1)
		if i == m.selectedIndex {
			style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
		}

		items = append(items, style.Render(item))
	}

	content := strings.Join(items, "\n")

	sidebarStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return sidebarStyle.Render(content)
}

func (m model) renderLogPanel(width, height int) string {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.commands) {
		return ""
	}

	cmd := m.commands[m.selectedIndex]
	cmd.mu.RLock()
	output := cmd.Output
	cmd.mu.RUnlock()

	// Filter output if search mode
	var displayLines []string
	if m.searchQuery != "" {
		for _, line := range output {
			if strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
				displayLines = append(displayLines, line)
			}
		}
	} else {
		displayLines = output
	}

	// Show last lines that fit in the panel
	maxLines := height - 2
	startIdx := 0
	if len(displayLines) > maxLines {
		startIdx = len(displayLines) - maxLines
	}

	content := strings.Join(displayLines[startIdx:], "\n")

	logStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	return logStyle.Render(content)
}

func (m model) renderStatusBar() string {
	var status string
	if m.searchMode {
		status = fmt.Sprintf("Search: %s", m.searchQuery)
	} else {
		status = "q: quit | ↑/↓: navigate | /: search"
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	return style.Render(status)
}

func (m model) getStatusIcon(status CommandStatus) string {
	switch status {
	case StatusPending:
		return "○"
	case StatusRunning:
		return "◐"
	case StatusSuccess:
		return "●"
	case StatusFailed:
		return "✗"
	default:
		return "?"
	}
}

func (m model) allCommandsComplete() bool {
	for _, cmd := range m.commands {
		cmd.mu.RLock()
		status := cmd.Status
		cmd.mu.RUnlock()

		if status != StatusSuccess && status != StatusFailed {
			return false
		}
	}
	return true
}
