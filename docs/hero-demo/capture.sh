#!/usr/bin/env bash
#
# Launches `stock tui` against the demo's packages.yaml. Unlike the store
# hero demo, stock queries live package managers on the host, so what you
# see depends on what's installed locally. That's intentional — the same
# config renders differently on a freshly-installed machine vs. one with
# a full dev toolchain.
#
# Requires the `stock` binary on your PATH (or pass one via `STOCK=...`).

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STOCK="${STOCK:-stock}"

# Surface the repo as ~/dotfiles for a portable header path, matching the
# store hero demo. No filesystem setup is needed beyond that — stock has
# no symlinks to place.
STAGE="${TMPDIR:-/tmp}/stock-hero-demo"
FAKE_HOME="$STAGE/home"
rm -rf "$STAGE"
mkdir -p "$FAKE_HOME"
ln -s "$HERE" "$FAKE_HOME/dotfiles"

cd "$FAKE_HOME/dotfiles"
exec env HOME="$FAKE_HOME" PWD="$FAKE_HOME/dotfiles" "$STOCK" tui
