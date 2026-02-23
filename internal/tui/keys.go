package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Left      key.Binding
	Right     key.Binding
	Up        key.Binding
	Down      key.Binding
	MoveRight key.Binding
	MoveLeft  key.Binding
	New       key.Binding
	Enter     key.Binding
	Delete    key.Binding
	Search    key.Binding
	Help      key.Binding
	Quit      key.Binding
	Escape    key.Binding
}

var keys = keyMap{
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "prev column"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "next column"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "prev task"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "next task"),
	),
	MoveRight: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move task right"),
	),
	MoveLeft: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "move task left"),
	),
	New: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "new task"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open task"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete task"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close/cancel"),
	),
}
