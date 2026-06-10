# ccx — Claude Code Explorer

A terminal UI for browsing, inspecting, and managing [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

Browse sessions, read conversations, inspect tool calls, view agent hierarchies, explore configs/plugins, and get aggregated stats — all from your terminal.

![ccx demo](docs/gifs/01-browse.gif)

> More demos: [conversation](docs/DEMOS.md#conversation), [command mode](docs/DEMOS.md#command-mode), [views](docs/DEMOS.md#views), [URL/file actions](docs/DEMOS.md#actions), [sandbox testing](docs/DEMOS.md#sandbox)

## Install

This program is best installed from source:

```bash
git clone https://github.com/goerz/ccx.git
cd ccx
make build      # -> bin/ccx
make install    # -> ~/.local/bin/ccx
```

Or, install into `~/go/bin/ccx`

```bash
go install github.com/goerz/ccx@latest
```


## Usage

```bash
ccx                        # launch TUI
ccx -view config           # start in config explorer
ccx -view stats            # start in global stats
ccx -view plugins          # start in plugin explorer
ccx -group tree            # start with tree grouping
ccx -preview stats         # start with stats preview open
ccx -search "is:live"      # start filtered to live sessions
ccx -here                  # scope to sessions in the current directory
```

### `ccx sessions -pick`

Interactive session resolver for shells, scripts, and agents. Launches the full `ccx` TUI on **stderr**; stdout is reserved for JSON.

To confirm a pick, press `P`. Navigate with arrows, multi-select with `space`, filter with `/`.

```bash
# basic usage
sid=$(ccx sessions -pick | jq -r '.sessions[0].id')
claude --resume "$sid"

# narrow with filter query (same syntax as the TUI `f` filter)
ccx sessions -pick -search "is:current is:live"

# multi-select
ccx sessions -pick -multi | jq '.sessions | length'
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-search STR` | Initial filter query (same syntax as the TUI `f` filter) |
| `-multi` | Allow multi-selection (space to toggle, `P` to confirm) |
| `-dir PATH` | Claude data directory (default: `~/.claude`) |

**Output schema (stable):**

```json
{
  "sessions": [
    {
      "id": "a1b2c3…",
      "project_root_path": "/Users/edgar/code/...",
      "transcript_path": "/Users/edgar/.claude/projects/-Users-.../a1b2c3….jsonl"
    }
  ]
}
```

`sessions` is always an array; single-select yields length 1.

**Exit codes:**

| Code | Condition |
|------|-----------|
| 0    | User confirmed; JSON printed to stdout |
| 1    | Internal error (stderr message) |
| 2    | No candidates after filtering |
| 130  | User cancelled (Esc / Ctrl-C) |


### CLI Flags

| Flag | Description |
|------|-------------|
| `-version`, `-v` | Print version and exit |
| `-dir PATH` | Claude data directory (default: `~/.claude`) |
| `-view MODE` | Initial view: `sessions`, `config`, `plugins`, `stats` |
| `-group MODE` | Initial grouping: `flat`, `proj`, `tree`, `chain`, `fork` |
| `-preview MODE` | Initial preview: `conv`, `stats`, `mem`, `tasks` |
| `-search QUERY` | Start with session filter applied |
| `-session ID` | Open a specific session by ID (prefix match) |
| `-here` | Scope the list to sessions in the current directory |
| `-tmux` | Enable tmux integration (auto-detected) |
| `-tmux-auto-live` | Auto-enter live session in same tmux window |
| `-worktree-dir NAME` | Worktree subdirectory name (default: `.worktree`) |
| `-theme MODE` | Color scheme: `auto`, `light`, `dark` (overrides config) |

The Claude data directory is resolved in order: `--dir` flag → `CLAUDE_CONFIG_DIR` env → `~/.claude`.

The color scheme is resolved in order: `-theme` flag → `theme` in config → `auto`. In `auto` mode ccx defaults to the light palette and only switches to dark when it has positive evidence of a dark background (a `COLORFGBG` hint, or a direct terminal that reports a dark background via OSC 11). Inside tmux/screen the background query is unreliable, so `auto` stays light there — set `theme: dark` (or `-theme dark`) to pin a dark scheme.

## Views

ccx is organized into a handful of top-level **views** — the Session Browser, Conversation, Detail, Global Stats, Config Explorer, and Plugin Explorer. You switch between them with the keys noted in each section below; `Esc` returns to the previous view.

### Split panes and the preview

Most views are a **split pane**: a scrollable **list** on the left and a **preview pane** on the right. The preview pane is part of the current view — opening it does *not* switch views.

The preview pane has three states, driven entirely by the arrow keys:

| State | How to reach it | Keys act on |
|-------|-----------------|-------------|
| **Closed** | starting state; `←` again with the list focused | the list |
| **Open, list focused** | `→` once | the list (the preview just follows the cursor) |
| **Open, preview focused** | `→` again | the preview pane |

`←` reverses the sequence: it unfocuses the preview, then closes it, then (in the Conversation view) steps back a level. So in the Session Browser, pressing `→` once opens the preview beside the list; a second `→` moves focus *into* it.

Which side is focused changes what many keys do — for example, in the Session Browser `Tab` cycles the **group mode** when the list is focused and the **preview mode** when the preview is focused, and the `1`–`9` [number shortcuts](#number-key-shortcuts) are scoped to the focused side. `[` and `]` adjust the split ratio.

### Session Browser

The default view ccx opens to. Browse all Claude Code sessions across projects, sorted by recency.

- **Status badges** — at-a-glance session state (see [Session Badges](#session-badges))
- **Cross-session search** (`/`) — full-text search across all session contents
- **Filter** (`f`) — narrow the visible list by project, branch, prompt, window name, tags, or `is:`/`has:` tokens
- **Group modes** (`Tab`/`Shift-Tab` when the list is focused, or `:group:*`):
  - **Flat** — simple list sorted by time
  - **Project** — clustered by project path
  - **Tree** — team hierarchy with leader/teammate nesting
  - **Chain** — resume-chain grouping (parent → child)
  - **Fork** — agent-fork grouping
  - **Repo** — clustered by base git repository
- **Current-directory scope** (`.`) — toggle restricting the list to sessions in the current working directory (start scoped with `ccx -here`)
- **Preview pane** (`→`, see [Split panes and the preview](#split-panes-and-the-preview)) — `Tab`/`Shift-Tab` (or `:preview:*`) cycles the preview mode: conversation, stats, memory, tasks/plan, agents, contexts, live
- **Multi-select** (`Space`) — bulk delete, copy paths, send input
- **Actions menu** (`x`) — delete, move, resume, fork, copy path, worktree, kill, input, jump, URLs, files
- **Command mode** (`:`) — vim-style commands with fuzzy suggestions

#### Search Filters

| Filter | Matches |
|--------|---------|
| `is:here` | In the current tmux window |
| `is:live` | Running Claude process |
| `is:busy` | Actively responding |
| `is:bg` | Background work in flight (shell/Monitor/cron) |
| `is:wait` | Idle with unfinished todos/tasks |
| `is:done` | All todos/tasks completed |
| `is:stuck` | Live but stale with unfinished work |
| `is:wt` | In a git worktree |
| `is:team` | Part of a team session |
| `is:fork` | Forked from another session |
| `is:remote` | Remote session (experimental) |
| `is:current` | Project path matches the directory ccx was launched from |
| `has:mem` | Has memory file |
| `has:todo` | Has todos |
| `has:task` | Has tasks |
| `has:plan` | Has plan |
| `has:agent` | Has subagents |
| `has:compact` | Uses message compaction |
| `has:skill` | Used skills |
| `has:mcp` | Used MCP tools |
| `proj:NAME` | Filter by project name |
| `team:NAME` | Filter by team name |
| `win:NAME` | Filter by tmux window name |
| `tag:NAME` | Filter by custom tag/badge |

Plain text terms match against project path, name, branch, session ID, first prompt, and teammate name. Multiple terms are AND-matched (all must match).

#### Session Badges

Each session row carries two kinds of badges. Independent badges can co-occur; lifecycle badges are mutually exclusive (highest-priority one wins).

**Independent:**

- `[HERE]` — session belongs to the current tmux window
- `[LIVE]` — a Claude process is attached to the session
- `[R·exp]` — remote session (experimental; see [docs/remote-execution.md](docs/remote-execution.md))
- Custom tags — user-applied via `x` → `t` (see [docs/CUSTOM_BADGES.md](docs/CUSTOM_BADGES.md))

**Lifecycle** (priority high → low; at most one shown):

| Badge | When |
|-------|------|
| `[BUSY]` | Claude is actively responding (JSONL written within ~10s) |
| `[BG]` | Live session has a shell/Monitor job, or any cron is `active` |
| `[STUCK]` | Live, JSONL stale for >30min, and unfinished todos/tasks exist |
| `[WAIT]` | Live, idle, with unfinished todos/tasks |
| `[DONE]` | Session had todos/tasks and all are completed |

Example: `[HERE][LIVE][WAIT] my-feature` — current window, live process, idle with pending work.

### Cross-Session Search

From the session browser, press `/` (or `:search`) to search inside conversation content across all sessions.

**Search syntax:**

- `word1 word2` — AND match (all terms must appear)
- `"exact phrase"` — Exact phrase matching
- `-exclude` — Exclude terms from results
- `user:` — Only search user messages
- `assistant:` — Only search assistant responses
- `tool:ToolName` — Only search specific tool calls

**Features:**

- Searches text, tool inputs/outputs, thinking blocks
- Results stream in real-time as they're found
- Matched terms are highlighted in snippets
- Press `Enter` to jump directly to the matching message
- Press `/` to edit the query

**Example queries:**

```
database migration                    # Find both terms
"how do I" API                        # Phrase + term
user: error -test                     # User messages with "error", excluding "test"
assistant: "I recommend" -deprecated  # Complex combination
```

### Conversation View

From the session browser, press `Enter` on a session to drill in and read the full conversation.

- **Preview pane** (`→`, see [Split panes and the preview](#split-panes-and-the-preview)) — foldable message detail at three levels (`Tab`/`Shift-Tab` cycles when the preview is focused):
  - **Compact** — text blocks only
  - **Standard** — text + per-turn artifact list (images, files, changes, URLs)
  - **Verbose** — text + tool blocks + full hook details
- **Kitty image preview** — inline image rendering in the left pane for Kitty-compatible terminals (kitty, WezTerm, ghostty). Aspect-ratio-preserving, auto-detected inside tmux (see [docs/image-rendering.md](docs/image-rendering.md)).

![Kitty image preview](docs/gifs/08-kitty-image-preview.png)

- **Block navigation** (`↑`/`↓`) — navigate text, tool calls, and results
- **Fold/unfold** (`←`/`→`, `f`/`F`) — collapse/expand content blocks
- **System tag folding** — `<system-reminder>`, `<task-notification>`, `<available-deferred-tools>`, etc. are folded by default, expandable on demand
- **Block filter** (`/`, when preview focused) — filter by `is:tool`, `is:hook`, `is:error`, `is:skill`, `tool:Name`
- **Subagent drill-down** (`Enter` on agent) — recursive navigation into sub-sessions with back-stack
- **Side-question context** — background context from parent sessions is collapsed into a summary; only the actual question/answer is shown
- **Artifact browser** (`p`) — list a session's URLs, files, images, changes, and context tree; open, edit, or copy each
- **Live tail** (`L`) — auto-follow active sessions in real-time
- **Send input** (`I`) — send text to running Claude via tmux
- **Jump to pane** (`J`) — switch to the tmux pane running the session

#### Subagent Support

Subagents are displayed inline in the conversation with type badges:

| Type | Badge | Source |
|------|-------|--------|
| `aside_question` | `?` `:btw` | Side-question (background Q&A) |
| `Explore` | `⊕ Explore` | Codebase exploration agent |
| `general-purpose` | `⊕ general-purpose` | Default agent |
| Custom types | `⊕ {type}` | From `agent-*.meta.json` |

Agent type detection: reads `agent-{id}.meta.json` (preferred) or parses type from filename `agent-{type}-{hash}.jsonl`. Auto-compaction files (`agent-acompact-*.jsonl`) are excluded.

Timestamp ordering uses the **last message** in the subagent file (most recent activity), not the first.

### Detail View

From the conversation view, press `Enter` on a message to open this full-screen viewer with block-level navigation.

- **Block cursor** (`↑`/`↓`) — navigate between blocks
- **Fold/unfold** (`←`/`→`, `f`/`F`) — collapse/expand blocks
- **Message navigation** (`n`/`N` or `]`/`[`) — step through messages
- **Block filter** (`/`) — filter by `is:tool`, `is:hook`, `tool:Name`, etc.
- **Copy mode** (`v`) — line-by-line selection with anchor/cursor, vim-style navigation
- **Clipboard** (`y`) — copy selected blocks to system clipboard
- **Actions menu** (`x`) — extract URLs, files, changes, or copy

### Global Stats (`v` → `s`)

From the session browser, press `v` then `s` (or `:view:stats`) to open aggregated metrics across all sessions with detail drill-down.

Press `p` for the page menu, then a letter to drill in:

- **Overview** (`o`) — total sessions, messages, tokens, duration, cost
- **Tools** (`t`) — built-in tool usage with timeline sparklines
- **MCP Tools** (`m`) — MCP tool usage with error tracking
- **Agents** (`a`) — agent type breakdown (Explore, general-purpose, etc.)
- **Skills** (`s`) — skill usage with per-skill error counts
- **Commands** (`c`) — command usage with per-command error counts
- **Errors** (`e`) — error breakdown by tool/skill/command category
- **Repos** (`r`) / **Projects** (`p`) — activity grouped by base repo or project path

Metrics tracked per session: token usage (input/output/cache per model), code activity (write/edit/read/bash counts), files touched, tool call timelines, message timing gaps, model switches, compaction events, hook invocations, and turns per request.

### Config Explorer (`v` → `c`)

From the session browser, press `v` then `c` (or `:view:config`) to browse and manage all Claude Code configuration files.

- **Category filter** (`Tab`) — global, project, local, skills, agents, commands, MCP, hooks, enterprise
- **Split preview** — file content with syntax awareness
- **Multi-select** (`Space`) — select configs for testing
- **Test env** (`t`) — launch isolated Claude session with only selected configs
- **Edit** (`e` / `Enter`) — open in `$EDITOR`
- **Actions menu** (`x`) — edit, copy path, open shell at path

Categories discovered:
- **Global** — `~/.claude/CLAUDE.md` + memory, contexts, rules (with `@reference` walking)
- **Project** — project-level `CLAUDE.md` + memory from `projects/{encoded}/memory/`
- **Local** — parent CLAUDE.md files found by walking up from project directory
- **Skills/Agents/Commands** — plugin component configs
- **MCP** — MCP server configurations
- **Hooks** — hook definitions
- **Enterprise** — managed enterprise settings

#### Config Test Environment

The test environment (`t` key) creates an isolated Claude Code session with only the selected configs active:

1. Creates a temporary `HOME` directory
2. Symlinks only the selected memory/config files
3. Preserves editor config (`.config/`, shell dotfiles)
4. Extracts OAuth credentials from macOS keychain for connector MCP access
5. Launches `claude` with the isolated environment
6. Supports git worktree detection

This lets you test specific config combinations without affecting your main setup.

### Plugin Explorer (`v` → `p`)

From the session browser, press `v` then `p` (or `:view:plugins`) to browse installed Claude Code plugins and their components.

- **Component drill-down** (`Enter`) — view plugin agents, skills, commands, hooks, MCP servers
- **Multi-select** (`Space`) — select components for batch editing
- **Edit** (`e`) — open component files in `$EDITOR`
- **Actions menu** (`x`) — edit, copy path, open shell
- **Component badges** — e.g. `[3a 2s 1c]` = 3 agents, 2 skills, 1 command
- **Status badges** — DISABLED, BLOCKED (with reasons from blocklist)

Plugin discovery reads from:
- `installed_plugins.json` — install paths and versions
- `blocklist.json` — blocked plugins with reasons
- `known_marketplaces.json` — marketplace metadata (git/github sources)
- `settings.json` — `enabledPlugins` list
- `.claude-plugin/` — component directories per plugin

Component types: agents (`.md`), skills (`.md`), commands (`.md`), hooks (`.py`/`.sh`), MCP servers (`.json`), LSP servers, scripts, settings, memory, references.

#### Plugin Test Environment

Multi-select plugin components and press `t` to launch an isolated Claude session with only the selected plugins active. Uses the same isolated HOME mechanism as the config test environment.

## Keybindings

Navigation is shared across all views: arrow keys (or vim `h`/`j`/`k`/`l`), `pgup`/`pgdn` (or `ctrl+b`/`ctrl+f`), and `g`/`G` for top/bottom. `Esc` goes back or closes; `q` quits; `?` opens context help. Every key is remappable — see [Keybindings](#keybindings-1) under Configuration.

### Sessions

| Key | Action |
|-----|--------|
| `Enter` | Open conversation (jump to message when preview focused) |
| `→` / `←` | Open/focus preview · close/unfocus preview |
| `Tab` / `Shift+Tab` | Cycle group mode (list focused) · cycle preview detail (preview focused) |
| `[` / `]` | Adjust split ratio |
| `g` / `G` | Jump to top / bottom |
| `.` | Toggle current-directory scope |
| `Space` | Toggle multi-select |
| `1`–`9` | Number-key shortcuts (preview/page, per focus side) |
| `f` | Filter the list (path, name, `is:`/`has:` tokens) |
| `/` | Cross-session content search |
| `x` | Actions menu (delete, move, resume, fork, URLs, ...) |
| `v` | Views menu (stats/config/plugins) |
| `e` | Edit session files |
| `L` | Live preview (tmux) |
| `:` | Command mode |
| `R` | Refresh |

### Conversation

| Key | Action |
|-----|--------|
| `Enter` | Open message detail · drill into agent/task |
| `Tab` / `Shift+Tab` | Toggle flat/tree (list) · cycle compact/standard/verbose (preview) |
| `→` / `←` | Focus preview · back/close |
| `↑` / `↓` | Move cursor (list) · navigate blocks (preview) |
| `←` / `→` | Fold/unfold block (preview focused) |
| `f` / `F` | Fold all / expand all (preview focused) |
| `Space` | Select block (preview focused) |
| `v` | Copy mode · `y`/`Enter` copy selection |
| `/` | Search messages (list) · filter blocks (preview) |
| `J` | Jump to tree / origin message / tmux pane |
| `p` | Artifact browser (URLs, files, images, changes, contexts) |
| `e` | Edit menu (session/agent JSONL, export conversation as text) |
| `x` | Actions menu (URLs, files, changes, copy) |
| `L` | Toggle live tail |
| `I` | Send input to live session (tmux) |
| `i` | Open selected image |
| `t` | Toggle tooltips |
| `R` | Refresh |

### Detail (full-screen message)

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate blocks |
| `←` / `→` | Fold / unfold block |
| `n` / `N` (or `]` / `[`) | Next / previous message |
| `f` / `F` | Fold all / expand all |
| `Space` | Select block |
| `v` | Copy mode · `y` copy selection |
| `/` | Filter / search blocks |
| `x` | Actions menu (URLs, files, changes, copy) |
| `Enter` / `i` | Open image · drill into agent |
| `L` | Toggle live tail |
| `R` | Refresh |

### Command Mode (`:`)

Available from any view; suggestions are context-aware. Short aliases are in parentheses; commands can be chained (`view:config page:hooks`).

| Command | View | Action |
|---------|------|--------|
| `view:sessions\|stats\|config\|plugins` (`v:…`) | All | Switch view |
| `view:stats:tools\|mcp\|agents\|skills\|commands\|errors` | All | Stats detail page |
| `view:config:hooks` (`v:hooks`) | All | Config → hooks filter |
| `group:flat\|proj\|tree\|chain\|fork\|repo` (`g:…`) | Sessions | Change grouping |
| `preview:conv\|stats\|mem\|tasks\|agents\|contexts\|live` (`p:…`) | Sessions | Change preview |
| `detail:compact\|standard\|verbose` (`d:…`) | Conversation | Set detail level |
| `page:memory\|project\|skills\|hooks\|mcp\|keymaps\|shortcuts\|…` (`p:…`) | Config | Filter category |
| `page:overview\|tools\|errors` | Stats | Switch stats page |
| `set:ratio N` (`ratio`) | Sessions | Set split ratio (15–85) |
| `set:worktree-dir NAME` (`wt:dir`) | Sessions | Set worktree subdirectory |
| `badge:toggle KEY` (`bt`) | Sessions | Toggle badge visibility (HERE,LIVE,BUSY,BG,WAIT,DONE,STUCK) |
| `badge:rm KEY` | Sessions | Remove a badge from all sessions |
| `refresh` (`R`) | Sessions | Reload sessions |
| `search` (`find`, `grep`) | All | Cross-session content search |
| `config:edit` (`cfg:edit`, `km:edit`) | All | Edit config file |

## Configuration

Config file: `~/.config/ccx/config.yaml` (bootstrap with `:config:edit`)

The config file contains these sections:

### Theme

```yaml
theme: auto   # auto|light|dark — color scheme for the terminal background
```

`auto` defaults to light and only uses dark when it detects a dark background (unreliable inside tmux/screen, where it stays light); `light`/`dark` force the corresponding palette. The `-theme` flag overrides this setting.

### Keybindings

Every binding is remappable. User values merge over the defaults, so you only list the keys you want to change. The sections are `session`, `actions`, `views`, `conversation`, `preview`, and `navigation` (extra aliases that supplement the built-in arrow keys). Run `:config:edit` to see all options.

```yaml
session:
  quit: q
  open: enter
  actions: x
  filter: f
  search: /
actions:
  delete: d
  fork: F
  import_mem: M
  remove_mem: X
navigation:                 # vim/emacs aliases on top of the arrow keys
  up: [k]
  down: [j]
  left: [h]
  right: [l]
  page_up: [ctrl+b]
  page_down: [ctrl+f]
  home: [g]
  end: [G]
```

### Preferences (auto-saved on quit)

```yaml
preferences:
  group_mode: flat          # flat|proj|tree|chain|fork
  preview_mode: stats       # conv|stats|mem|tasks|live
  view_mode: sessions       # sessions|config|plugins|stats
  conv_detail_level: 1      # 0=compact, 1=standard, 2=verbose
  split_ratio: 35           # 15-85
  worktree_dir: .worktree   # git worktree subdirectory name
  hidden_badges: [DONE, STUCK]  # hide specific badges
  filter_term: "is:live"    # last applied session filter
  editor_input: true        # prefer $EDITOR for live input (ctrl+e to toggle)
```

### Claude command template

Configure the local Claude command used by session resume/new-session, tmux windows,
plugin commands, and config/plugin test popups:

```yaml
claude:
  command_template: "claude {{args}}"
```

`{{args}}` expands to the arguments supplied by ccx, such as `--resume <session-id>`
or `plugin install <id>`. If `{{args}}` is omitted, ccx appends its arguments at
the end. The template is parsed into argv and is not shell-evaluated for normal
process launches; tmux/script launches shell-quote the rendered argv.

Examples:

```yaml
claude:
  command_template: "ccproxy -- claude {{args}}"
```

```yaml
claude:
  command_template: "claude --model opus {{args}}"
```

### Number Key Shortcuts

Number keys `1-9` trigger commands based on the active view and split focus side.
Configure in the `shortcuts` section:

```yaml
shortcuts:
  sessions:
    left:                     # session list focused
      "1": "preview:conv"
      "2": "preview:stats"
      "3": "preview:mem"
      "4": "preview:tasks"
      "5": "preview:agents"
      "6": "preview:live"
      "7": "preview:contexts"
    right:                    # preview pane focused
      "1": "some:command"
  conversation:
    left:                     # message list focused
      "1": "detail:compact"
      "2": "detail:standard"
      "3": "detail:verbose"
  config:
    left:
      "1": "page:overview"
      "2": "page:memory"
      "3": "page:project"
      "4": "page:skills"
      "5": "page:hooks"
      "6": "page:mcp"
  stats:
    left:
      "1": "page:overview"
      "2": "page:tools"
      "3": "page:errors"
```

Values are command names from the command registry (`:` command mode).
User config merges over defaults — override specific keys or add new views.

### Config Explorer

The config explorer (`:view:config` or `v` → `c`) shows all Claude Code configuration organized by category. Use `:page:<category>` to filter. Categories include:

| Category | Content |
|----------|---------|
| MEMORY | Global CLAUDE.md, memory files, contexts, rules |
| PROJECT | Project-level CLAUDE.md and memory |
| LOCAL | Parent CLAUDE.md files up the directory tree |
| SKILLS | User-defined skills |
| AGENTS | User-defined agents |
| COMMANDS | User-defined slash commands |
| HOOKS | Hooks from settings.json |
| MCP | MCP server configurations |
| KEYMAPS | Current keybindings (from config.yaml or defaults) |
| SHORTCUTS | Number key shortcuts per view and focus side |

### Actions Menu

The actions menu (`x` key) provides session-specific operations:

| Key | Action | Condition |
|-----|--------|-----------|
| `d` | Delete session | Always |
| `m` | Move/rename project | Always |
| `r` | Resume session | Always |
| `n` | New session | Always |
| `F` | Fork session | Always |
| `w` | Create git worktree | Always |
| `y` | Copy project path | Always |
| `c` | Copy conversation text | Always |
| `t` | Edit custom tags/badges | Always |
| `u` | Extract URLs | Always |
| `f` | Extract file paths | Always |
| `g` | Extract changed files | Always |
| `R` | Send to remote (experimental) | Always |
| `X` | Remove memory files | Has memory |
| `M` | Import memory from worktree | Is worktree |
| `k` | Kill live session | Live + tmux |
| `i` | Send input | Live + tmux |
| `j` | Jump to tmux pane | Live + tmux |

## Development

### Build

```bash
make build      # build binary → bin/ccx
make run        # build + run
make install    # build + install to ~/.local/bin/ccx
make test       # run all tests
make vet        # go vet
make tidy       # go mod tidy
make clean      # remove build artifacts
```

Version is injected via `-ldflags` from `git describe --tags --always --dirty`.

### Debug

```bash
CCX_DEBUG=1 ccx    # enables debug logging to /tmp/ccx-debug.log
```

### Recording Demo GIFs

```bash
# Prerequisites: brew install asciinema agg
./docs/record-demos.sh all       # record all 6 demos
./docs/record-demos.sh browse    # record just one
```

Uses tmux + asciinema + agg for fully automated terminal recording.

### Testing

```bash
go test ./internal/...                                    # run all tests
go test ./internal/tui/ -run TestRender                   # run render snapshot tests
UPDATE_GOLDEN=1 go test ./internal/tui/ -run TestRender   # regenerate golden files
go test ./internal/session/ -run TestSplit                 # run system tag tests
go test -v ./internal/tui/ -run TestConv                  # verbose conversation UX tests
```

#### Test Patterns

**Pure function tests** — parser, merge, filter, fold logic:
- `internal/session/parser_test.go` — JSONL parsing, content blocks, timestamps
- `internal/session/systemtag_test.go` — XML tag splitting, system tag detection
- `internal/tui/merge_test.go` — conversation merging, context filtering, fold defaults
- `internal/tui/blockfilter_test.go` — block filter parsing and matching

**State machine tests** — TUI interactions via `setupConvApp` + `pressKey`:
- `internal/tui/conversation_ux_test.go` — preview updates, live tail, resize, fold state
- `internal/tui/cmdmode_test.go` — command mode parsing and execution
- `internal/tui/resize_test.go` — resize preservation of fold/scroll/cursor state

**Golden file snapshot tests** — render output captured to `testdata/*.golden`:
- `internal/tui/render_test.go` — message rendering with system tags, tools, block cursor
- Regenerate with `UPDATE_GOLDEN=1`

**Integration tests** — config/plugin discovery with temp directories:
- `internal/session/config_test.go` — config file scanning
- `internal/session/plugin_test.go` — plugin and marketplace discovery
- `internal/tui/config_test.go` — config explorer UI
- `internal/tui/plugins_test.go` — plugin explorer UI

### Benchmarks

```bash
go run ./cmd/bench    # run performance benchmarks
```

### Project Structure

```
cmd/bench/              benchmark tool
internal/
  session/              JSONL parsing, scanning, models, stats, config/plugin discovery
  tui/                  Bubble Tea UI (app, sessions, conversation, messages, stats, config, plugins)
  tmux/                 tmux integration (live detection, pane capture, input)
  extract/              URL and file path extraction from sessions
```

## How It Works

ccx reads Claude Code's session files from `~/.claude/projects/`. Each session is a JSONL file containing the full conversation history — user prompts, assistant responses, tool calls, and results. Subagent sessions live under `{sessionID}/subagents/agent-*.jsonl` with optional `*.meta.json` for type metadata.

Session metadata is cached to `~/.claude/sessions.gob` for instant startup (~1ms). A full async scan runs in the background to pick up new sessions.

Live sessions are detected by reading Claude Code's process registry at `~/.claude/sessions/<pid>.json`; the full schema and ccx's read strategy are documented in [docs/claude-code/live-session-registry.md](docs/claude-code/live-session-registry.md).

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Requirements

- Go 1.25+
- Claude Code sessions in `~/.claude/projects/`
- tmux (optional, for live session features)

## License

Apache License 2.0
