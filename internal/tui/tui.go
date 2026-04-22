package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
	"github.com/cushycush/stock/internal/runner"
)

// App is the read-only Bubble Tea model for `stock tui`. It owns the loaded
// config + the computed rows and provides a small set of navigation keys.
// Mutations (install, snapshot) still run from the CLI; the TUI is the
// inspector.
type App struct {
	root    string
	version string
	cfg     *config.File
	info    platform.Info

	rows    *Rows
	keys    Keymap
	hasStore bool

	width, height int
	ready         bool

	filterMode  bool
	filterInput textinput.Model

	showHelp bool
}

// New constructs the model. cfg may be nil if packages.yaml is missing —
// the view degrades to an "empty config" hint. Managers must already be
// bound to a runner (callers that need to re-run managers.Bind should do
// so before calling New).
func New(root, version string, cfg *config.File, info platform.Info) *App {
	_, storeErr := exec.LookPath("store")
	a := &App{
		root:     root,
		version:  version,
		cfg:      cfg,
		info:     info,
		keys:     DefaultKeymap(),
		hasStore: storeErr == nil,
	}
	a.rows = NewRows(Build(cfg, info))
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 64
	a.filterInput = ti
	return a
}

// Run starts the program. Use tea.WithAltScreen so the TUI doesn't smear
// over the user's scrollback, matching store's behavior.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		a.ready = true
		return a, nil
	}

	if a.showHelp {
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "?", "esc", "q", "enter":
				a.showHelp = false
			case "ctrl+c":
				return a, tea.Quit
			}
		}
		return a, nil
	}

	if a.filterMode {
		return a.updateFilter(msg)
	}

	if k, ok := msg.(tea.KeyMsg); ok {
		return a.handleKey(k)
	}
	return a, nil
}

func (a *App) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, a.keys.Quit):
		return a, tea.Quit
	case key.Matches(k, a.keys.Help):
		a.showHelp = true
		return a, nil
	case key.Matches(k, a.keys.Refresh):
		a.refresh()
		return a, nil
	case key.Matches(k, a.keys.Filter):
		a.filterMode = true
		a.filterInput.SetValue("")
		a.filterInput.Focus()
		return a, textinput.Blink
	case key.Matches(k, a.keys.Up):
		a.rows.Up()
	case key.Matches(k, a.keys.Down):
		a.rows.Down()
	case key.Matches(k, a.keys.Top):
		a.rows.Top()
	case key.Matches(k, a.keys.Bottom):
		a.rows.Bottom()
	case key.Matches(k, a.keys.Back):
		if a.rows.FilterQuery() != "" {
			a.rows.Filter("")
		}
	}
	return a, nil
}

func (a *App) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch k.String() {
	case "esc":
		a.filterMode = false
		a.filterInput.Blur()
		a.rows.Filter("")
		return a, nil
	case "enter":
		a.filterMode = false
		a.filterInput.Blur()
		return a, nil
	case "ctrl+c":
		return a, tea.Quit
	}
	var cmd tea.Cmd
	a.filterInput, cmd = a.filterInput.Update(msg)
	a.rows.Filter(a.filterInput.Value())
	return a, cmd
}

// refresh rebuilds the rows from the current config. Re-reads packages.yaml
// from disk so the user can edit it in another pane and hit `r`.
func (a *App) refresh() {
	// Rebind managers to a fresh exec runner; Installed() lists can change
	// out from under us (brew install in another terminal) and we want the
	// latest view when the user explicitly asks for it.
	managers.Bind(runner.NewExec())
	if cfg, err := config.Load(a.root); err == nil {
		a.cfg = cfg
	}
	a.rows.Replace(Build(a.cfg, a.info))
}

