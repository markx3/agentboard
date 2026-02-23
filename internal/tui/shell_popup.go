package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tmuxpkg "github.com/marcosfelipeeipper/agentboard/internal/tmux"
)

type shellPopup struct {
	viewport   viewport.Model
	tmux       *tmuxpkg.Manager
	windowName string
	content    string
	width      int
	height     int
}

type shellContentMsg struct {
	content string
}

func newShellPopup(tm *tmuxpkg.Manager, windowName string, w, h int) shellPopup {
	vp := viewport.New(w/2, h-6)
	return shellPopup{
		viewport:   vp,
		tmux:       tm,
		windowName: windowName,
		width:      w,
		height:     h,
	}
}

func (s shellPopup) Init() tea.Cmd {
	return s.refreshContent()
}

func (s shellPopup) refreshContent() tea.Cmd {
	tm := s.tmux
	wn := s.windowName
	return func() tea.Msg {
		content, err := tm.CapturePane(context.Background(), wn)
		if err != nil {
			return shellContentMsg{content: "Error capturing pane: " + err.Error()}
		}
		return shellContentMsg{content: content}
	}
}

func (s shellPopup) Update(msg tea.Msg) (shellPopup, tea.Cmd) {
	switch msg := msg.(type) {
	case shellContentMsg:
		s.content = msg.content
		s.viewport.SetContent(msg.content)
		s.viewport.GotoBottom()
		return s, nil
	}

	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return s, cmd
}

func (s shellPopup) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e6b450")).
		Render("Shell: " + s.windowName)

	help := helpStyle.Render("esc: close | r: refresh | j/k: scroll")

	lines := []string{title, "", s.viewport.View(), "", help}
	content := strings.Join(lines, "\n")

	return overlayStyle.Width(s.width / 2).Height(s.height - 4).Render(content)
}
