package consoleui

import (
	"strings"
	"testing"
	"time"

	coresession "nosleep-cli/internal/session"
)

func TestBackgroundStartedOutput(t *testing.T) {
	startedAt := time.Date(2026, 4, 24, 10, 27, 40, 0, time.Local)
	autoStopAt := startedAt.Add(time.Minute)

	got := BackgroundStartedOutput(coresession.ModeTimed, "generic", &autoStopAt, startedAt)
	for _, want := range []string{
		"NoSleep started",
		"Status     Active in background",
		"Mode       Timed session",
		"Auto-stop  10:28:40",
		"Next:",
		"nosleep status",
		"nosleep stop",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestStatusOutput(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 28, 0, 0, time.Local)
	autoStopAt := now.Add(40 * time.Second)
	processStartedAt := now.Add(-21 * time.Second)
	state := coresession.State{
		PID:              26060,
		StartedAt:        now.Add(-20 * time.Second),
		ProcessStartedAt: &processStartedAt,
		Mode:             coresession.ModeTimed,
		AwakeMode:        coresession.AwakeModeSystemDisplay,
		AutoStopAt:       &autoStopAt,
	}

	got := StatusOutput(state, now)
	for _, want := range []string{
		"NoSleep status",
		"Status     Active",
		"Elapsed    00:00:20",
		"Remaining  00:00:40",
		"Awake      System + Display",
		"PID        26060",
		"nosleep stop",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestStoppedOutput(t *testing.T) {
	got := StoppedOutput()
	for _, want := range []string{
		"NoSleep stopped",
		"Normal Windows sleep behavior restored.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}
