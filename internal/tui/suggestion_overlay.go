package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/markx3/agentboard/internal/db"
)

type suggestionOverlay struct {
	suggestions []db.Suggestion
	cursor      int
	width       int
	height      int
}

func newSuggestionOverlay(suggestions []db.Suggestion, w, h int) suggestionOverlay {
	return suggestionOverlay{suggestions: suggestions, width: w, height: h}
}

func (o suggestionOverlay) Update(msg tea.KeyMsg) (suggestionOverlay, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if o.cursor > 0 {
			o.cursor--
		}
	case key.Matches(msg, keys.Down):
		if o.cursor < len(o.suggestions)-1 {
			o.cursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(o.suggestions) > 0 {
			s := o.suggestions[o.cursor]
			return o, func() tea.Msg {
				return suggestionAcceptedMsg{suggestionID: s.ID}
			}
		}
	case msg.String() == "d":
		if len(o.suggestions) > 0 {
			s := o.suggestions[o.cursor]
			return o, func() tea.Msg {
				return suggestionDismissedMsg{suggestionID: s.ID}
			}
		}
	}
	return o, nil
}

func (o suggestionOverlay) View() string {
	title := formTitleStyle.Render("Pending Proposals")

	var lines []string
	lines = append(lines, title, "")

	if len(o.suggestions) == 0 {
		lines = append(lines, "No pending proposals.")
		lines = append(lines, "", helpStyle.Render("esc: close"))
		return overlayStyle.Width(o.width/2).Render(strings.Join(lines, "\n"))
	}

	// List items
	for i, s := range o.suggestions {
		typeBadge := fmt.Sprintf("[%s]", s.Type)
		titleStr := s.Title
		if titleStr == "" {
			titleStr = "(untitled)"
		}
		line := fmt.Sprintf("%s %s", typeBadge, titleStr)
		if i == o.cursor {
			line = agentActiveStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	// Detail panel for selected item
	if len(o.suggestions) > 0 {
		selected := o.suggestions[o.cursor]
		lines = append(lines, "")
		if selected.Author != "" {
			lines = append(lines, fmt.Sprintf("Author: %s", selected.Author))
		}
		if selected.Message != "" {
			// Truncate long messages to fit the overlay
			msg := selected.Message
			maxLen := o.width/2 - 8
			if maxLen < 20 {
				maxLen = 20
			}
			if len(msg) > maxLen {
				msg = msg[:maxLen-3] + "..."
			}
			lines = append(lines, fmt.Sprintf("Details: %s", msg))
		}
	}

	lines = append(lines, "", helpStyle.Render("↑/↓: navigate  Enter: accept  d: dismiss  Esc: close"))

	return overlayStyle.Width(o.width/2).Render(strings.Join(lines, "\n"))
}
