package store

import "time"

type Step struct {
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
	Status     string    `json:"status,omitempty"` // OK, FAILED, REDACTED
	Reason     string    `json:"reason,omitempty"` // nonzero_exit, command_not_found, start_failed, policy_redacted, unknown
	ExitCode   *int      `json:"exit_code,omitempty"`
	DurationMS int64     `json:"duration_ms"`
	CWD        string    `json:"cwd,omitempty"`
}

type Session struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Env       string     `json:"env,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Steps     []Step     `json:"steps"`
}
