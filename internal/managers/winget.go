package managers

type winget struct{ base }

func init() { Register(&winget{}) }

func (w *winget) Name() string    { return "winget" }
func (w *winget) Available() bool { return w.runner().Has("winget") }
func (w *winget) BootstrapHint() string {
	return "winget ships with Windows 10 1809+ / Windows 11 via App Installer"
}

// winget cannot be enumerated into a simple name list without brittle parsing
// of its tabular output. Return empty + nil so callers fall back to winget's
// own idempotency (`--exact --accept-package-agreements` is a no-op if present).
func (w *winget) Installed() ([]string, error) { return nil, nil }

func (w *winget) Install(pkgs []string) error {
	for _, p := range pkgs {
		if err := w.runner().Run(
			"winget", "install",
			"--exact", "--silent",
			"--accept-package-agreements",
			"--accept-source-agreements",
			"--id", p,
		); err != nil {
			return err
		}
	}
	return nil
}
