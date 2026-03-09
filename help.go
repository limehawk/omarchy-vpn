package main

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Connect    key.Binding
	Disconnect key.Binding
	Import     key.Binding
	Rename     key.Binding
	Delete     key.Binding
	Help       key.Binding
	Quit       key.Binding
	ForceQuit  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "move down"),
		),
		Connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		Disconnect: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "disconnect"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import config"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename config"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete config"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Connect, k.Disconnect, k.Import, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Connect, k.Disconnect},
		{k.Import, k.Rename, k.Delete},
		{k.Help, k.Quit, k.ForceQuit},
	}
}

func newHelp() help.Model {
	h := help.New()
	s := help.DefaultDarkStyles()
	s.ShortKey = lipgloss.NewStyle().Foreground(lavender).Bold(true)
	s.ShortDesc = lipgloss.NewStyle().Foreground(overlay0)
	s.ShortSeparator = lipgloss.NewStyle().Foreground(surface1)
	s.FullKey = lipgloss.NewStyle().Foreground(lavender).Bold(true)
	s.FullDesc = lipgloss.NewStyle().Foreground(subtext)
	s.FullSeparator = lipgloss.NewStyle().Foreground(surface1)
	h.Styles = s
	h.ShortSeparator = " │ "
	h.FullSeparator = " │ "
	return h
}
