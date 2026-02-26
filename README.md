# Commandry (MVP)

Commandry is a local-first CLI that records explicitly executed commands and exports a deterministic markdown runbook.

## Install

Primary path (pre-`v1.0.0`):
- download the latest binary from GitHub Releases
- run once by path (for example `.\cmdry.exe` on Windows), then run setup:
  - `cmdry setup`
  - `cmdry setup status`

Setup commands:
- `cmdry setup` - interactive install flow (bin + user PATH)
- `cmdry setup plan` - preview setup actions (no changes)
- `cmdry setup apply` - non-interactive/direct apply
- `cmdry setup undo` - revert setup changes recorded in setup state

Updating from a GitHub-downloaded binary (important):
- `setup` installs the currently running executable. To update, run setup from the new downloaded file path first.
- Windows (PowerShell/CMD): `.\cmdry.exe setup apply --yes`
- Linux: `./cmdry setup apply --yes`
- macOS: `./cmdry setup apply --yes`
- Then open a new terminal and verify with `cmdry version`.

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

The binary is created as `cmdry` (or `cmdry.exe` on Windows).

## Quickstart Demo

```bash
cmdry i
cmdry s "Deploy to staging" -e staging
cmdry r -- kubectl apply -f deploy.yaml
cmdry r -- kubectl rollout status deploy/api
cmdry stp
cmdry x -l -f md
```

Expected output artifact:
- `runbooks/<timestamp>-<slug>.md`

## Shortest Path to Runbook

Already installed and initialized:

```bash
cmdry s "my runbook session"
cmdry r -- <your command>
cmdry stp
cmdry x -l -f md
```

First run only (one extra command):

```bash
cmdry i
```

## CLI Commands

- `cmdry init` (alias: `i`) initializes local config and session storage in `os.UserConfigDir()/commandry`.
- `cmdry setup` installs Commandry to the user bin path and updates user PATH.
- `cmdry setup plan` previews setup actions without applying changes.
- `cmdry setup apply` applies setup changes directly (supports `--yes` and `--verbose`).
- `cmdry setup status` shows setup status for current or specified `--bin-dir`.
- `cmdry setup undo` reverts setup changes based on setup state.
- `cmdry start "<title>"` (alias: `s`) starts recording session metadata. Optional environment label: `--env/-e`.
- `cmdry run -- <cmd ...>` (alias: `r`) executes command and records a sanitized step.
- `cmdry status` shows current recording state.
- `cmdry doctor` runs local diagnostics (paths, write access, PATH hints, tool availability).
- `cmdry stop` (alias: `stp`) finalizes the active session.
- `cmdry export --last --md` (alias: `x`) exports the latest completed session to markdown.
- `cmdry export --session <id> -f md` exports a specific completed session by id.
- `cmdry sessions list` lists recent completed sessions (use `-n` to control count).
- `cmdry alias --shell <powershell|bash|zsh|cmd>` prints alias snippet for `it` (no system changes).
- `cmdry version` (alias: `v`) prints build version metadata.
- Short flags: `export --last/-l`, `export --format/-f md`; `--md` remains supported for compatibility.

## Optional: Shell Hooks (Faster Workflow)

Hooks make command capture feel natural in daily shell usage.
Hooks automatically capture commands between start and stop, so you don't need to prefix each command with cmdry run.

PowerShell:

```bash
cmdry hooks install powershell --yes
cmdry hooks status
```

Bash:

```bash
cmdry hooks install bash
cmdry hooks status
```

Zsh:

```bash
cmdry hooks install zsh
cmdry hooks status
```

Remove hooks at any time:

```bash
cmdry hooks uninstall powershell
cmdry hooks uninstall bash
cmdry hooks uninstall zsh
```

## Windows Shell Builtins

On Windows, `echo`, `dir`, `copy`, and similar commands are `cmd.exe` builtins, not standalone executables.

Use one of these forms:

```bash
cmdry run -- cmd /c echo "build started"
cmdry run -- powershell -NoProfile -Command "Write-Output 'build started'"
```

## Diagnostics

Use `cmdry doctor` when command resolution or local setup looks wrong.

It prints:
- Commandry paths (`root`, `config.yaml`, `sessions.jsonl`, `active_session.json`)
- initialization status
- storage writeability check
- PATH-related hints (including Windows-specific hints)
- availability of `kubectl`, `docker`, and `terraform`

## Data Location and Reset

Commandry stores local data under `os.UserConfigDir()/commandry`.

Typical paths:
- Windows: `%APPDATA%\commandry` (for example `C:\Users\<you>\AppData\Roaming\commandry`)
- macOS: `~/Library/Application Support/commandry`
- Linux: `~/.config/commandry`

Inside that directory:
- `config.yaml` (policy/config)
- `sessions.jsonl` (completed sessions store)
- `active_session.json` (active recording state, only while recording)

Reset/uninstall:
- stop active recording if any (`cmdry stop`)
- delete the `commandry` directory in your config location
- optionally delete local `runbooks/` in your project directory

## Interpreting Runbook Results

- `Result: OK` -> command started and finished with exit code `0`
- `Result: FAILED (command_not_found)` -> process did not start
- `Result: FAILED (nonzero_exit)` -> process started and returned non-zero
- `Exit code:` is shown only when a process actually started

## Security Notes

- Recording is off by default.
- Commandry only records commands executed through `cmdry run -- ...` while a session is active.
- Captured metadata is minimal: timestamp, sanitized command, exit code, duration, and optional working directory.
- Stdout and stderr are never stored in MVP.
- Redaction happens before writing to disk.
- Denylisted commands are stored as `[REDACTED BY POLICY]` by default.
- Optional: set `policy.enforce_denylist: true` in `config.yaml` to block denylisted commands before execution in `cmdry run`.
- Commandry does not perform telemetry, analytics, or network calls in MVP.

Quick examples:
- `cmdry run -- curl -H "Authorization: Bearer abcdef" https://example.com` -> token value is stored as `[REDACTED]`
- `cmdry run -- printenv` -> stored command becomes `[REDACTED BY POLICY]`

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
This is usually a false positive for local build artifacts, not Commandry runtime behavior.

If this happens on Windows, run tests with local cache/temp folders inside the repo:

```powershell
Set-Location <repo-path>
New-Item -ItemType Directory -Force .gocache, .gotmp | Out-Null
$env:GOCACHE = "$PWD\.gocache"
$env:GOTMPDIR = "$PWD\.gotmp"
go test ./...
```

If your AV still blocks test binaries, add a narrow exclusion for the repository test build paths (`.gocache`, `.gotmp`) rather than disabling protection globally.
