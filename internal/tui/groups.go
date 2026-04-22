package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
)

// ManagerPlan is one manager's declared-vs-installed view for a single group.
// Unavailable managers are still recorded so the detail pane can show that
// an alternative exists for a different platform.
type ManagerPlan struct {
	Name      string
	Available bool
	Desired   []string
	Missing   []string // populated only when Available
}

// Row is a single group row in the ledger.
type Row struct {
	Name     string
	State    State
	Summary  string       // short right-hand blurb: "5 pkgs · 2 missing", "darwin-only", etc.
	Managers []ManagerPlan
}

// Build computes a Row per group in cfg, aggregating per-group state from the
// managers that actually apply on this machine. The caller owns the cfg and
// platform info; Build touches neither.
func Build(cfg *config.File, info platform.Info) []Row {
	if cfg == nil {
		return nil
	}
	rows := make([]Row, 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		rows = append(rows, buildRow(g, info))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

func buildRow(g config.Group, info platform.Info) Row {
	r := Row{Name: g.Name}

	if !g.Applies(info) {
		r.State = StateSkipped
		r.Summary = whenSummary(g, info)
		r.Managers = declaredManagers(g)
		return r
	}

	names := managerKeys(g)
	var plans []ManagerPlan
	anyAvailable := false
	totalDesired := 0
	totalMissing := 0
	for _, name := range names {
		m := managers.Get(name)
		plan := ManagerPlan{Name: name, Desired: dedupe(g.Managers[name])}
		if m == nil {
			plans = append(plans, plan) // unknown key — doctor surfaces this
			continue
		}
		if !m.Available() {
			plans = append(plans, plan)
			continue
		}
		plan.Available = true
		anyAvailable = true
		installed, _ := m.Installed()
		plan.Missing = diffMissing(plan.Desired, installed)
		totalDesired += len(plan.Desired)
		totalMissing += len(plan.Missing)
		plans = append(plans, plan)
	}
	r.Managers = plans

	switch {
	case !anyAvailable:
		r.State = StateUnservable
		r.Summary = "no manager available"
	case totalMissing == 0:
		r.State = StateInstalled
		r.Summary = plural(totalDesired, "pkg", "pkgs")
	case totalMissing == totalDesired:
		r.State = StateMissing
		r.Summary = plural(totalDesired, "pkg", "pkgs") + " missing"
	default:
		r.State = StatePartial
		r.Summary = plural(totalDesired-totalMissing, "pkg", "pkgs") +
			" · " + itoa(totalMissing) + " missing"
	}
	return r
}

// whenSummary renders a compact reason a group was skipped, e.g.
// "needs darwin", by asking the when: clause which axis disagrees first.
func whenSummary(g config.Group, info platform.Info) string {
	if g.When == nil {
		return "skipped"
	}
	field, want, _ := g.When.FirstMismatch(info)
	if field == "" {
		return "skipped"
	}
	return "needs " + field + " " + want
}

func declaredManagers(g config.Group) []ManagerPlan {
	names := managerKeys(g)
	out := make([]ManagerPlan, 0, len(names))
	for _, name := range names {
		out = append(out, ManagerPlan{
			Name:    name,
			Desired: dedupe(g.Managers[name]),
		})
	}
	return out
}

func managerKeys(g config.Group) []string {
	names := make([]string, 0, len(g.Managers))
	for k := range g.Managers {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func diffMissing(desired, installed []string) []string {
	have := make(map[string]struct{}, len(installed))
	for _, p := range installed {
		have[p] = struct{}{}
	}
	var missing []string
	for _, p := range desired {
		if _, ok := have[p]; !ok {
			missing = append(missing, p)
		}
	}
	return missing
}

// Rows is the filtered, cursor-aware list view over a slice of Row.
type Rows struct {
	all    []Row
	view   []Row
	filter string
	cursor int
}

// NewRows wraps an initial slice of rows.
func NewRows(all []Row) *Rows {
	r := &Rows{all: all}
	r.rebuild()
	return r
}

// Replace swaps the backing rows, keeping the current filter and clamping
// the cursor if the new slice is shorter.
func (r *Rows) Replace(all []Row) {
	r.all = all
	r.rebuild()
}

func (r *Rows) Filter(q string)      { r.filter = q; r.rebuild() }
func (r *Rows) FilterQuery() string  { return r.filter }
func (r *Rows) Count() int           { return len(r.view) }
func (r *Rows) TotalCount() int      { return len(r.all) }
func (r *Rows) Cursor() int          { return r.cursor }
func (r *Rows) View() []Row          { return r.view }

// Selected returns the row under the cursor, or a zero Row if empty.
func (r *Rows) Selected() (Row, bool) {
	if r.cursor < 0 || r.cursor >= len(r.view) {
		return Row{}, false
	}
	return r.view[r.cursor], true
}

func (r *Rows) Up() {
	if r.cursor > 0 {
		r.cursor--
	}
}
func (r *Rows) Down() {
	if r.cursor < len(r.view)-1 {
		r.cursor++
	}
}
func (r *Rows) Top()    { r.cursor = 0 }
func (r *Rows) Bottom() {
	r.cursor = len(r.view) - 1
	if r.cursor < 0 {
		r.cursor = 0
	}
}

// Window returns the slice of rows that should be visible given the available
// height, keeping the cursor in view. topElided / bottomElided count rows
// clipped above and below the window.
func (r *Rows) Window(height int) (visible []Row, topElided, bottomElided int) {
	if height <= 0 || len(r.view) == 0 {
		return nil, 0, 0
	}
	if len(r.view) <= height {
		return r.view, 0, 0
	}
	start := r.cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(r.view) {
		end = len(r.view)
		start = end - height
	}
	return r.view[start:end], start, len(r.view) - end
}

// Summary returns "N installed  M missing  ..." for the ledger rule.
func (r *Rows) Summary() string {
	counts := map[State]int{}
	for _, row := range r.all {
		counts[row.State]++
	}
	var parts []string
	for _, st := range []State{StateInstalled, StatePartial, StateMissing, StateSkipped, StateUnservable} {
		if n := counts[st]; n > 0 {
			c := lipgloss.NewStyle().Foreground(st.Color())
			parts = append(parts,
				c.Render(st.Glyph())+" "+StyleMuted.Render(plural(n, st.Label(), st.Label())),
			)
		}
	}
	return strings.Join(parts, "   ")
}

// HeaderLine returns "N groups" or "N of M groups" when a filter is active.
func (r *Rows) HeaderLine() string {
	if r.filter == "" {
		return plural(len(r.all), "group", "groups")
	}
	return plural(len(r.view), "match", "matches") + " of " + plural(len(r.all), "group", "groups")
}

// RenderRow renders one row of the group ledger at the given content width.
func RenderRow(row Row, width int, selected bool) string {
	marker := "  "
	nameStyle := StyleFg
	if selected {
		marker = StyleEmber.Render(GlyphCursor) + " "
		nameStyle = StyleSelected
	}

	stateGlyph := lipgloss.NewStyle().Foreground(row.State.Color()).Render(row.State.Glyph())
	stateLabel := lipgloss.NewStyle().Foreground(row.State.Color()).Render(row.State.Label())
	rightCol := stateGlyph + "  " + stateLabel

	nameWidth := 14
	name := nameStyle.Render(padName(row.Name, nameWidth))
	prefixW := 2 // marker
	summaryBudget := width - prefixW - nameWidth - 3 - lipgloss.Width(rightCol)
	if summaryBudget < 10 {
		summaryBudget = 10
	}
	summary := StyleDim.Render(Clip(row.Summary, summaryBudget))

	line := marker + name + " " + summary
	used := lipgloss.Width(line)
	rightW := lipgloss.Width(rightCol)
	gap := width - used - rightW
	if gap < 1 {
		gap = 1
	}
	return line + strings.Repeat(" ", gap) + rightCol
}

func padName(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s + " "
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func (r *Rows) rebuild() {
	if r.filter == "" {
		r.view = append(r.view[:0], r.all...)
	} else {
		q := strings.ToLower(r.filter)
		r.view = r.view[:0]
		for _, row := range r.all {
			if strings.Contains(strings.ToLower(row.Name), q) {
				r.view = append(r.view, row)
			}
		}
	}
	if r.cursor >= len(r.view) {
		r.cursor = len(r.view) - 1
	}
	if r.cursor < 0 {
		r.cursor = 0
	}
}
