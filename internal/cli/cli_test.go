package cli

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
			if got.StartedAt.IsZero() {
				t.Fatal("started at should not be zero")
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() {
		Version = oldVersion
	})

	Version = "v1.2.3"

	if got := VersionString(); got != "nosleep v1.2.3" {
		t.Fatalf("version string = %q, want %q", got, "nosleep v1.2.3")
	}
}

func TestStartSessionTUISession(t *testing.T) {
	autoStopAt := time.Date(2026, 4, 24, 18, 0, 0, 0, time.Local)
	startedAt := time.Date(2026, 4, 24, 16, 0, 0, 0, time.Local)
	session := Session{
		Duration:   2 * time.Hour,
		StartedAt:  startedAt,
		AutoStopAt: &autoStopAt,
		Mode:       coresession.ModeTimed,
		Label:      "Monitoring",
	}

	got := session.tuiSession(false)
	if got.Kind != "Timed session" {
		t.Fatalf("kind = %q, want Timed session", got.Kind)
	}
	if !got.AutoStopAt.Equal(autoStopAt) {
		t.Fatalf("auto stop = %v, want %v", got.AutoStopAt, autoStopAt)
	}
	if !got.StartedAt.Equal(startedAt) {
		t.Fatalf("started at = %v, want %v", got.StartedAt, startedAt)
	}
	if got.Label != "Monitoring" {
		t.Fatalf("label = %q, want Monitoring", got.Label)
	}
	if got.WatchMode {
		t.Fatal("watch mode should be false")
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
