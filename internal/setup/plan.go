package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		actions = append(actions, "binary already installed (no copy needed)")
	} else {
		actions = append(actions, "install binary")
	}

	if input.NoPath {
		actions = append(actions, "skip PATH update (--no-path)")
	} else if PathContainsDir(os.Getenv("PATH"), binDir) {
		actions = append(actions, "PATH already configured (no change)")
	} else {
		actions = append(actions, "add target bin dir to user PATH")
	}

	if input.Completion == CompletionNone {
		actions = append(actions, "completion: none")
	}

	return Plan{
		OS:               runtime.GOOS,
		Scope:            input.Scope,
		CurrentExe:       exe,
		TargetBinDir:     binDir,
		TargetBinaryPath: targetBinary,
		Actions:          actions,
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

	pathValue := os.Getenv("PATH")
	pathConfigured := PathContainsDir(pathValue, binDir)
	if runtime.GOOS == "windows" {
		if userPath, readErr := readWindowsUserPathFn(); readErr == nil {
			if strings.TrimSpace(pathValue) == "" {
				pathValue = userPath
			} else if strings.TrimSpace(userPath) != "" {
				pathValue = userPath + string(os.PathListSeparator) + pathValue
			}
		}
		pathConfigured = PathContainsDir(pathValue, binDir)
	} else if !pathConfigured {
		if persisted, err := hasPosixPathMarker(binDir); err == nil && persisted {
			pathConfigured = true
		}
	}

	return Status{
		OS:               runtime.GOOS,
		Scope:            scope,
		CurrentExe:       exe,
		BinDir:           binDir,
		TargetBinaryPath: targetBinary,
		Installed:        installed,
		PathOK:           pathConfigured,
		StateFound:       found,
		PendingFinalize:  state.PendingFinalize,
	}, nil
}

func hasPosixPathMarker(binDir string) (bool, error) {
	profile, err := resolvePosixProfileFn()
	if err != nil {
		return false, err
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	begin, end, exists, err := findSetupMarkerSpan(string(content))
	if err != nil || !exists {
		return false, err
	}
	block := strings.ToLower(strings.TrimSpace(string(content[begin:end])))
	quoted, err := quotePOSIXSingle(binDir)
	if err != nil {
		return false, err
	}
	want := strings.ToLower("export PATH=" + quoted + ":\"$PATH\"")
	return strings.Contains(block, want), nil
}
