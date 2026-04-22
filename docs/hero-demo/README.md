# hero-demo

The fixture behind `docs/hero.png`. Scaffolding for the README
screenshot — not part of the `stock` binary.

## What it is

A minimal `.store/packages.yaml` laid out so every row state the TUI can
render appears in a single frame:

- **installed** (`cli`, `editors`) — every pacman package is already on
  disk, so the row gets the sage `●` badge.
- **partial** (`dev-tools`) — mix of present and missing packages across
  three parallel manager entries (pacman / apt / brew). This is the row
  the hero shot selects; the detail pane shows apt and brew dimmed as
  unavailable alternatives, pacman broken down as `3/5 installed · 2
  missing`, and the two missing packages listed with a green `+`.
- **missing** (`prompt-tools`) — nothing installed yet.
- **skipped** (`macos-gui`) — `when: { os: [darwin] }` so Linux skips it
  with the reason `needs os darwin`.
- **unservable** (`casks`) — `brew-cask` is the only declared manager
  and it isn't available on Linux, so `stock doctor` would flag it as a
  config problem (not a noisy cross-platform alternative).

## Layout

```text
hero-demo/
  .store/packages.yaml   # the demo fixture, committed to the repo
  capture.sh             # launch the live TUI against it
  snapshot.py            # regenerate docs/hero.png
  README.md              # this file
```

## See the demo live

```sh
$ make build && mv stock ~/.local/bin/     # or wherever's on your PATH
$ ./capture.sh                             # opens stock tui against the fixture
```

`capture.sh` symlinks this directory in as `~/dotfiles` inside an
ephemeral `$HOME` so the header shows a portable path, then exec's
`stock tui`. Your real home and real `.store/` are never touched.

State depends on the managers on your host. On a typical Arch machine
without `starship`/`atuin` installed you'll see the same layout as the
hero. On a fresh VM with no pacman packages, everything flips to
missing — still a valid picture of the TUI, just a different story.
Press `q` to quit; `r` recomputes against live manager state.

## Regenerate the PNG

```sh
$ make build
$ python3 -m venv .venv && .venv/bin/pip install pyte pillow
$ .venv/bin/python snapshot.py --stock /absolute/path/to/stock --out ../hero.png
```

The script spawns `stock tui` in a pty, feeds `jj` after the initial
render so the cursor lands on `dev-tools` (row 2 — casks, cli,
**dev-tools**), captures the final frame with
[pyte](https://github.com/selectel/pyte), and rasterises it with Pillow
using the palette from `internal/tui/theme.go`.

Flags:

| Flag          | Default                            | Notes                                       |
| ------------- | ---------------------------------- | ------------------------------------------- |
| `--stock`     | `$(which stock)` or `$TMPDIR/stock`| Absolute path to the `stock` binary         |
| `--out`       | `docs/hero.png`                    | Output PNG                                  |
| `--cols`      | `96`                               | Terminal width in cells                     |
| `--rows`      | `30`                               | Terminal height in cells                    |
| `--font-size` | `22`                               | Pixel size passed to Pillow                 |

Use an absolute path for `--stock`. The script chdirs the child process
into the ephemeral home before exec, so relative paths will miss.

## Why the initial wait

`snapshot.py` idles ~6.5s before pressing any keys. `termenv` probes
the terminal for truecolor support on startup and waits up to 5
seconds for a response. We let it time out rather than forging a reply
— answering the probe races with bubbletea's input reader and produced
garbled frames in testing.
