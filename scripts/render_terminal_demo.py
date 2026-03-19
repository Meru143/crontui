#!/usr/bin/env python3
"""
Generate a short README/release terminal demo GIF for CronTUI.

This renderer creates a deterministic terminal-style animation from curated
frames, which keeps the README asset stable even when live terminal capture
tooling is unavailable on the current machine.
"""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent
OUTPUT = ROOT / "media" / "demo" / "crontui-native-windows.gif"
FONT_PATHS = [
    Path(r"C:\WINDOWS\Fonts\consola.ttf"),
    Path(r"C:\WINDOWS\Fonts\consolab.ttf"),
]

WIDTH = 1400
HEIGHT = 760
PADDING_X = 42
PADDING_Y = 34
LINE_HEIGHT = 32

BG = "#0d1117"
PANEL = "#161b22"
BORDER = "#30363d"
TEXT = "#c9d1d9"
MUTED = "#8b949e"
PROMPT = "#7ee787"
ACCENT = "#58a6ff"
TITLE = "#f0f6fc"


FRAMES = [
    {
        "duration": 1200,
        "command": None,
        "body": [
            ("title", "CronTUI native Windows demo"),
            ("muted", "Add a managed task, verify it in Task Scheduler, create a backup."),
            ("blank", ""),
            ("text", "Environment: native Windows Task Scheduler"),
            ("text", "Scope: one short CLI workflow for the README and release page"),
        ],
    },
    {
        "duration": 1800,
        "command": r'.\crontui.exe add "@hourly" whoami --desc "hourly identity"',
        "body": [
            ("text", "Added job #1: @hourly whoami"),
        ],
    },
    {
        "duration": 2200,
        "command": r".\crontui.exe list",
        "body": [
            ("text", "ID  Status  Schedule  Command  Description"),
            ("muted", "--  ------  --------  -------  -----------"),
            ("text", "1   ON      @hourly   whoami   hourly identity"),
        ],
    },
    {
        "duration": 2200,
        "command": r'''powershell -NoProfile -Command "Get-ScheduledTask | Where-Object TaskName -eq 'job-1' | Select-Object TaskName,TaskPath,State"''',
        "body": [
            ("text", "TaskName TaskPath       State"),
            ("muted", "-------- --------       -----"),
            ("text", r"job-1    \CronTUI-Demo\ Ready"),
        ],
    },
    {
        "duration": 2200,
        "command": r".\crontui.exe backup",
        "body": [
            ("text", r"Backup created: .tmp\readme-demo\taskscheduler_20260319_191321.bak"),
        ],
    },
]


def load_font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for path in FONT_PATHS:
        if path.exists():
            return ImageFont.truetype(str(path), size=size)
    return ImageFont.load_default()


FONT = load_font(22)
FONT_BOLD = load_font(24)


def terminal_frame(command: str | None, body: list[tuple[str, str]]) -> Image.Image:
    image = Image.new("RGB", (WIDTH, HEIGHT), BG)
    draw = ImageDraw.Draw(image)

    panel_left = 28
    panel_top = 28
    panel_right = WIDTH - 28
    panel_bottom = HEIGHT - 28

    draw.rounded_rectangle(
        (panel_left, panel_top, panel_right, panel_bottom),
        radius=18,
        fill=PANEL,
        outline=BORDER,
        width=2,
    )

    header_bottom = panel_top + 56
    draw.rounded_rectangle(
        (panel_left, panel_top, panel_right, header_bottom),
        radius=18,
        fill="#0f141b",
        outline=BORDER,
        width=2,
    )
    draw.rectangle(
        (panel_left, header_bottom - 18, panel_right, header_bottom),
        fill="#0f141b",
        outline="#0f141b",
    )

    dots = ["#ff5f57", "#febc2e", "#28c840"]
    for i, color in enumerate(dots):
        cx = panel_left + 24 + i * 22
        cy = panel_top + 28
        draw.ellipse((cx - 6, cy - 6, cx + 6, cy + 6), fill=color)

    draw.text(
        (panel_left + 90, panel_top + 16),
        "CronTUI terminal demo",
        fill=MUTED,
        font=FONT,
    )

    x = panel_left + PADDING_X
    y = header_bottom + PADDING_Y

    if command is not None:
        prompt = r"PS C:\Users\merup\Downloads\crontui> "
        draw.text((x, y), prompt, fill=PROMPT, font=FONT)
        prompt_width = draw.textlength(prompt, font=FONT)
        draw.text((x + prompt_width, y), command, fill=TEXT, font=FONT)
        y += LINE_HEIGHT + 6

    for kind, line in body:
        if kind == "blank":
            y += LINE_HEIGHT // 2
            continue
        fill = TEXT
        font = FONT
        if kind == "muted":
            fill = MUTED
        elif kind == "title":
            fill = TITLE
            font = FONT_BOLD
        elif kind == "accent":
            fill = ACCENT
        draw.text((x, y), line, fill=fill, font=font)
        y += LINE_HEIGHT

    return image


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    frames = [terminal_frame(frame["command"], frame["body"]) for frame in FRAMES]
    durations = [frame["duration"] for frame in FRAMES]
    frames[0].save(
        OUTPUT,
        save_all=True,
        append_images=frames[1:],
        duration=durations,
        loop=0,
        optimize=False,
        disposal=2,
    )
    print(f"Generated {OUTPUT}")


if __name__ == "__main__":
    main()
