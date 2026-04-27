package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ModeOpenEnded = "open_ended"
	ModeTimed     = "timed"
	ModeUntil     = "until"

	AwakeModeSystemDisplay = "system_display"
)

var ErrInvalidState = errors.New("invalid state")

type State struct {
	PID              int        `json:"pid"`
	StartedAt        time.Time  `json:"started_at"`
	ProcessStartedAt *time.Time `json:"process_started_at"`
	Mode             string     `json:"mode"`
	AwakeMode        string     `json:"awake_mode"`
	AutoStopAt       *time.Time `json:"auto_stop_at,omitempty"`
	Executable       string     `json:"executable"`
	Label            string     `json:"label,omitempty"`
}

type Store struct {
	path string
}

func DefaultStore() (Store, error) {
	path, err := DefaultStatePath()
	if err != nil {
		return Store{}, err
	}
	return Store{path: path}, nil
}

func NewStore(path string) Store {
	return Store{path: path}
}

func DefaultStatePath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("find user config directory: %w", err)
		}
	}

	return filepath.Join(base, "NoSleepCLI", "state.json"), nil
}

func (s Store) Path() string {
	return s.path
}

func (s Store) Read() (State, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, false, nil
		}
		return State{}, false, fmt.Errorf("read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false, fmt.Errorf("decode state file: %w", err)
	}
	if err := state.Validate(); err != nil {
		return State{}, false, err
	}

	return state, true, nil
}

func (s Store) Write(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.path), "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("commit state file: %w", err)
	}
	removeTmp = false

	return nil
}

func (s Store) Remove() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove state file: %w", err)
	}
	return nil
}

func ModeLabel(mode string) string {
	switch mode {
	case ModeTimed:
		return "Timed session"
	case ModeUntil:
		return "Until time"
	default:
		return "Open-ended"
	}
}

func AwakeModeLabel(mode string) string {
	switch mode {
	case AwakeModeSystemDisplay:
		return "System + Display"
	default:
		return "System + Display"
	}
}

func (s State) Validate() error {
	if s.PID <= 0 {
		return fmt.Errorf("%w: missing pid", ErrInvalidState)
	}
	if s.StartedAt.IsZero() {
		return fmt.Errorf("%w: missing start time", ErrInvalidState)
	}
	if s.ProcessStartedAt == nil || s.ProcessStartedAt.IsZero() {
		return fmt.Errorf("%w: missing process start time", ErrInvalidState)
	}
	switch s.Mode {
	case ModeOpenEnded, ModeTimed, ModeUntil:
	default:
		return fmt.Errorf("%w: unknown mode %q", ErrInvalidState, s.Mode)
	}
	if s.AwakeMode != AwakeModeSystemDisplay {
		return fmt.Errorf("%w: unknown awake mode %q", ErrInvalidState, s.AwakeMode)
	}
	if s.Executable == "" {
		return fmt.Errorf("%w: missing executable", ErrInvalidState)
	}
	return nil
}
