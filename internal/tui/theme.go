// Package tui is the read-only dashboard for stock. It shows every group in
// packages.yaml, whether its packages are installed, and what install would
// do — with the same visual language as the store TUI so the two binaries
// feel like a matched pair.
package tui

import "github.com/charmbracelet/lipgloss"

// Palette. Deliberately identical to the store TUI so the two tools share
// a single visual identity. The signature accent (Ember) carries the UI;
// severity hues stay dusty so nothing competes with it.
var (
	ColorFg    = lipgloss.Color("#EDE6DC")
	ColorMuted = lipgloss.Color("#A69B8A")
	ColorDim   = lipgloss.Color("#6B6558")
	ColorFaint = lipgloss.Color("#3F3B35")

	ColorEmber    = lipgloss.Color("#E89A3A")
	ColorEmberLow = lipgloss.Color("#7A5324")

	ColorInstalled = lipgloss.Color("#8AA27A") // dusty sage
	ColorPartial   = lipgloss.Color("#D9A55E") // dim amber
	ColorMissing   = lipgloss.Color("#847C6E") // warm gray — missing is not an error
	ColorError     = lipgloss.Color("#C27B6B") // terracotta
)

var (
	StyleFg    = lipgloss.NewStyle().Foreground(ColorFg)
	StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleDim   = lipgloss.NewStyle().Foreground(ColorDim)
	StyleFaint = lipgloss.NewStyle().Foreground(ColorFaint)

	StyleEmber    = lipgloss.NewStyle().Foreground(ColorEmber)
	StyleEmberDim = lipgloss.NewStyle().Foreground(ColorEmberLow)

	StyleBold     = lipgloss.NewStyle().Foreground(ColorFg).Bold(true)
	StyleCursor   = lipgloss.NewStyle().Foreground(ColorEmber)
	StyleSelected = lipgloss.NewStyle().Foreground(ColorFg).Bold(true)
	StyleHint     = lipgloss.NewStyle().Foreground(ColorDim)
	StyleHintKey  = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Glyphs are single-rune and never padded.
const (
	GlyphInstalled  = "●"
	GlyphPartial    = "◐"
	GlyphMissing    = "○"
	GlyphSkipped    = "—"
	GlyphUnservable = "✕"
	GlyphCursor     = "▸"
	GlyphHeart      = "·"
)

// State is the aggregate state of a group.
type State int

const (
	StateInstalled State = iota
	StatePartial
	StateMissing
	StateSkipped    // when: filter does not match this machine
	StateUnservable // no manager listed for this group is available
)

// Glyph returns the single-rune glyph for the state.
func (s State) Glyph() string {
	switch s {
	case StateInstalled:
		return GlyphInstalled
	case StatePartial:
		return GlyphPartial
	case StateMissing:
		return GlyphMissing
	case StateSkipped:
		return GlyphSkipped
	case StateUnservable:
		return GlyphUnservable
	}
	return " "
}

// Color returns the foreground color for the state.
func (s State) Color() lipgloss.Color {
	switch s {
	case StateInstalled:
		return ColorInstalled
	case StatePartial:
		return ColorPartial
	case StateMissing:
		return ColorMissing
	case StateSkipped:
		return ColorDim
	case StateUnservable:
		return ColorError
	}
	return ColorMuted
}

// Label returns the short status word.
func (s State) Label() string {
	switch s {
	case StateInstalled:
		return "installed"
	case StatePartial:
		return "partial"
	case StateMissing:
		return "missing"
	case StateSkipped:
		return "skipped"
	case StateUnservable:
		return "unservable"
	}
	return ""
}
