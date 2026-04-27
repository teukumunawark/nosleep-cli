package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreReadWriteRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NoSleepCLI", "state.json")
	store := NewStore(path)

	autoStopAt := time.Date(2026, 4, 24, 18, 0, 0, 0, time.UTC)
	want := State{
		PID:        1234,
		StartedAt:  time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC),
		Mode:       ModeTimed,
		AwakeMode:  AwakeModeSystemDisplay,
		AutoStopAt: &autoStopAt,
		Executable: `C:\Tools\nosleep\nosleep.exe`,
		Label:      "Monitoring",
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
