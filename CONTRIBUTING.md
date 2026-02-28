# Contributing

## Rebrand Notes

The project was renamed from InfraTrack to Commandry during the `v0.6.0` rebrand.

- Treat `InfraTrack` / `infratrack` as legacy compatibility text, not as a second product name.
- Do not introduce new legacy-name references in new changes unless compatibility explicitly requires it.
- When touching a legacy file, move it toward `Commandry` where the change is low-risk, or open a focused issue/PR for the remaining cleanup.
- Be deliberate around risky rename areas: public API/contracts, external integrations, migrations, release metadata, and compatibility paths.

## Commit Message Style

Default format:

`<type>(<scope>): <imperative summary>`

Examples:

- `feat(cli): add cmdry root alias and help updates`
- `fix(setup): migrate legacy config dir to commandry`
- `test(export): update runbook golden for branding changes`
- `chore(ci): pin github actions by commit sha`

### Rules

- Use English.
- Keep summary short and specific (target: <= 72 chars).
- Do not include version tags in commit titles (for example `v0.6.0`).
- One commit should represent one logical change.

### Recommended types

- `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

### Recommended scopes

- `cli`, `setup`, `hooks`, `export`, `policy`, `store`, `ci`, `tests`, `docs`, `rebrand`

### Controlled exceptions

This style is the default and should be followed in normal work.

Exceptions are allowed when needed:

- a better type/scope exists for the specific change but is not listed above
- a temporary or emergency change needs a narrower custom scope

When using an exception:

- keep the same overall format
- keep wording clear and concrete
- avoid inventing many new types/scopes without reason

## Merge Rules

- Use fast-forward merge when the branch already has a clean, intentional history.
- Use a non-fast-forward merge only when keeping the branch boundary is useful as a distinct milestone.
- If a branch has noisy or duplicate commits, clean the branch history before merging.

## Branch Hygiene

- Keep one branch focused on one logical change.
- Do not mix code changes, docs cleanup, and unrelated test churn in one branch unless they are inseparable.
- Before commit/merge, remove temporary local artifacts (`*.exe`, `*.tmp`, `*.go.<digits>`, local transcripts) that should not enter the repository.

## Optional local commit template

This repository includes `.gitmessage` to help keep commit titles consistent.

Use it locally:

`git config commit.template .gitmessage`
