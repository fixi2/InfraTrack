package setup

import "time"

const StateSchemaVersion = 1

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

type CompletionMode string

const (
	CompletionNone CompletionMode = "none"
)

type StateFile struct {
	SchemaVersion    int           `json:"schemaVersion"`
	CreatedDirs      []string      `json:"createdDirs,omitempty"`
	InstalledBinPath string        `json:"installedBinPath,omitempty"`
	PathEntryAdded   string        `json:"pathEntryAdded,omitempty"`
	FilesTouched     []TouchedFile `json:"filesTouched,omitempty"`
	PendingFinalize  bool          `json:"pendingFinalize,omitempty"`
	Timestamp        time.Time     `json:"timestamp"`
}

type TouchedFile struct {
	Path   string `json:"path"`
	Marker string `json:"marker,omitempty"`
}

type PlanInput struct {
	Scope      Scope
	BinDir     string
	NoPath     bool
	Completion CompletionMode
}

type ApplyInput struct {
	Scope            Scope
	BinDir           string
	NoPath           bool
	Completion       CompletionMode
	SourceBinaryPath string
}

type ApplyResult struct {
	OS               string   `json:"os"`
	Scope            Scope    `json:"scope"`
	SourceBinaryPath string   `json:"sourceBinaryPath"`
	TargetBinDir     string   `json:"targetBinDir"`
	InstalledBinPath string   `json:"installedBinPath"`
	StatePath        string   `json:"statePath"`
	CreatedDirs      []string `json:"createdDirs,omitempty"`
	PendingFinalize  bool     `json:"pendingFinalize"`
	Actions          []string `json:"actions"`
	Notes            []string `json:"notes,omitempty"`
}

type Plan struct {
	OS               string   `json:"os"`
	Scope            Scope    `json:"scope"`
	CurrentExe       string   `json:"currentExe"`
	TargetBinDir     string   `json:"targetBinDir"`
	TargetBinaryPath string   `json:"targetBinaryPath"`
	Actions          []string `json:"actions"`
	Notes            []string `json:"notes,omitempty"`
}

type Status struct {
	OS               string `json:"os"`
	Scope            Scope  `json:"scope"`
	CurrentExe       string `json:"currentExe"`
	BinDir           string `json:"binDir"`
	TargetBinaryPath string `json:"targetBinaryPath"`
	Installed        bool   `json:"installed"`
	PathOK           bool   `json:"pathOk"`
	StateFound       bool   `json:"stateFound"`
	PendingFinalize  bool   `json:"pendingFinalize"`
}
