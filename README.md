# InfraTrack (MVP)

InfraTrack is a local-first CLI that records explicitly executed commands and exports a deterministic markdown runbook.

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

## CLI Commands

- `infratrack init` (alias: `i`) initializes local config and session storage in `os.UserConfigDir()/infratrack`.
- `infratrack start "<title>"` (alias: `s`) starts recording session metadata. Optional environment label: `--env/-e`.
- `infratrack run -- <cmd ...>` (alias: `r`) executes command and records a sanitized step.
- `infratrack status` shows current recording state.
- `infratrack stop` (alias: `stp`) finalizes the active session.
- `infratrack export --last --md` (alias: `x`) exports the latest completed session to markdown.
- Short flags: `export --last/-l`, `export --format/-f md`; `--md` remains supported for compatibility.

## Windows Shell Builtins

On Windows, `echo`, `dir`, `copy`, and similar commands are `cmd.exe` builtins, not standalone executables.

Use one of these forms:

```bash
infratrack run -- cmd /c echo "build started"
infratrack run -- powershell -NoProfile -Command "Write-Output 'build started'"
```

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