// View implements tea.Model.
func (a *App) View() string {
	if !a.ready {
		return ""
	}
	if a.showHelp {
		return a.renderHelp()
	}
	main := a.renderMain()

	footer := "  " + a.keys.FooterHints()
	budget := a.height - 2
	if budget < 1 {
		budget = 1
	}
	lines := strings.Split(main, "\n")
	if len(lines) > budget {
		lines = lines[:budget]
	}
	for len(lines) < budget {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n\n" + footer
}

func (a *App) renderMain() string {
	width := a.width
	if width < 40 {
		width = 40
	}

	var b strings.Builder
	b.WriteString(a.renderHeader(width))
	b.WriteString("\n\n")

	// Empty-config hint.
	if a.cfg == nil || len(a.cfg.Groups) == 0 {
		b.WriteString("  " + StyleDim.Render("no packages.yaml here yet · run "+StyleEmber.Render("stock snapshot")+StyleDim.Render(" to seed one")))
		b.WriteString("\n")
		return b.String()
	}

	// Groups section.
	b.WriteString(Rule(width-4, a.rows.HeaderLine(), a.rows.Summary(), ""))
	b.WriteString("\n\n")

	rows := a.rows.View()
	if len(rows) == 0 && a.rows.FilterQuery() != "" {
		b.WriteString("  " + StyleDim.Render("no matches for \""+a.rows.FilterQuery()+"\""))
		b.WriteString("\n")
	} else {
		budget := (a.height - 14) / 2
		if budget < 5 {
			budget = 5
		}
		if budget > len(rows) {
			budget = len(rows)
		}
		visible, topElided, bottomElided := a.rows.Window(budget)
		cursor := a.rows.Cursor()
		if topElided > 0 {
			b.WriteString("  " + StyleDim.Render("↑ "+itoa(topElided)+" more") + "\n")
		}
		for i, row := range visible {
			globalI := topElided + i
			b.WriteString("  " + RenderRow(row, width-4, globalI == cursor))
			b.WriteString("\n")
		}
		if bottomElided > 0 {
			b.WriteString("  " + StyleDim.Render("↓ "+itoa(bottomElided)+" more") + "\n")
		}
	}

	if a.filterMode {
		b.WriteString("\n  " + StyleEmber.Render("/") + a.filterInput.View() + "\n")
	}
	b.WriteString("\n")

	// Detail section for the selected group.
	if sel, ok := a.rows.Selected(); ok {
		ruleColor := sel.State.Color()
		b.WriteString(Rule(width-4, sel.Name, sel.State.Label(), ruleColor))
		b.WriteString("\n\n")
		b.WriteString(RenderDetail(sel, width-4))
	}
	return b.String()
}

func (a *App) renderHeader(width int) string {
	brand := StyleBold.Render("stock")
	root := StyleDim.Render(shortHome(a.root))
	plat := StyleDim.Render(a.info.OS + "/" + a.info.Arch)

	// Mirror store's "companion tool" signpost: when store is on $PATH,
	// point at it so users know the pair exists.
	var storeHint string
	if a.hasStore {
		storeHint = StyleDim.Render("   ") + StyleEmberDim.Render("store")
	}

	left := "  " + brand
	right := root + StyleDim.Render("   ") + plat + storeHint + " "
	used := lipgloss.Width(left) + lipgloss.Width(right)
	fill := width - used
	if fill < 1 {
		fill = 1
	}
	return left + strings.Repeat(" ", fill) + right
}

func (a *App) renderHelp() string {
	width := a.width * 7 / 10
	if width < 48 {
		width = a.width - 8
	}
	if width < 36 {
		width = a.width - 2
	}

	var body strings.Builder
	body.WriteString(StyleEmber.Render(":: ") + StyleBold.Render("help"))
	body.WriteString("\n\n")
	for _, line := range a.keys.HelpLines() {
		body.WriteString(StyleFg.Render(line))
		body.WriteString("\n")
	}
	body.WriteString("\n")
	body.WriteString(StyleHint.Render("? · esc  close"))

	frame := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(ColorDim).
		Padding(1, 2).
		Width(width).
		Render(body.String())
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, frame, lipgloss.WithWhitespaceChars(" "))
}

func shortHome(root string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return root
	}
	if strings.HasPrefix(root, home) {
		return "~" + root[len(home):]
	}
	return root
}
