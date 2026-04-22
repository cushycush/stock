# stock

Package, tool, and runtime installer. Companion to
[`store`](https://github.com/cushycush/store) — you stock a store with
inventory. Same `.store/` directory, same `when:` platform filters, same
hook env contract.

> **Status:** v1 complete, no unit tests yet. Manually exercised against
> fixture configs on Linux. Use on real machines at your own risk.

## Install

```sh
go install github.com/cushycush/stock/cmd/stock@latest
```

Requires Go 1.26+.

## Config

`stock` reads `.store/packages.yaml` at the repo root. Each top-level group
maps a manager key (`brew`, `apt`, `pacman`, …) to a list of package names.
An optional `when:` clause gates the group.

```yaml
packages:
  core:
    pacman: [git, ripgrep, fd, bat]
    apt:    [git, ripgrep, fd-find, bat]
    brew:   [git, ripgrep, fd, bat]
    when:   { os: [linux, darwin] }

  gui-linux:
    pacman: [firefox, alacritty]
    apt:    [firefox-esr, alacritty]
    when:   { os: linux }

  work-laptop:
    brew: [tailscale, 1password-cli]
    when: { hostname: [work-mbp, work-mbp-2] }
```

Within a group, listing multiple managers side-by-side is the standard
pattern: `stock` runs whichever manager is available on the current
machine. A group is flagged as unservable by `doctor` only when **none** of
its managers are installed — `apt`-on-Arch or `brew`-on-Linux are not
warnings.

### `when:` fields

`os`, `arch`, `distro`, `distro_version`, `hostname`, `shell`, `wsl`. Each
string field accepts a scalar (`os: linux`) or a list (`os: [linux, darwin]`).
All specified fields must match; within a list, any entry matches.
Semantics match [store-core](https://github.com/cushycush/store-core#when-matching).

## Commands

```
stock install [group...]   install everything matching platform + when:
stock diff    [group...]   preview what install would change (read-only)
stock doctor               verify managers, detect drift from packages.yaml
stock snapshot             write currently installed packages to .store/packages.yaml
stock platform             print detected platform info
stock bootstrap            run the full new-machine flow (hooks, install, store)
```

Flags:

- `--dry-run` — print commands that would run; don't execute. Works for
  `install` and `bootstrap`.
- `snapshot --write` — write to `.store/packages.yaml` instead of stdout.
  Pair with `--force` to overwrite an existing file. `--group <name>`
  chooses the group header (default `host`). `--managers brew,cargo`
  restricts which managers are snapshotted.
- `bootstrap --skip-store` — run hooks and `install`, but skip invoking
  the `store` binary afterwards.

Unknown commands fall back to `stock-<name>` on `$PATH` (git-style), so you
can add your own subcommands without recompiling.

## Hooks

`stock` runs executables placed under `<root>/.store/hooks/` before and
after install:

| Hook name | When it runs |
|---|---|
| `pre-install` | before `stock install` |
| `post-install` | after `stock install` |
| `pre-bootstrap` | at the start of `stock bootstrap` |
| `post-bootstrap` | at the end of `stock bootstrap` |

Hooks receive the standard [`STORE_*` env vars](https://github.com/cushycush/store-core#hook-env-contract)
plus `STORE_ACTION=install` (or `bootstrap`), and run with `$STORE_ROOT`
as the working directory.

## Supported managers

`brew`, `brew-cask`, `apt`, `pacman`, `dnf` (alias `yum`), `zypper`, `apk`,
`winget`, `cargo`, `go`, `npm`, `pipx`, `gem`.

Contributions of new managers live in [`internal/managers/`](./internal/managers).
Each file registers itself via `init()` and implements a short interface
(`Name`, `Available`, `Installed`, `Install`, `BootstrapHint`).

## License

MIT.
