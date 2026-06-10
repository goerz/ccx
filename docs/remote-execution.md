# Remote Execution Mode (experimental)

Run Claude Code on a remote Kubernetes pod with your local configuration and auth
credentials, then attach to it or stream the session back into ccx. Remote
sessions appear in the session list with the `[R·exp]` badge and match the
`is:remote` filter (see [Session Badges](../README.md#session-badges)). The
feature is experimental and macOS-only for auth extraction.

## Architecture

```
[Local ccx]  ──kubectl exec──>  [K8s pod: ccx-remote-xxxx]
   │                                 │ container "main" (sleeps; we exec in)
   ├ extract OAuth token (keychain)  ├ create user "claude", install Node + claude CLI
   ├ tar config + workdir, upload    ├ run claude (stream-json) or interactive attach
   └ stream / attach via kubectl     └ work in /workspace
```

ccx drives a plain pod entirely over `kubectl exec` — there is no in-cluster
agent. The container just stays alive; ccx execs into it to provision, sync, run,
and stream.

## Configuration

Defaults live under a `remote:` section in `~/.config/ccx/config.yaml` and are
merged with per-invocation overrides (`mergeRemoteConfig`). The full schema is
`remote.Config` in [`internal/remote/config.go`](../internal/remote/config.go);
key fields and their defaults (`Config.Defaults()`):

| Field | Default | Notes |
|-------|---------|-------|
| `context` | current kubectl context | required |
| `namespace` | `default` | |
| `image` | `ubuntu:24.04` | base image; Node + claude installed on first use |
| `container` | `main` | |
| `remote_user` | `claude` | non-root user Claude runs as |
| `remote_home` | `/home/<remote_user>` | |
| `work_dir` | `/workspace` | remote working directory |
| `git_repo` / `git_branch` | — / `main` | clone source when no `local_dir` |
| `local_dir` | — | local workdir to tar and upload |
| `cpu_limit` / `memory_limit` | `2` / `4Gi` | pod resource limits |
| `arch` | host arch | `amd64` / `arm64`; `ArchMismatch()` warns on mismatch |
| `env_vars` / `mirror_env` | — | inject literal or mirror local env vars |
| `claude_args` | — | extra `claude` CLI flags (e.g. `--model`) |

## Components (`internal/remote/`)

- **`config.go`** — `Config`, `Defaults()`, `Validate()`, `GeneratePodName()`
  (`ccx-remote-<hex>`), arch helpers.
- **`pod.go`** — `podSpec()`, `CreatePod()`, `WaitForPod()`, `ExecInPod()`,
  `ExecInteractive()`, `PodPhase()`, `DeletePod()`. The pod is a single container
  that runs a sleep command so ccx can exec into it.
- **`session.go`** — `Start()` / `Adopt()` / `StartFromSnapshot()` orchestrate
  setup; `BuildAttachCmd()` and `BuildClaudeCmd()` construct the `claude`
  invocation; `FetchSessionJSONL()` pulls the transcript back; `Stop()`,
  `IsRunning()`.
- **`sync.go`** — `CreateConfigTarball()` (settings, memory, skills, agents,
  commands, hooks, project config, optional session JSONL) and
  `CreateWorkdirTarball()`, uploaded via `UploadTarball()` (streamed through
  `kubectl exec ... tar -x`).
- **`stream.go`** — `StreamExec()` runs a pod command and streams stdout line by
  line as `StreamLine` values; used to follow `claude --output-format
  stream-json --verbose`.
- **`snapshot.go`** — save/restore/export/import remote workdir + session as
  tarballs under `~/.config/ccx/snapshots/`.
- **`persist.go`** — `SavedSession` records persisted to
  `~/.config/ccx/remote-sessions.yaml` so remote pods survive ccx restarts and
  reappear as virtual session items.

## Setup flow (`Session.setup`)

1. **Auth** — `tmux.ExtractClaudeOAuthToken()` reads the OAuth token from the
   macOS keychain (same mechanism as the config test environment).
2. **Pod** — reuse a `Running`/`Pending` pod with the configured name, else
   `CreatePod` + `WaitForPod` (3 min timeout).
3. **User + auth env** — create the `claude` user; write the token to
   `~/.claude_env` (mode 600) so it is sourced per-exec rather than baked into the
   image.
4. **CLI** — if `claude` isn't on `PATH`, `apt-get` Node.js 22 and
   `npm install -g @anthropic-ai/claude-code`.
5. **Config sync** — upload the config tarball into the remote home.
6. **Workdir sync** — upload a prebuilt tarball (snapshot/fork) if present, else
   tar `local_dir`.

## Running and attaching

- **Interactive attach** — `BuildAttachCmd()` runs `kubectl exec -it` as the
  `claude` user, sources `~/.claude_env`, `cd`s into the workdir, and launches
  `claude` (with `--resume <id>` and any `claude_args`). In the TUI, `Enter` on a
  remote session attaches.
- **Live preview** — the streaming variant (`stream-json`) is captured through a
  hidden tmux pane and shown in the live preview pane (`L`), reusing the same
  preview path as local live sessions.

## TUI commands

All are command-mode entries (`:`), with `r:` short aliases:

| Command | Action |
|---------|--------|
| `remote:start` (`r:start`) | resume the selected session on a remote pod |
| `remote:sync-up` (`r:sync-up`) | sync local session/workdir up to the pod |
| `remote:attach` (`r:attach`) | reattach to remote Claude |
| `remote:stop` (`r:stop`, `remote:rm`) | stop and delete the pod |
| `remote:stop-pull` (`r:stop-pull`) | pull the workdir back, then stop |
| `remote:pull` (`r:pull`, `remote:sync-down`) | fetch the pod workdir to the host |
| `remote:ls` (`r:ls`) | jump to the first remote session |
| `remote:phase` (`r:phase`) | show the pod phase for the selected remote |
| `remote:exec <cmd…>` (`r:exec`) | `kubectl exec` in the selected pod |
| `remote:fork` (`r:fork`) | clone the selected pod into a fresh one |
| `remote:snapshot [name]` (`r:snap`) | save remote workdir + session |
| `remote:snapshots` (`r:snaps`) | list saved snapshots |
| `remote:restore <name>` (`r:restore`) | boot a new pod from a snapshot |
| `remote:rm-snap <name>` (`r:rm-snap`) | delete a saved snapshot |
| `remote:export-snap <name> <out.tgz>` (`r:export-snap`) | export a snapshot |
| `remote:import-snap <bundle.tgz> [name]` (`r:import-snap`) | import a snapshot |

## Security

- The OAuth token is extracted locally and transmitted only over the encrypted
  kubectl channel; on the pod it lives in `~/.claude_env` (mode 600, owned by the
  `claude` user) rather than the image or a committed file.
- Claude runs as a non-root user so `--dangerously-skip-permissions` is usable.
- The pod runs in your namespace; ccx needs only pod create/exec/delete rights
  there.

## Prerequisites

- `kubectl` configured with a usable context and namespace.
- Permission to create/exec/delete pods in the target namespace.
- macOS (OAuth token extraction reads the macOS keychain).
- Outbound HTTPS from the pod to `api.anthropic.com`.
