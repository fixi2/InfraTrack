package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func BuildPlan(input PlanInput) (Plan, error) {
	if input.Scope == "" {
		input.Scope = ScopeUser
	}
	if input.Scope != ScopeUser {
		return Plan{}, fmt.Errorf("scope %q is not available in v0.5.0 setup (use --scope user)", input.Scope)
	}

	binDir := input.BinDir
	if binDir == "" {
		var err error
		binDir, err = DefaultBinDir()
		if err != nil {
			return Plan{}, err
		}
	}
	binDir = filepath.Clean(binDir)

	exe, err := CurrentExecutable()
	if err != nil {
		return Plan{}, err
	}
	targetBinary := ResolveTargetBinaryPath(binDir)

	actions := make([]string, 0, 4)
	if normalizePathForCompare(exe) == normalizePathForCompare(targetBinary) {
		actions = append(actions, "Binary already runs from target location (no copy needed).")
	} else {
		actions = append(actions, fmt.Sprintf("Copy binary to %s", targetBinary))
	}

	if input.NoPath {
		actions = append(actions, "Skip PATH changes (--no-path).")
	} else if PathContainsDir(os.Getenv("PATH"), binDir) {
		actions = append(actions, "PATH already contains target bin dir (no change).")
	} else {
		actions = append(actions, fmt.Sprintf("Add %s to user PATH", binDir))
	}

	if input.Completion == CompletionNone {
		actions = append(actions, "Completion: none")
	}

	notes := []string{
		"Dry-run only. No changes are applied.",
		"Run `infratrack setup` to apply changes now, or `infratrack setup apply` for direct apply.",
		"Restart terminal after PATH changes to pick up updates.",
	}

	return Plan{
		OS:               runtime.GOOS,
		Scope:            input.Scope,
		CurrentExe:       exe,
		TargetBinDir:     binDir,
		TargetBinaryPath: targetBinary,
		Actions:          actions,
		Notes:            notes,
	}, nil
}

func BuildStatus(scope Scope, binDir string) (Status, error) {
	if scope == "" {
		scope = ScopeUser
	}
	if scope != ScopeUser {
		return Status{}, fmt.Errorf("scope %q is not available in v0.5.0 setup (use --scope user)", scope)
	}

	if binDir == "" {
		var err error
		binDir, err = DefaultBinDir()
		if err != nil {
			return Status{}, err
		}
	}
	binDir = filepath.Clean(binDir)

	exe, err := CurrentExecutable()
	if err != nil {
		return Status{}, err
	}
	targetBinary := ResolveTargetBinaryPath(binDir)
	_, statErr := os.Stat(targetBinary)
	installed := statErr == nil

	statePath, err := DefaultStatePath()
	if err != nil {
		return Status{}, err
	}
	state, found, err := LoadState(statePath)
	if err != nil {
		return Status{}, err
	}

	return Status{
		OS:               runtime.GOOS,
		Scope:            scope,
		CurrentExe:       exe,
		BinDir:           binDir,
		TargetBinaryPath: targetBinary,
		Installed:        installed,
		PathOK:           PathContainsDir(os.Getenv("PATH"), binDir),
		StateFound:       found,
		PendingFinalize:  state.PendingFinalize,
	}, nil
}
