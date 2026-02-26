package blackbox

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Contract: C6
func TestHooksInstallUninstallIdempotentOnTempProfile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific powershell hooks test")
	}
	h := newHarness(t)
	h.mustRun("init")

	profilePath := filepath.Join(h.rootDir, "home", "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")

	h.mustRun("hooks", "install", "powershell", "--yes")
	h.mustRun("hooks", "install", "powershell", "--yes")
	content := readFile(t, profilePath)
	if strings.Count(content, "# >>> commandry hooks >>>") != 1 {
		t.Fatalf("expected single hook marker block after double install")
	}

	h.mustRun("hooks", "uninstall", "powershell")
	h.mustRun("hooks", "uninstall", "powershell")
	content2 := readFile(t, profilePath)
	if strings.Contains(content2, "# >>> commandry hooks >>>") {
		t.Fatalf("expected hook marker removed after uninstall")
	}
}

// Contract: C6
func TestHookRecorderAntiRecursion(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific hooks recorder contract test")
	}
	h := newHarness(t)
	h.mustRun("init")
	h.mustRun("hooks", "enable")
	h.mustRun("start", "hook-anti-rec")

	h.mustRun("hook", "record", "--command", "echo one", "--exit-code", "0", "--duration-ms", "5", "--cwd", h.workDir)
	h.mustRun("hook", "record", "--command", "cmdry status", "--exit-code", "0", "--duration-ms", "5", "--cwd", h.workDir)

	h.mustRun("stop")
	runbook := readFile(t, h.exportLastMD())
	if !strings.Contains(runbook, "echo one") {
		t.Fatalf("expected normal hook command in runbook")
	}
	if strings.Contains(runbook, "cmdry status") {
		t.Fatalf("anti-recursion violated: cmdry command captured")
	}
}
