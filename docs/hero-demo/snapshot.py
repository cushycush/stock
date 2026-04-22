#!/usr/bin/env python3
"""
Capture the stock TUI as a PNG for the README hero image.

Spawns `stock tui` inside an ephemeral HOME whose packages.yaml is the
demo fixture, presses `j j` so the cursor lands on `dev-tools` (the
partial row — best showcase for the detail pane), reads the final frame
through a pty, parses it with pyte, and renders it to a PNG with Pillow.

Usage:
  docs/hero-demo/snapshot.py [--stock PATH] [--out PATH] [--cols N] [--rows N]

Defaults write docs/hero.png at 96x30 cells.

States displayed depend on the host's live package managers — capturing
on a machine without atuin and starship gives the canonical mix.
"""
from __future__ import annotations

import argparse
import fcntl
import os
import pty
import select
import shutil
import signal
import struct
import subprocess
import sys
import termios
import time
from pathlib import Path

import pyte
from PIL import Image, ImageDraw, ImageFont


HERE = Path(__file__).resolve().parent
REPO = HERE.parent.parent


# ----- palette (pulled from internal/tui/theme.go) -----------------------

BG = "#111111"
COLORS = {
    "default":   "#EDE6DC",  # ColorFg
    "muted":     "#A69B8A",
    "dim":       "#6B6558",
    "faint":     "#3F3B35",
    "ember":     "#E89A3A",
    "emberlow":  "#7A5324",
    "installed": "#8AA27A",
    "partial":   "#D9A55E",
    "missing":   "#847C6E",
    "error":     "#C27B6B",
}


def setup_stage(stage: Path) -> Path:
    """Create the ephemeral HOME. stock itself doesn't place anything on
    disk — we only need a symlink so the header reads `~/dotfiles` rather
    than an absolute repo path."""
    if stage.exists():
        shutil.rmtree(stage)
    home = stage / "home"
    home.mkdir(parents=True)
    dotfiles = home / "dotfiles"
    dotfiles.symlink_to(HERE)
    return dotfiles


def spawn_tui(stock_bin: str, home: Path, cwd: Path, cols: int, rows: int) -> tuple[int, int]:
    pid, fd = pty.fork()
    if pid == 0:
        env = os.environ.copy()
        env["HOME"] = str(home)
        env["TERM"] = "xterm-256color"
        env["COLORTERM"] = "truecolor"
        # PWD tells Go's os.Getwd to keep the symlink path; without it
        # the tui would display the resolved absolute path in the header.
        env["PWD"] = str(cwd)
        os.chdir(cwd)
        os.execvpe(stock_bin, [stock_bin, "tui"], env)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
    return pid, fd


def drain(fd: int, stream: pyte.Stream, timeout: float) -> None:
    end = time.monotonic() + timeout
    while True:
        remaining = end - time.monotonic()
        if remaining <= 0:
            return
        r, _, _ = select.select([fd], [], [], remaining)
        if not r:
            return
        try:
            data = os.read(fd, 65536)
        except OSError:
            return
        if not data:
            return
        stream.feed(data.decode("utf-8", errors="replace"))


def capture_frame(stock_bin: str, cols: int, rows: int) -> pyte.Screen:
    stage = Path(os.environ.get("TMPDIR", "/tmp")) / "stock-hero-demo"
    dotfiles = setup_stage(stage)
    home = dotfiles.parent
    pid, fd = spawn_tui(stock_bin, home, dotfiles, cols, rows)

    screen = pyte.Screen(cols, rows)
    stream = pyte.Stream(screen)
    try:
        # termenv probes terminal capabilities on startup and waits up to
        # ~5 seconds for a response. Letting it time out is slower but
        # reliable — feeding fake responses races with bubbletea's input
        # reader and produced garbled frames in testing. One extra 5s at
        # snapshot time is fine for a script no-one runs interactively.
        drain(fd, stream, 6.5)
        # Move to `dev-tools` — the partial row. Alphabetical order is
        # casks, cli, dev-tools, editors, macos-gui, prompt-tools, so two
        # `j` presses from the top land on row index 2.
        os.write(fd, b"j")
        time.sleep(0.15)
        drain(fd, stream, 0.15)
        os.write(fd, b"j")
        drain(fd, stream, 1.0)
    finally:
        try:
            os.write(fd, b"q")
        except OSError:
            pass
        time.sleep(0.2)
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        os.close(fd)
        os.waitpid(pid, 0)
    return screen


