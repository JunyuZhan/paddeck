#!/usr/bin/env python3
"""Generate one CSS file per palette from the master stylesheet.

Safari 10 has no CSS custom properties, so themes are shipped as complete
stylesheets rendered from web/style.css (which is kept in the Mocha palette).

Run:  python3 tools/themes.py
"""
import os
import re

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WEB = os.path.join(ROOT, "web")

# role -> hex currently present in web/style.css (Catppuccin Mocha)
MOCHA = {
    "bg":       "#1e1e2e",
    "panel":    "#313244",
    "border":   "#45475a",
    "hair":     "#3d3f54",
    "offbg":    "#26263a",
    "warnbg":   "#3a3340",
    "critbg":   "#3b2430",
    "track":    "#2a2a3e",
    "dotoff":   "#585b70",
    "text":     "#cdd6f4",
    "name":     "#e0e6fb",
    "num":      "#eef1ff",
    "sub1":     "#bac2de",
    "sub0":     "#a6adc8",
    "ov1":      "#7f849c",
    "ov0":      "#6c7086",
    "ov2":      "#9399b2",
    "a1":       "#cba6f7",
    "a2":       "#94e2d5",
    "a3":       "#f5c2e7",
    "a4":       "#b4befe",
    "a5":       "#fab387",
    "a6":       "#89dceb",
    "ok":       "#a6e3a1",
    "warn":     "#f9e2af",
    "crit":     "#f38ba8",
    "sh1":      "#2a2840",
    "sh2":      "#26243a",
    "sh3":      "#232136",
}

# rgb triplets used inside rgba() in the master stylesheet
RGB_ROLES = ["a1", "a2", "a3", "a4", "a5", "a6", "ok", "warn", "crit", "bg", "hair", "panel"]


def hexrgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


def tohex(t):
    return "#%02x%02x%02x" % t


def mix(a, b, k):
    """k of b over a"""
    ra, ga, ba = hexrgb(a)
    rb, gb, bb = hexrgb(b)
    return tohex((int(ra + (rb - ra) * k), int(ga + (gb - ga) * k), int(ba + (bb - ba) * k)))


def dark_shadows(panel):
    return mix(panel, "#000000", 0.16), mix(panel, "#000000", 0.22), mix(panel, "#000000", 0.28)


def build(p):
    """Fill derived roles for one palette dict."""
    out = dict(p)
    out.setdefault("warnbg", mix(p["panel"], p["warn"], 0.12))
    out.setdefault("critbg", mix(p["panel"], p["crit"], 0.13))
    if "sh1" not in out:
        out["sh1"], out["sh2"], out["sh3"] = dark_shadows(p["panel"])
    return out


THEMES = []


def theme(tid, name, desc, pal):
    THEMES.append((tid, name, desc, build(pal)))


theme("mocha", "MOCHA 紫灰", "Catppuccin · 冷紫底 + 淡紫强调", {
    "bg": "#1e1e2e", "panel": "#313244", "border": "#45475a", "hair": "#3d3f54",
    "offbg": "#26263a", "track": "#2a2a3e", "dotoff": "#585b70",
    "text": "#cdd6f4", "name": "#e0e6fb", "num": "#eef1ff", "sub1": "#bac2de",
    "sub0": "#a6adc8", "ov1": "#7f849c", "ov0": "#6c7086", "ov2": "#9399b2",
    "a1": "#cba6f7", "a2": "#94e2d5", "a3": "#f5c2e7", "a4": "#b4befe",
    "a5": "#fab387", "a6": "#89dceb",
    "ok": "#a6e3a1", "warn": "#f9e2af", "crit": "#f38ba8",
})

theme("rosepine", "ROSE 梅子", "Rosé Pine Moon · 深梅底 + 玫瑰金", {
    "bg": "#232136", "panel": "#2a273f", "border": "#44415a", "hair": "#393552",
    "offbg": "#1f1d30", "track": "#1f1d30", "dotoff": "#524f67",
    "text": "#e0def4", "name": "#eae7f8", "num": "#f4f2ff", "sub1": "#b8b3d4",
    "sub0": "#908caa", "ov1": "#7e7a9a", "ov0": "#6e6a86", "ov2": "#908caa",
    "a1": "#c4a7e7", "a2": "#9ccfd8", "a3": "#ea9a97", "a4": "#eb6f92",
    "a5": "#f6c177", "a6": "#3e8fb0",
    "ok": "#95b1ac", "warn": "#f6c177", "crit": "#eb6f92",
})

