package tui

import "github.com/charmbracelet/bubbles/key"

type globalKeyMap struct {
	Quit key.Binding
}

var globalKeys = globalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
	),
}