# ----- PNG renderer ------------------------------------------------------

def ansi_color(attr, fg: bool) -> str:
    name = attr.fg if fg else attr.bg
    if name == "default":
        return COLORS["default"] if fg else BG
    if isinstance(name, str) and len(name) == 6 and all(c in "0123456789abcdefABCDEF" for c in name):
        return "#" + name.upper()
    return COLORS.get(name, COLORS["default"] if fg else BG)


def pick_font(size: int) -> ImageFont.FreeTypeFont:
    # Prefer fonts with broad Unicode coverage. The TUI leans on geometric
    # shapes (▸ ● ◐ ○ ✕ —) — a font that falls back to tofu for any of
    # these breaks the signature look.
    candidates = [
        "/home/cush/FontBase/IosevkaNerdFontMono-Regular.ttf",
        "/home/cush/FontBase/HackNerdFontMono-Regular.ttf",
        "/home/cush/FontBase/DejaVuSansMNerdFontMono-Regular.ttf",
        "/usr/share/fonts/TTF/DejaVuSansMono.ttf",
        "/usr/share/fonts/noto/NotoSansMono-Regular.ttf",
        "/usr/share/fonts/liberation/LiberationMono-Regular.ttf",
    ]
    for path in candidates:
        if os.path.exists(path):
            return ImageFont.truetype(path, size)
    raise RuntimeError("no monospace TTF found")


def render_png(screen: pyte.Screen, out: Path, font_size: int = 22) -> None:
    font = pick_font(font_size)
    bold_font = font  # keep monospaced; bold pulls glyph widths on some families.

    bbox = font.getbbox("M")
    cell_w = bbox[2] - bbox[0]
    ascent, descent = font.getmetrics()
    cell_h = ascent + descent + 2

    pad_x, pad_y = 24, 20
    width = pad_x * 2 + cell_w * screen.columns
    height = pad_y * 2 + cell_h * screen.lines

    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)

    for y in range(screen.lines):
        row = screen.buffer[y]
        for x in range(screen.columns):
            cell = row[x]
            ch = cell.data
            if not ch:
                continue
            fg = ansi_color(cell, fg=True)
            bg = ansi_color(cell, fg=False)
            px = pad_x + x * cell_w
            py = pad_y + y * cell_h
            if bg != BG:
                draw.rectangle([px, py, px + cell_w, py + cell_h], fill=bg)
            use_font = bold_font if cell.bold else font
            if ch.strip() or bg != BG:
                draw.text((px, py), ch, font=use_font, fill=fg)

    img.save(out, "PNG", optimize=True)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--stock", default=shutil.which("stock") or f"{os.environ.get('TMPDIR', '/tmp')}/stock")
    ap.add_argument("--out", default=str(REPO / "docs/hero.png"))
    ap.add_argument("--cols", type=int, default=96)
    ap.add_argument("--rows", type=int, default=30)
    ap.add_argument("--font-size", type=int, default=22)
    args = ap.parse_args()

    if not os.path.isfile(args.stock) or not os.access(args.stock, os.X_OK):
        print(f"stock binary not found or not executable: {args.stock}", file=sys.stderr)
        print("  build it with: make build && ./snapshot.py --stock ./stock", file=sys.stderr)
        return 1

    screen = capture_frame(args.stock, args.cols, args.rows)
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    render_png(screen, out, args.font_size)
    print(f"wrote {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
