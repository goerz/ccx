# Image Rendering with the Kitty Graphics Protocol

How ccx renders images from Claude sessions (screenshots, pasted images,
generated output) **inline** in the conversation preview, using the Kitty
graphics protocol, with a fallback to the system viewer for unsupported
terminals. This is the implementation behind the "Kitty image preview" feature
described in the [README](../README.md#conversation-view).

## Overview

Images are stored as base64 in the JSONL transcript. When the focused block in
the conversation preview is an image, ccx decodes it to a cached PNG and draws
it directly into the preview pane as a Kitty graphics layer — there is no modal
overlay. On terminals without Kitty support, pressing `Enter`/`i` on an image
opens it in the system viewer instead.

Supported terminals: kitty, WezTerm, ghostty (detected automatically, including
through an outer tmux session).

## Image cache (`internal/session/image.go`)

Decoded images are cached on disk so they don't have to be re-extracted on every
render:

- `ImageCachePath(home, sessionID, pasteID)` → `~/.claude/image-cache/<sessionID>/<pasteID>.png`
- `ExtractImageToTemp(...)` returns the cached path if present, otherwise decodes
  the base64 block from the JSONL, writes it to the cache, and returns the path.

`resolveImagePath()` in the TUI falls back to a scratch extraction when no cache
entry exists yet.

## Terminal detection (`internal/kitty/graphics.go`)

`kitty.Supported() bool` (cached for the process lifetime) decides whether to
draw inline. Detection order:

1. `CCX_KITTY=1` / `CCX_KITTY=0` — explicit override.
2. `TERM_PROGRAM` is `kitty`, `WezTerm`, or `ghostty`; or `TERM` begins with
   `xterm-kitty`.
3. `KITTY_WINDOW_ID` / `KITTY_PID` present.
4. Inside tmux (`$TMUX` set): `detectKittyViaTmux()` queries the tmux server
   environment (`tmux show-environment -g TERM_PROGRAM` / `KITTY_WINDOW_ID`).

When support is detected inside tmux, `ensureTmuxAllowPassthrough()` promotes the
tmux `allow-passthrough` option from `on` to `all`. Without `all`, tmux drops the
DCS passthrough sequence emitted from a hidden pane, so the clear sequence can
never reach the terminal and an image gets stuck on screen after switching tmux
windows.

## Graphics primitives (`internal/kitty/graphics.go`)

```go
ImageSize(path string) (width, height int)                  // pixel dimensions via image.DecodeConfig
FitSize(imgW, imgH, maxCols, maxRows int) (cols, rows int)  // aspect-preserving fit to a cell box
PlaceImage(path, row, col, cols, rows int) string           // escape sequence to draw at a cell position
DisplayImage(path string, cols, rows int) string            // draw at the cursor
ClearImages() string                                        // clear all placed images
```

`PaneOffset()` / `PaneVisible()` / `InvalidatePaneOffset()` track the pane's
position inside tmux so images land at the right screen cell. Sequences are
wrapped for tmux passthrough automatically when `$TMUX` is set.

## Inline rendering (`internal/tui/conversation.go`)

`kittyImageLayer()` produces the escape sequences appended to the rendered frame
(`screen += a.kittyImageLayer()` in `app.go`). It returns `kitty.ClearImages()`
when nothing should be drawn — when the terminal isn't focused (`termFocused`),
the view isn't the conversation, or the focused block isn't an image.

Two placements exist:

- **Conversation preview** — when the preview pane is focused and the block under
  the cursor is an image (`kittyImagePath()` / `kittyImageActive()`), the image is
  drawn into the left detail pane, centered and scaled to fit. The text tooltip is
  suppressed for that block so it doesn't overlap the image.
- **Artifact browser, Images page** (`p` → images) — pressing `i` toggles
  `convPageKitty`; when on, the selected image is drawn into the browser's right
  detail pane.

Sizing always goes through `ImageSize` + `FitSize`, so the aspect ratio is
preserved within the available pane (`maxCols` × `maxRows`).

## Fallback: external viewer

`openCachedImage()` in `app.go` is the non-Kitty path: it resolves the cached
image path and launches the macOS `open` command. It is reached by pressing
`Enter`/`i` on an image block in the conversation or detail view, and by `o`
(open) in the artifact browser's Images page. This always works regardless of
terminal support.

## Limitations

- One focused image is drawn at a time.
- Inline rendering requires a Kitty-graphics-capable terminal; otherwise the
  external viewer is used.
- The external-viewer fallback uses macOS `open`; other platforms get no inline
  rendering and no viewer launch.
