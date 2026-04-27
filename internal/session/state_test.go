package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreReadWriteRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NoSleepCLI", "state.json")
	store := NewStore(path)

	autoStopAt := time.Date(2026, 4, 24, 18, 0, 0, 0, time.UTC)
	processStartedAt := time.Date(2026, 4, 24, 15, 59, 59, 0, time.UTC)
	want := State{
		PID:              1234,
		StartedAt:        time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC),
		ProcessStartedAt: &processStartedAt,
		Mode:             ModeTimed,
		AwakeMode:        AwakeModeSystemDisplay,
		AutoStopAt:       &autoStopAt,
		Executable:       `C:\Tools\nosleep\nosleep.exe`,
		Label:            "Monitoring",
	}

	if err := store.Write(want); err != nil {
		t.Fatalf("write state: %v", err)
	}

	got, ok, err := store.Read()
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !ok {
		t.Fatal("expected state to exist")
	}
	if got.PID != want.PID ||
		!got.StartedAt.Equal(want.StartedAt) ||
		got.Mode != want.Mode ||
		got.AwakeMode != want.AwakeMode ||
		got.Executable != want.Executable ||
		got.Label != want.Label {
		t.Fatalf("state mismatch: got %#v, want %#v", got, want)
	}
	if got.ProcessStartedAt == nil || !got.ProcessStartedAt.Equal(processStartedAt) {
		t.Fatalf("process start = %v, want %v", got.ProcessStartedAt, processStartedAt)
	}
	if got.AutoStopAt == nil || !got.AutoStopAt.Equal(autoStopAt) {
		t.Fatalf("auto stop = %v, want %v", got.AutoStopAt, autoStopAt)
	}

	if err := store.Remove(); err != nil {
		t.Fatalf("remove state: %v", err)
	}
	_, ok, err = store.Read()
	if err != nil {
		t.Fatalf("read removed state: %v", err)
	}
	if ok {
		t.Fatal("expected state to be removed")
	}
}

func TestStoreWriteReplacesExistingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NoSleepCLI", "state.json")
	store := NewStore(path)

	processStartedAt := time.Date(2026, 4, 24, 15, 59, 59, 0, time.UTC)
	first := State{
		PID:              1234,
		StartedAt:        time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC),
		ProcessStartedAt: &processStartedAt,
		Mode:             ModeOpenEnded,
		AwakeMode:        AwakeModeSystemDisplay,
		Executable:       `C:\Tools\nosleep\nosleep.exe`,
	}
	second := first
	second.PID = 5678
	second.Mode = ModeTimed

	if err := store.Write(first); err != nil {
		t.Fatalf("write first state: %v", err)
	}
	if err := store.Write(second); err != nil {
		t.Fatalf("write second state: %v", err)
	}

	got, ok, err := store.Read()
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !ok {
		t.Fatal("expected state")
	}
	if got.PID != second.PID || got.Mode != second.Mode {
		t.Fatalf("state = %#v, want %#v", got, second)
	}
}

func TestLabels(t *testing.T) {
	if got := ModeLabel(ModeTimed); got != "Timed session" {
		t.Fatalf("ModeLabel(%q) = %q", ModeTimed, got)
	}
	if got := ModeLabel(ModeUntil); got != "Until time" {
		t.Fatalf("ModeLabel(%q) = %q", ModeUntil, got)
	}
	if got := ModeLabel(ModeOpenEnded); got != "Open-ended" {
		t.Fatalf("ModeLabel(%q) = %q", ModeOpenEnded, got)
	}
	if got := AwakeModeLabel(AwakeModeSystemDisplay); got != "System + Display" {
		t.Fatalf("AwakeModeLabel = %q", got)
	}
}

func TestStoreRejectsInvalidState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"pid":0}`), 0o600); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}

	_, _, err := NewStore(path).Read()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("error = %v, want ErrInvalidState", err)
	}
}
