# Changelog

All notable changes to `stock` are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.1] - 2026-04-22

### Fixed

- `go.sum` was missing entries for `golang.org/x/term` and `golang.org/x/sys`
  after the v0.2.0 bump to `store-core` (which pulled in `x/term` through
  its new `ui` package). The workspace `go.work` file masked the gap
  locally; CI and the release workflow failed on the v0.3.0 tag. Resolved
  with `GOWORK=off go mod tidy`. The v0.3.0 tag still exists but has no
  release artifacts — use v0.3.1 or later.

## [0.3.0] - 2026-04-22

### Added

- **Arch Linux (AUR)** distribution via three packages:
  [`stock`](https://aur.archlinux.org/packages/stock) (source build),
  [`stock-bin`](https://aur.archlinux.org/packages/stock-bin) (prebuilt
  binary), and [`stock-git`](https://aur.archlinux.org/packages/stock-git)
  (tip-of-main). The release workflow pushes `stock` and `stock-bin`
  updates automatically on every tagged release.
- **Nix flake** — `nix run github:cushycush/stock`,
  `nix profile install github:cushycush/stock`, and a pinned Go + gopls
  dev shell via `nix develop github:cushycush/stock`.
- **GitHub release binaries** — cross-compiled zips for linux, macOS, and
  Windows on amd64 / arm64 (except windows/arm64), attached to every
  `v*.*.*` tag by `.github/workflows/release.yml`.
- **CI** — `.github/workflows/test.yml` runs `go test ./...` and
  `go build ./cmd/stock` on ubuntu-latest, macos-latest, and
  windows-latest for every PR and push to `main`.
- `stock version` / `stock --version` now reports the string injected at
  build time via `-ldflags "-X main.version=vX.Y.Z"`. `go install` builds
  still report `stock dev`.

## [0.2.0] - 2026-04-22

### Changed

- CLI output is now styled through
  [`store-core/ui`](https://github.com/cushycush/store-core/tree/main/ui).
  `doctor` shows green `[ok]` chips for available managers and a dim
  placeholder for absent ones; `diff` uses a green `+` for additions and
  `ui.Success` for the "nothing to install" line; `install` and
  `bootstrap` render phase headers and manager names in bold. Warnings go
  through `ui.Warning` with a `⚠` glyph. Styling auto-disables on
  non-terminal stdout and honors `NO_COLOR` / `FORCE_COLOR`.
- Requires `store-core v0.2.0`.

## [0.1.0] - 2026-04-22

First tagged release.

### Added

- `.store/packages.yaml` schema: groups map manager keys (`brew`, `apt`,
  `pacman`, …) to package lists, with optional `when:` platform filters
  that accept either a YAML scalar (`os: linux`) or a list
  (`os: [linux, darwin]`).
- Supported managers: `brew`, `brew-cask`, `apt`, `pacman`, `dnf`
  (falls back to `yum`), `zypper`, `apk`, `winget`, `cargo`, `go`, `npm`,
  `pipx`, `gem`. Listing multiple managers inside one group is expected —
  `stock` runs whichever is available on the current machine.
- Commands: `install`, `diff`, `doctor`, `snapshot`, `platform`,
  `bootstrap`. `--dry-run` on `install` and `bootstrap`.
- Git-style subcommand dispatch: unknown commands fall back to
  `stock-<name>` on `$PATH`.
- Hooks under `.store/hooks/` run with the shared `STORE_*` env contract
  provided by
  [`store-core/hooks`](https://github.com/cushycush/store-core/tree/main/hooks).
- Unit tests for the load-bearing pieces: config parsing (96%), runner
  dispatch (87%), managers registry, and full coverage of plan
  orchestration and `RunInstall`.
