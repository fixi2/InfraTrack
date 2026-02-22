# UX Contract (v0.5)

This document defines user-visible UX behavior that must remain stable unless explicitly changed and reviewed.

## Output Messaging

1. Status lines use role labels/icons (for example OK/WARN/ERROR) and keep one-line, action-first wording.
2. Single hint/help message is rendered as:
   - `Tip:`
   - `   <message>`
3. Multiple hint/help messages in the same context are rendered as:
   - `Tips:`
   - `   -> <message>`
   - `   -> <message>`
4. Tips should be grouped per command result section (no repeated `Tip:` headers for adjacent setup tips).

## Setup UX

1. `infratrack setup` is interactive apply flow.
2. `infratrack setup plan` is dry-run only and never modifies files or PATH.
3. `infratrack setup apply --yes` is non-interactive and should finish with concise completion output.
4. Setup completion tips must include terminal restart guidance when PATH was modified.

## Runbook Readability UX

1. Step titles use status + command snippet:
   - `<index>. [OK|FAILED|REDACTED|UNKNOWN] <snippet>`
2. Summary includes:
   - recorded step count
   - result counters: `OK/FAILED/REDACTED`
   - total duration in ms
3. Reviewer comments are rendered as:
   - `Reviewer note:` for one comment
   - `Reviewer notes:` for two or more comments
4. No generic `Execute command` headings in step list.

## Security UX Guarantees

1. Redaction remains strict:
   - denylisted commands remain policy-redacted
   - sensitive values remain masked
2. `REDACTED` summary counter is step-based and includes:
   - policy-redacted steps
   - inline redaction (`[REDACTED]`) in sanitized commands
3. Raw secrets must not appear in exported runbooks.

## Test Mapping

- `internal/cli/output_test.go` -> output/tips contract
- `internal/export/markdown_test.go` -> runbook readability summary/comments contract
- `e2e/blackbox/golden_test.go` -> stable user-facing help/runbook skeleton contract
- `e2e/blackbox/security_policy_test.go` -> secret non-leak contract
