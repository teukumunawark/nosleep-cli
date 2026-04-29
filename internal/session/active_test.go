package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActiveState(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "state.json")
	store := NewStore(storePath)

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}

	pid := os.Getpid()
	processStartedAt, err := ProcessStartedAt(pid)
	if err != nil {
		t.Fatalf("failed to get process start time: %v", err)
	}

	validState := State{
		PID:              pid,
		StartedAt:        time.Now(),
		ProcessStartedAt: &processStartedAt,
		Mode:             ModeOpenEnded,
		AwakeMode:        AwakeModeSystemDisplay,
		Executable:       executable,
	}

	t.Run("no state file", func(t *testing.T) {
		_, ok, err := ActiveState(store)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ok {
			t.Fatalf("expected ok to be false")
		}
	})

	t.Run("valid matching state", func(t *testing.T) {
		if err := store.Write(validState); err != nil {
			t.Fatalf("failed to write state: %v", err)
		}

		state, ok, err := ActiveState(store)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !ok {
			t.Fatalf("expected ok to be true")
		}
		if state.PID != validState.PID {
			t.Errorf("expected PID %d, got %d", validState.PID, state.PID)
		}
	})

	t.Run("state not matching process", func(t *testing.T) {
		unmatchingState := validState
		unmatchingState.PID = 999999
		assertStateRemovedAndFalse(t, store, storePath, unmatchingState)
	})

	t.Run("expired auto stop", func(t *testing.T) {
		expiredAt := time.Now().Add(-time.Minute)
		expiredState := validState
		expiredState.Mode = ModeTimed
		expiredState.AutoStopAt = &expiredAt
		assertStateRemovedAndFalse(t, store, storePath, expiredState)
	})

	t.Run("invalid state", func(t *testing.T) {
		invalidState := validState
		invalidState.Mode = "invalid_mode"
		assertStateRemovedAndFalse(t, store, storePath, invalidState)
	})
}

func assertStateRemovedAndFalse(t *testing.T, store Store, storePath string, state State) {
	t.Helper()
	if err := store.Write(state); err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	_, ok, err := ActiveState(store)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected ok to be false")
	}

	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Errorf("expected state file to be removed")
	}
}
