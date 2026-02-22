# InfraTrack (MVP)

InfraTrack is a local-first CLI that records explicitly executed commands and exports a deterministic markdown runbook.

## Install

Primary path (pre-`v1.0.0`):
- download the latest binary from GitHub Releases
- run once by path (for example `.\infratrack.exe` on Windows), then run setup:
  - `infratrack setup`
  - `infratrack setup status`

Setup commands:
- `infratrack setup` - interactive install flow (bin + user PATH)
- `infratrack setup plan` - preview setup actions (no changes)
- `infratrack setup apply` - non-interactive/direct apply
- `infratrack setup undo` - revert setup changes recorded in setup state

Updating from a GitHub-downloaded binary (important):
- `setup` installs the currently running executable. To update, run setup from the new downloaded file path first.
- Windows (PowerShell/CMD): `.\infratrack.exe setup apply --yes`
- Linux: `./infratrack setup apply --yes`
- macOS: `./infratrack setup apply --yes`
- Then open a new terminal and verify with `infratrack version`.

Notes:
- restart terminal after PATH updates
- `winget` flow is intentionally not primary before `v1.0.0`
- macOS is not directly tested in CI yet; `zsh` support is expected to work via POSIX-compatible hooks.

## Build

Requirements:
- Go 1.22 or newer

Commands:

```bash
go mod tidy
go build ./cmd/infratrack
```

The binary is created as `infratrack` (or `infratrack.exe` on Windows).

## Quickstart Demo

```bash
infratrack i
infratrack s "Deploy to staging" -e staging
infratrack r -- kubectl apply -f deploy.yaml
infratrack r -- kubectl rollout status deploy/api
infratrack stp
infratrack x -l -f md
```

Expected output artifact:
- `runbooks/<timestamp>-<slug>.md`

## Shortest Path to Runbook

Already installed and initialized:

```bash
infratrack s "my runbook session"
infratrack r -- <your command>
infratrack stp
infratrack x -l -f md
```

First run only (one extra command):

```bash
infratrack i
```

## CLI Commands

- `infratrack init` (alias: `i`) initializes local config and session storage in `os.UserConfigDir()/infratrack`.
- `infratrack setup` installs InfraTrack to the user bin path and updates user PATH.
- `infratrack setup plan` previews setup actions without applying changes.
- `infratrack setup apply` applies setup changes directly (supports `--yes` and `--verbose`).
- `infratrack setup status` shows setup status for current or specified `--bin-dir`.
- `infratrack setup undo` reverts setup changes based on setup state.
- `infratrack start "<title>"` (alias: `s`) starts recording session metadata. Optional environment label: `--env/-e`.
- `infratrack run -- <cmd ...>` (alias: `r`) executes command and records a sanitized step.
- `infratrack status` shows current recording state.
- `infratrack doctor` runs local diagnostics (paths, write access, PATH hints, tool availability).
- `infratrack stop` (alias: `stp`) finalizes the active session.
- `infratrack export --last --md` (alias: `x`) exports the latest completed session to markdown.
- `infratrack export --session <id> -f md` exports a specific completed session by id.
- `infratrack sessions list` lists recent completed sessions (use `-n` to control count).
- `infratrack alias --shell <powershell|bash|zsh|cmd>` prints alias snippet for `it` (no system changes).
- `infratrack version` (alias: `v`) prints build version metadata.
- Short flags: `export --last/-l`, `export --format/-f md`; `--md` remains supported for compatibility.

## Optional: Shell Hooks (Faster Workflow)

Hooks make command capture feel natural in daily shell usage.
Hooks automatically capture commands between start and stop, so you don't need to prefix each command with infratrack run.

PowerShell:

```bash
infratrack hooks install powershell --yes
infratrack hooks status
```

Bash:

```bash
infratrack hooks install bash
infratrack hooks status
```

Zsh:

```bash
infratrack hooks install zsh
infratrack hooks status
```

Remove hooks at any time:

```bash
infratrack hooks uninstall powershell
infratrack hooks uninstall bash
infratrack hooks uninstall zsh
```

## Windows Shell Builtins

On Windows, `echo`, `dir`, `copy`, and similar commands are `cmd.exe` builtins, not standalone executables.

Use one of these forms:

```bash
infratrack run -- cmd /c echo "build started"
infratrack run -- powershell -NoProfile -Command "Write-Output 'build started'"
```

## Diagnostics

Use `infratrack doctor` when command resolution or local setup looks wrong.

It prints:
- InfraTrack paths (`root`, `config.yaml`, `sessions.jsonl`, `active_session.json`)
- initialization status
- storage writeability check
- PATH-related hints (including Windows-specific hints)
- availability of `kubectl`, `docker`, and `terraform`

## Data Location and Reset

InfraTrack stores local data under `os.UserConfigDir()/infratrack`.

Typical paths:
- Windows: `%APPDATA%\infratrack` (for example `C:\Users\<you>\AppData\Roaming\infratrack`)
- macOS: `~/Library/Application Support/infratrack`
- Linux: `~/.config/infratrack`

Inside that directory:
- `config.yaml` (policy/config)
- `sessions.jsonl` (completed sessions store)
- `active_session.json` (active recording state, only while recording)

Reset/uninstall:
- stop active recording if any (`infratrack stop`)
- delete the `infratrack` directory in your config location
- optionally delete local `runbooks/` in your project directory

## Interpreting Runbook Results

- `Result: OK` -> command started and finished with exit code `0`
- `Result: FAILED (command_not_found)` -> process did not start
- `Result: FAILED (nonzero_exit)` -> process started and returned non-zero
- `Exit code:` is shown only when a process actually started

## Security Notes

- Recording is off by default.
- InfraTrack only records commands executed through `infratrack run -- ...` while a session is active.
- Captured metadata is minimal: timestamp, sanitized command, exit code, duration, and optional working directory.
- Stdout and stderr are never stored in MVP.
- Redaction happens before writing to disk.
- Denylisted commands are stored as `[REDACTED BY POLICY]`.
- InfraTrack does not perform telemetry, analytics, or network calls in MVP.

Quick examples:
- `infratrack run -- curl -H "Authorization: Bearer abcdef" https://example.com` -> token value is stored as `[REDACTED]`
- `infratrack run -- printenv` -> stored command becomes `[REDACTED BY POLICY]`

## Tests

Run all tests:

```bash
go test ./...
```

Black-box contract suite:

```bash
go test ./e2e/blackbox -count=1
```

See `TESTING.md` for CI packs, volume runs, and golden update flow.
UX behavior contract: `docs/testing/ux-contract.md`.

## Antivirus Note for Contributors

Some antivirus products may flag temporary Go test binaries (for example `*.test.exe`) during `go test`.
This is usually a false positive for local build artifacts, not InfraTrack runtime behavior.

If this happens on Windows, run tests with local cache/temp folders inside the repo:

```powershell
Set-Location C:\Projects\InfraTrack
New-Item -ItemType Directory -Force .gocache, .gotmp | Out-Null
$env:GOCACHE = "$PWD\.gocache"
$env:GOTMPDIR = "$PWD\.gotmp"
go test ./...
```

If your AV still blocks test binaries, add a narrow exclusion for the repository test build paths (`.gocache`, `.gotmp`) rather than disabling protection globally.
