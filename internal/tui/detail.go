package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderDetail composes the per-group detail pane: one sub-section per
// manager declared under the group, with the desired list and a compact
// note about what install would do. Unavailable managers are dimmed and
// labelled so the user sees cross-platform alternatives without thinking
// they're missing.
func RenderDetail(row Row, width int) string {
	if len(row.Managers) == 0 {
		return "  " + StyleDim.Render("no managers declared")
	}

	var b strings.Builder
	for i, mp := range row.Managers {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderManagerBlock(row, mp, width))
		b.WriteString("\n")
	}
	return b.String()
}

func renderManagerBlock(row Row, mp ManagerPlan, width int) string {
	header := renderManagerHeader(row, mp)
	body := renderManagerBody(row, mp, width)
	if body == "" {
		return "  " + header
	}
	return "  " + header + "\n" + body
}

func renderManagerHeader(row Row, mp ManagerPlan) string {
	name := StyleBold.Render(mp.Name)
	var note string
	switch {
	case row.State == StateSkipped:
		note = StyleDim.Render("(" + plural(len(mp.Desired), "pkg", "pkgs") + " · skipped)")
	case !mp.Available:
		note = StyleDim.Render("(" + plural(len(mp.Desired), "pkg", "pkgs") + " · not on this machine)")
	case len(mp.Missing) == 0:
		note = lipgloss.NewStyle().Foreground(ColorInstalled).
			Render(plural(len(mp.Desired), "pkg", "pkgs") + " · installed")
	case len(mp.Missing) == len(mp.Desired):
		note = lipgloss.NewStyle().Foreground(ColorMissing).
			Render(plural(len(mp.Desired), "pkg", "pkgs") + " · " + itoa(len(mp.Missing)) + " missing")
	default:
		note = lipgloss.NewStyle().Foreground(ColorPartial).
			Render(itoa(len(mp.Desired)-len(mp.Missing)) + "/" + itoa(len(mp.Desired)) + " installed · " +
				itoa(len(mp.Missing)) + " missing")
	}
	return name + "  " + note
}

func renderManagerBody(row Row, mp ManagerPlan, width int) string {
	if len(mp.Desired) == 0 {
		return ""
	}
	// Skipped / unavailable managers: show the declared package list dimmed,
	// wrapped. Users reading the detail pane are trying to understand the
	// config — "what was declared here" matters even when nothing will run.
	if row.State == StateSkipped || !mp.Available {
		return IndentBlock(StyleDim.Render(wrapList(mp.Desired, width-6)), 6)
	}
	// Applicable manager with nothing missing: one soft "all present" line
	// beats listing dozens of package names the user already trusts.
	if len(mp.Missing) == 0 {
		return IndentBlock(StyleDim.Render("all present"), 6)
	}
	// Partial / missing: show only the missing packages, marked `+`.
	lines := make([]string, 0, len(mp.Missing))
	marker := lipgloss.NewStyle().Foreground(ColorInstalled).Render("+")
	for _, pkg := range mp.Missing {
		lines = append(lines, marker+" "+StyleFg.Render(pkg))
	}
	return IndentBlock(strings.Join(lines, "\n"), 6)
}

// wrapList renders a comma-separated list, wrapping before width.
func wrapList(items []string, width int) string {
	if width < 10 {
		width = 10
	}
	var b strings.Builder
	line := ""
	for i, s := range items {
		sep := ""
		if i > 0 {
			sep = ", "
		}
		if lipgloss.Width(line)+lipgloss.Width(sep)+lipgloss.Width(s) > width {
			if line != "" {
				b.WriteString(line)
				b.WriteString("\n")
			}
			line = s
			continue
		}
		line += sep + s
	}
	if line != "" {
		b.WriteString(line)
	}
	return b.String()
}
