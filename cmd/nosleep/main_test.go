package main

import (
	"strings"
	"testing"
	"time"

	coresession "nosleep-cli/internal/session"
)

func TestNewSession(t *testing.T) {
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.Local)

	tests := []struct {
		name       string
		duration   string
		until      string
		wantMode   string
		wantStopAt time.Time
	}{
		{
			name:     "open ended",
			wantMode: coresession.ModeOpenEnded,
		},
		{
			name:       "duration",
			duration:   "1h30m",
			wantMode:   coresession.ModeTimed,
			wantStopAt: now.Add(90 * time.Minute),
		},
		{
			name:       "until",
			until:      "17:30",
			wantMode:   coresession.ModeUntil,
			wantStopAt: time.Date(2026, 4, 24, 17, 30, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSession(tt.duration, tt.until, "test", now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Mode != tt.wantMode {
				t.Fatalf("mode = %q, want %q", got.Mode, tt.wantMode)
			}
			if got.AutoStopAt == nil {
				if !tt.wantStopAt.IsZero() {
					t.Fatalf("auto stop = nil, want %v", tt.wantStopAt)
				}
			} else if !got.AutoStopAt.Equal(tt.wantStopAt) {
				t.Fatalf("auto stop = %v, want %v", got.AutoStopAt, tt.wantStopAt)
			}
			if got.Label != "test" {
				t.Fatalf("label = %q, want test", got.Label)
			}
		})
	}
}

func TestStartSessionTUISession(t *testing.T) {
	autoStopAt := time.Date(2026, 4, 24, 18, 0, 0, 0, time.Local)
	session := startSession{
		Duration:   2 * time.Hour,
		AutoStopAt: &autoStopAt,
		Mode:       coresession.ModeTimed,
		Label:      "Monitoring",
	}

	got := session.tuiSession()
	if got.Kind != "Timed session" {
		t.Fatalf("kind = %q, want Timed session", got.Kind)
	}
	if !got.AutoStopAt.Equal(autoStopAt) {
		t.Fatalf("auto stop = %v, want %v", got.AutoStopAt, autoStopAt)
	}
	if got.Label != "Monitoring" {
		t.Fatalf("label = %q, want Monitoring", got.Label)
	}
}

func TestNewSessionRejectsInvalidInputs(t *testing.T) {
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, time.Local)

	tests := []struct {
		name     string
		duration string
		until    string
		wantText string
	}{
		{name: "bad duration", duration: "abc", wantText: "Invalid duration: abc"},
		{name: "bad until", until: "99:99", wantText: "Invalid time: 99:99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSession(tt.duration, tt.until, "test", now)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("error = %q, want text %q", err, tt.wantText)
			}
		})
	}
}

func TestBackgroundStartedOutput(t *testing.T) {
	startedAt := time.Date(2026, 4, 24, 10, 27, 40, 0, time.Local)
	autoStopAt := startedAt.Add(time.Minute)
	session := startSession{
		AutoStopAt: &autoStopAt,
		Mode:       coresession.ModeTimed,
	}

	got := backgroundStartedOutput(session, startedAt)
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

	got := statusOutput(state, now)
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
	got := stoppedOutput()
	for _, want := range []string{
		"NoSleep stopped",
		"Normal Windows sleep behavior restored.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}
