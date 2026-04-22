package tui

import "github.com/charmbracelet/bubbles/key"

// Keymap is the read-only keymap for the stock TUI. The surface is small on
// purpose: this cut doesn't run installers, so there's nothing to bind
// capital-letter keys to. When install/diff actions land later they'll
// follow store's convention — lowercase for safe, capital for broad.
type Keymap struct {
	Up      key.Binding
	Down    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Back    key.Binding
	Filter  key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultKeymap returns the canonical keymap.
func DefaultKeymap() Keymap {
	return Keymap{
		Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		Top:     key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:  key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		Back:    key.NewBinding(key.WithKeys("esc", "h"), key.WithHelp("esc", "back")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// FooterHints returns the condensed hint strip shown at the bottom.
func (k Keymap) FooterHints() string {
	h := func(key, label string) string {
		return StyleHintKey.Render(key) + StyleHint.Render(" "+label)
	}
	sep := StyleDim.Render("   ")
	return h("j/k", "move") + sep +
		h("/", "filter") + sep +
		h("r", "refresh") + sep +
		h("?", "help") + sep +
		h("q", "quit")
}

// HelpLines returns the multi-line help text shown in the help overlay.
func (k Keymap) HelpLines() []string {
	return []string{
		"j / k        move",
		"g / G        top · bottom",
		"/            filter groups by name",
		"r            recompute plan",
		"?            this help",
		"q · esc      quit · close",
		"",
		"read-only dashboard — actions (install, diff) still run from the CLI.",
	}
}
