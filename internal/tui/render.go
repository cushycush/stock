package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Rule renders a horizontal rule with an embedded title and optional
// right-aligned status word — the typographic move that gives the store/stock
// TUIs their signature look. Borrowed verbatim from store.
func Rule(width int, title, rightLabel string, rightColor lipgloss.Color) string {
	if width < 8 {
		width = 8
	}
	leftDashes := StyleDim.Render(strings.Repeat("─", 3))
	titleMid := ""
	if title != "" {
		titleMid = " " + StyleFg.Render(title) + " "
	}
	rightPart := ""
	if rightLabel != "" {
		c := StyleEmber
		if rightColor != "" {
			c = lipgloss.NewStyle().Foreground(rightColor)
		}
		rightPart = " " + c.Render(rightLabel) + " " + StyleDim.Render("─")
	}
	used := lipgloss.Width(leftDashes) + lipgloss.Width(titleMid) + lipgloss.Width(rightPart)
	mid := width - used
	if mid < 3 {
		mid = 3
	}
	middleDashes := StyleDim.Render(strings.Repeat("─", mid))
	return leftDashes + titleMid + middleDashes + rightPart
}

// PadRight right-pads s with spaces so its printable width equals width.
func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// Clip truncates s to at most width printable columns, adding an ellipsis
// glyph if truncation occurred.
func Clip(s string, width int) string {
	w := lipgloss.Width(s)
	if w <= width || width < 1 {
		return s
	}
	if width < 2 {
		return string([]rune(s)[:1])
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// IndentBlock indents every line of s with n spaces.
func IndentBlock(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return "1 " + singular
	}
	return itoa(n) + " " + pluralForm
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
