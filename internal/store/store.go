package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNotInitialized      = errors.New("store is not initialized")
	ErrActiveSessionExists = errors.New("active session already exists")
	ErrNoActiveSession     = errors.New("no active session")
	ErrNoSessions          = errors.New("no completed sessions")
)

type SessionStore interface {
	Init(ctx context.Context) error
	IsInitialized(ctx context.Context) (bool, error)
	RootDir() string
	StartSession(ctx context.Context, title string, startedAt time.Time) (*Session, error)
	GetActiveSession(ctx context.Context) (*Session, error)
	AddStep(ctx context.Context, step Step) error
	StopSession(ctx context.Context, endedAt time.Time) (*Session, error)
	LastSession(ctx context.Context) (*Session, error)
}

type JSONStore struct {
	rootPath        string
	configPath      string
	sessionsPath    string
	activeStatePath string
}

func DefaultRootDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(configDir, "infratrack"), nil
}

func NewJSONStore(rootPath string) *JSONStore {
	return &JSONStore{
		rootPath:        rootPath,
		configPath:      filepath.Join(rootPath, "config.yaml"),
		sessionsPath:    filepath.Join(rootPath, "sessions.jsonl"),
		activeStatePath: filepath.Join(rootPath, "active_session.json"),
	}
}

func (s *JSONStore) RootDir() string {
	return s.rootPath
}

func (s *JSONStore) Init(_ context.Context) error {
	if err := os.MkdirAll(s.rootPath, 0o700); err != nil {
		return fmt.Errorf("create root directory: %w", err)
	}

	if err := s.ensureConfigFile(); err != nil {
		return fmt.Errorf("ensure config file: %w", err)
	}

	if err := s.ensureDataFile(); err != nil {
		return fmt.Errorf("ensure sessions file: %w", err)
	}

	return nil
}

func (s *JSONStore) IsInitialized(_ context.Context) (bool, error) {
	info, err := os.Stat(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat config file: %w", err)
	}

	return !info.IsDir(), nil
}

func (s *JSONStore) StartSession(_ context.Context, title string, startedAt time.Time) (*Session, error) {
	if err := s.requireInitialized(); err != nil {
		return nil, err
	}

	_, err := os.Stat(s.activeStatePath)
	if err == nil {
		return nil, ErrActiveSessionExists
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("check active state: %w", err)
	}

	session := &Session{
		ID:        fmt.Sprintf("%d", startedAt.UnixNano()),
		Title:     strings.TrimSpace(title),
		StartedAt: startedAt.UTC(),
		Steps:     make([]Step, 0, 8),
	}

	if err := s.writeJSONAtomic(s.activeStatePath, session); err != nil {
		return nil, fmt.Errorf("write active session: %w", err)
	}

	return session, nil
}

func (s *JSONStore) GetActiveSession(_ context.Context) (*Session, error) {
	if err := s.requireInitialized(); err != nil {
		return nil, err
	}

	session, err := s.readActive()
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *JSONStore) AddStep(_ context.Context, step Step) error {
	if err := s.requireInitialized(); err != nil {
		return err
	}

	session, err := s.readActive()
	if err != nil {
		return err
	}

	session.Steps = append(session.Steps, step)
	if err := s.writeJSONAtomic(s.activeStatePath, session); err != nil {
		return fmt.Errorf("persist active session: %w", err)
	}

	return nil
}

func (s *JSONStore) StopSession(_ context.Context, endedAt time.Time) (*Session, error) {
	if err := s.requireInitialized(); err != nil {
		return nil, err
	}

	session, err := s.readActive()
	if err != nil {
		return nil, err
	}

	end := endedAt.UTC()
	session.EndedAt = &end

	if err := s.appendCompleted(session); err != nil {
		return nil, fmt.Errorf("append completed session: %w", err)
	}

	if err := os.Remove(s.activeStatePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove active state: %w", err)
	}

	return session, nil
}

func (s *JSONStore) LastSession(_ context.Context) (*Session, error) {
	if err := s.requireInitialized(); err != nil {
		return nil, err
	}

	file, err := os.Open(s.sessionsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSessions
		}
		return nil, fmt.Errorf("open sessions file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastLine string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan sessions file: %w", err)
	}
	if lastLine == "" {
		return nil, ErrNoSessions
	}

	var session Session
	if err := json.Unmarshal([]byte(lastLine), &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}

	return &session, nil
}

func (s *JSONStore) ensureConfigFile() error {
	_, err := os.Stat(s.configPath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat config file: %w", err)
	}

	const defaultConfig = `policy:
  denylist:
    - env
    - printenv
    - cat ~/.ssh/*
    - "*id_rsa*"
    - "*.pem"
    - "*.key"
    - kubectl get secret -o yaml
    - kubectl get secret -o json
    - gcloud auth print-access-token
  redaction_keywords:
    - token
    - secret
    - password
    - passwd
    - authorization
    - bearer
    - api_key
    - apikey
    - private_key
capture:
  include_stdout: false
  include_stderr: false
`
	return os.WriteFile(s.configPath, []byte(defaultConfig), 0o600)
}

func (s *JSONStore) ensureDataFile() error {
	file, err := os.OpenFile(s.sessionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open sessions file: %w", err)
	}
	return file.Close()
}

func (s *JSONStore) requireInitialized() error {
	initialized, err := s.IsInitialized(context.Background())
	if err != nil {
		return err
	}
	if !initialized {
		return ErrNotInitialized
	}
	return nil
}

func (s *JSONStore) readActive() (*Session, error) {
	data, err := os.ReadFile(s.activeStatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("read active state: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode active state: %w", err)
	}

	return &session, nil
}

func (s *JSONStore) appendCompleted(session *Session) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	file, err := os.OpenFile(s.sessionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open sessions file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(string(payload) + "\n"); err != nil {
		return fmt.Errorf("append session record: %w", err)
	}

	return nil
}

func (s *JSONStore) writeJSONAtomic(path string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	_ = os.Remove(path)
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