theme("gruvbox", "GRUVBOX 暖褐", "Gruvbox Dark · 暖炭灰 + 奶油字", {
    "bg": "#282828", "panel": "#3c3836", "border": "#504945", "hair": "#453f3a",
    "offbg": "#32302f", "track": "#32302f", "dotoff": "#665c54",
    "text": "#ebdbb2", "name": "#f2e5c2", "num": "#fbf1c7", "sub1": "#d5c4a1",
    "sub0": "#bdae93", "ov1": "#a89984", "ov0": "#928374", "ov2": "#bdae93",
    "a1": "#d3869b", "a2": "#8ec07c", "a3": "#83a598", "a4": "#fe8019",
    "a5": "#fabd2f", "a6": "#b8bb26",
    "ok": "#b8bb26", "warn": "#fabd2f", "crit": "#fb4934",
})

theme("nord", "NORD 冷蓝", "Nord · 冷蓝灰 + 极光色", {
    "bg": "#2e3440", "panel": "#3b4252", "border": "#4c566a", "hair": "#434c5e",
    "offbg": "#292e39", "track": "#2e3440", "dotoff": "#4c566a",
    "text": "#d8dee9", "name": "#e5e9f0", "num": "#eceff4", "sub1": "#c0c8d6",
    "sub0": "#a3acbd", "ov1": "#8d97a8", "ov0": "#7b8496", "ov2": "#a3acbd",
    "a1": "#b48ead", "a2": "#8fbcbb", "a3": "#bf616a", "a4": "#81a1c1",
    "a5": "#d08770", "a6": "#88c0d0",
    "ok": "#a3be8c", "warn": "#ebcb8b", "crit": "#bf616a",
})

theme("latte", "LATTE 亮色", "Catppuccin Latte · 白天/强光版", {
    "bg": "#e6e9ef", "panel": "#eff1f5", "border": "#ccd0da", "hair": "#dce0e8",
    "offbg": "#dce0e8", "track": "#dce0e8", "dotoff": "#bcc0cc",
    "text": "#4c4f69", "name": "#3c3f58", "num": "#26293f", "sub1": "#5c5f77",
    "sub0": "#6c6f85", "ov1": "#8c8fa1", "ov0": "#9ca0b0", "ov2": "#acb0be",
    "a1": "#8839ef", "a2": "#179299", "a3": "#ea76cb", "a4": "#7287fd",
    "a5": "#fe640b", "a6": "#04a5e5",
    "ok": "#40a02b", "warn": "#df8e1d", "crit": "#d20f39",
    "sh1": "#d9dce5", "sh2": "#d3d7e1", "sh3": "#cdd1dc",
})


def render(master, pal):
    css = master
    # protect every Mocha literal with a role placeholder
    for role, val in MOCHA.items():
        css = css.replace(val, "\x00%s\x00" % role)
    for role in RGB_ROLES:
        r, g, b = hexrgb(MOCHA[role])
        css = css.replace("rgba(%d,%d,%d," % (r, g, b), "rgba(\x00%s.r\x00," % role)
    # paint the palette
    for role, val in pal.items():
        css = css.replace("\x00%s\x00" % role, val)
        r, g, b = hexrgb(val)
        css = css.replace("\x00%s.r\x00" % role, "%d,%d,%d" % (r, g, b))
    left = re.findall(r"\x00[^\x00]+\x00", css)
    if left:
        raise SystemExit("unresolved tokens: %s" % set(left))
    return css


def main():
    master = open(os.path.join(WEB, "style.css")).read()
    js = ["/* generated by tools/themes.py - do not edit */", "window.THEMES = ["]
    for tid, name, desc, pal in THEMES:
        out = os.path.join(WEB, "theme-%s.css" % tid)
        open(out, "w").write(render(master, pal))
        js.append(
            '  { id: "%s", name: "%s", desc: "%s", bg: "%s", panel: "%s", border: "%s",'
            ' text: "%s", sub: "%s", sub1: "%s", ov0: "%s",'
            ' a1: "%s", a2: "%s", a3: "%s", a4: "%s", a5: "%s", a6: "%s",'
            ' ok: "%s", warn: "%s", crit: "%s", dot: "%s" },'
            % (tid, name, desc, pal["bg"], pal["panel"], pal["border"],
               pal["text"], pal["sub0"], pal["sub1"], pal["ov0"],
               pal["a1"], pal["a2"], pal["a3"], pal["a4"], pal["a5"], pal["a6"],
               pal["ok"], pal["warn"], pal["crit"], pal["dotoff"])
        )
    js.append("];")
    open(os.path.join(WEB, "themes.js"), "w").write("\n".join(js) + "\n")
    print("wrote %d themes" % len(THEMES))


if __name__ == "__main__":
    main()
