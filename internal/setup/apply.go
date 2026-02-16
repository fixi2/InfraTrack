package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Apply(input ApplyInput) (ApplyResult, error) {
	if input.Scope == "" {
		input.Scope = ScopeUser
	}
	if input.Scope != ScopeUser {
		return ApplyResult{}, fmt.Errorf("scope %q is not available in v0.5.0 setup (use --scope user)", input.Scope)
	}

	binDir := strings.TrimSpace(input.BinDir)
	if binDir == "" {
		var err error
		binDir, err = DefaultBinDir()
		if err != nil {
			return ApplyResult{}, err
		}
	}
	binDir = filepath.Clean(binDir)

	source := strings.TrimSpace(input.SourceBinaryPath)
	if source == "" {
		var err error
		source, err = CurrentExecutable()
		if err != nil {
			return ApplyResult{}, err
		}
	}
	source = filepath.Clean(source)

	target := ResolveTargetBinaryPath(binDir)
	res := ApplyResult{
		OS:               runtime.GOOS,
		Scope:            input.Scope,
		SourceBinaryPath: source,
		TargetBinDir:     binDir,
		InstalledBinPath: target,
		Actions:          make([]string, 0, 8),
		Notes:            make([]string, 0, 4),
	}

	dirExisted := true
	if st, err := os.Stat(binDir); err != nil || !st.IsDir() {
		dirExisted = false
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return res, fmt.Errorf("create bin dir: %w", err)
	}
	if !dirExisted {
		res.CreatedDirs = append(res.CreatedDirs, binDir)
		res.Actions = append(res.Actions, "Created target bin directory.")
	}

	if normalizePathForCompare(source) == normalizePathForCompare(target) {
		res.Actions = append(res.Actions, "Binary already runs from target location.")
	} else {
		if runtime.GOOS == "windows" {
			staging := windowsStagingPath(target)
			if err := copyFile(source, staging, 0o600); err != nil {
				return res, fmt.Errorf("copy binary to staging: %w", err)
			}
			res.Actions = append(res.Actions, fmt.Sprintf("Copied binary to staging file %s", staging))

			pending, note, err := finalizeWindowsBinary(staging, target)
			if err != nil {
				return res, err
			}
			res.PendingFinalize = pending
			if pending {
				res.Actions = append(res.Actions, "Deferred binary swap because target executable is currently locked.")
				res.Notes = append(res.Notes, note)
			} else {
				res.Actions = append(res.Actions, "Installed binary into target location.")
			}
		} else {
			staging := target + ".new"
			if err := copyFile(source, staging, 0o700); err != nil {
				return res, fmt.Errorf("copy binary to staging: %w", err)
			}
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return res, fmt.Errorf("replace target binary: %w", err)
			}
			if err := os.Rename(staging, target); err != nil {
				return res, fmt.Errorf("activate new binary: %w", err)
			}
			if err := os.Chmod(target, 0o755); err != nil {
				return res, fmt.Errorf("set execute permissions: %w", err)
			}
			res.Actions = append(res.Actions, "Installed binary into target location.")
		}
	}

	if input.NoPath {
		res.Actions = append(res.Actions, "Skipped PATH updates (--no-path).")
	} else {
		res.Actions = append(res.Actions, "PATH changes are planned but not applied in this phase.")
	}

	statePath, err := DefaultStatePath()
	if err != nil {
		return res, err
	}
	res.StatePath = statePath
	state := StateFile{
		SchemaVersion:    StateSchemaVersion,
		CreatedDirs:      append([]string(nil), res.CreatedDirs...),
		InstalledBinPath: target,
		PathEntryAdded:   "",
		FilesTouched:     nil,
		PendingFinalize:  res.PendingFinalize,
		Timestamp:        time.Now().UTC(),
	}
	if err := SaveState(statePath, state); err != nil {
		return res, err
	}
	res.Actions = append(res.Actions, fmt.Sprintf("Saved setup state to %s", statePath))
	if res.PendingFinalize {
		res.Notes = append(res.Notes, "Restart terminal and run `infratrack setup apply` again to complete binary swap.")
	}
	return res, nil
}

func windowsStagingPath(target string) string {
	if strings.HasSuffix(strings.ToLower(target), ".exe") {
		return target[:len(target)-4] + ".new.exe"
	}
	return target + ".new"
}

func finalizeWindowsBinary(stagingPath, targetPath string) (bool, string, error) {
	if err := os.Remove(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		if isFileLockError(err) {
			return true, "Restart terminal to complete upgrade. Existing binary is in use.", nil
		}
		return false, "", fmt.Errorf("prepare target binary: %w", err)
	}
	if err := os.Rename(stagingPath, targetPath); err != nil {
		if isFileLockError(err) {
			return true, "Restart terminal to complete upgrade. Existing binary is in use.", nil
		}
		return false, "", fmt.Errorf("activate staged binary: %w", err)
	}
	return false, "", nil
}

func isFileLockError(err error) bool {
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access is denied") || strings.Contains(msg, "being used by another process")
}

func copyFile(sourcePath, destPath string, mode os.FileMode) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return fmt.Errorf("copy bytes: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close destination: %w", closeErr)
	}
	return nil
}
