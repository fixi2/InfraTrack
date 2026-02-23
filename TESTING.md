# Testing InfraTrack (Black-Box First)

## Behavior Contract

Authoritative contract: `docs/testing/behavior-contract.md`.
Every e2e black-box test enforces one or more contract items.

## Local Runs

Enable local Go cache/temp first (recommended on Windows with AV):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\.tests\use-local-go-cache.ps1
```

Build binary once:

```powershell
go build -o .\infratrack.exe ./cmd/infratrack
```

Run fast black-box pack:

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
go test ./e2e/blackbox -count=1 -run "TestCLI|TestSessionLifecycle|TestSecrets|TestExportLastEquals|TestAlias|TestEquivalentExportFlags|TestSetup"
```

Run full black-box pack:

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
go test ./e2e/blackbox -count=1
```

Run heavy volume:

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
$env:IT_BLACKBOX_VOLUME_COUNT = "10000"
go test ./e2e/blackbox -count=1 -run TestVolumeRecordingAndExport -timeout 40m
```

Release soak local (avoid double-running volume under a short timeout):

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
$env:IT_BLACKBOX_VOLUME_COUNT = "10000"
go test ./e2e/blackbox -count=1 -run TestVolumeRecordingAndExport -timeout 40m
go test ./e2e/blackbox -count=1 -timeout 20m -run "TestCLIHelpAndVersionContract|TestUnknownCommandContract|TestSessionLifecycleBlackBox|TestExportLastEqualsExportSessionID|TestGoldenHelpOutput|TestGoldenRunbookSkeleton|TestHooksInstallUninstallIdempotentOnTempProfile|TestHookRecorderAntiRecursion|TestAliasAndFullCommandsEquivalent|TestEquivalentExportFlags|TestSecretsAndDenylistDoNotLeakToRunbook|TestRandomSentinelNeverAppearsInArtifacts|TestSetupLifecycleContract"
```

## Golden Files

Golden files live in `e2e/blackbox/testdata`.

Update goldens intentionally:

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
go test ./e2e/blackbox -run "TestGoldenHelpOutput|TestGoldenRunbookSkeleton" -update
```

Review every golden update in PR.

## CI Packs

- PR: fast black-box contract pack + UX output pack + runbook readability pack (target <= 12 minutes)
- Nightly: full black-box + volume 10k + UX packs (target <= 35 minutes)
- Release soak: black-box volume/soak suite with explicit trigger (target <= 60 minutes)

## UX Packs (Local)

Build once:

```powershell
go build -o .\infratrack.exe ./cmd/infratrack
```

UX output pack (tips/labels/setup output contract):

```powershell
go test ./internal/cli -count=1 -run "TestOutputRolesNonTTYASCII|TestPrintHintsMultiUsesArrows|TestRunWithSpinnerNonTTY"
```

Runbook readability pack (summary counters/snippets/reviewer notes):

```powershell
go test ./internal/export -count=1 -run "TestRenderMarkdownGolden|TestRenderMarkdownWithOptionsComments|TestRenderMarkdownWithMultipleReviewerNotes|TestRenderMarkdownSummaryCountsInlineRedaction|TestStepTitleSnippetTruncatesLongCommand"
```

UX black-box readability check:

```powershell
$env:INFRATRACK_E2E_BIN = "$PWD\infratrack.exe"
go test ./e2e/blackbox -count=1 -run "TestGoldenRunbookSkeleton|TestGoldenHelpOutput|TestSecretsAndDenylistDoNotLeakToRunbook"
```
