# Commandry Black-Box Behavior Contract

This document defines user-visible CLI behavior that tests must enforce without relying on internal implementation details.

## C1. CLI Contract
- `cmdry --help` and `cmdry help` succeed (`exit=0`) and expose command-oriented usage.
- `cmdry version` succeeds (`exit=0`) and prints a non-empty version line.
- Unknown command fails (`exit!=0`) and includes an "unknown command" diagnostic.

## C2. Session Recording Contract
- Without `start`, record operations are rejected and no session step is added.
- Between `start` and `stop`, executed commands are recorded.
- After `stop`, recording commands are rejected.
- `export --last` produces a runbook file for the latest completed session.

## C3. Security / Policy Contract
- Sentinel secrets must not appear in exported runbooks.
- Denylisted commands are represented as policy-redacted text (`[REDACTED BY POLICY]` contract token).
- If `policy.enforce_denylist: true` is set in `config.yaml`, denylisted commands are blocked before execution in `cmdry run`.
- Raw command stdout/stderr payload is not persisted into runbook command content.

## C4. Export Contract
- For a completed session, `export --last` and `export --session <last-id>` are equivalent in content.
- Export format is deterministic for the same recorded session.

## C5. Metamorphic Contract
- Command aliases and full command names are behavior-equivalent (`s/stp/x` vs `start/stop/export`).
- Equivalent export flags (`--md` vs `--format md`) produce equivalent runbook content.

## C6. Hooks Contract
- Hook installation/uninstallation is idempotent on temp profile files.
- Hook recorder must not persist `cmdry ...` self-commands (anti-recursion contract token).

## C7. Setup Lifecycle Contract
- `setup plan` is non-mutating preview.
- `setup apply` installs and reports status for target bin dir.
- `setup undo` reverts setup changes based on setup state; second undo is a no-op.
