package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModelDefaultsOpenEndedSession(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.Local)

	got := initialModel(Session{
		StartedAt: startedAt,
		Label:     "generic",
	})

	if got.kind != "Open-ended" {
		t.Fatalf("kind = %q, want Open-ended", got.kind)
	}
	if !got.indefinite {
		t.Fatal("indefinite should be true")
	}
	if got.duration != 0 {
		t.Fatalf("duration = %v, want 0", got.duration)
	}
	if !got.startedAt.Equal(startedAt) {
		t.Fatalf("startedAt = %v, want %v", got.startedAt, startedAt)
	}
	if got.displayLabel() != "Default" {
		t.Fatalf("displayLabel = %q, want Default", got.displayLabel())
	}
}

func TestInitialModelDerivesDurationFromAutoStop(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.Local)
	autoStopAt := startedAt.Add(90 * time.Minute)

	got := initialModel(Session{
		StartedAt:  startedAt,
		AutoStopAt: autoStopAt,
		Kind:       "Timed session",
		Label:      "Monitoring",
		WatchMode:  true,
	})

	if got.duration != 90*time.Minute {
		t.Fatalf("duration = %v, want 90m", got.duration)
	}
	if got.indefinite {
		t.Fatal("indefinite should be false")
	}
	if got.kind != "Timed session" {
		t.Fatalf("kind = %q, want Timed session", got.kind)
	}
	if got.label != "Monitoring" {
		t.Fatalf("label = %q, want Monitoring", got.label)
	}
	if !got.watchMode {
		t.Fatal("watchMode should be true")
	}
}

func TestModelTimeCalculations(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.Local)
	m := initialModel(Session{
		StartedAt: startedAt,
		Duration:  2 * time.Hour,
	})
	m.indefinite = false
	m.now = startedAt.Add(45 * time.Minute)

	if got := m.elapsed(); got != 45*time.Minute {
		t.Fatalf("elapsed = %v, want 45m", got)
	}
	if got := m.remaining(); got != 75*time.Minute {
		t.Fatalf("remaining = %v, want 75m", got)
	}
	if got := m.percentDone(); got != 0.375 {
		t.Fatalf("percentDone = %v, want 0.375", got)
	}

	m.now = startedAt.Add(3 * time.Hour)
	if got := m.remaining(); got != 0 {
		t.Fatalf("remaining after expiry = %v, want 0", got)
	}

	m.now = startedAt.Add(-1 * time.Minute)
	if got := m.elapsed(); got != 0 {
		t.Fatalf("elapsed before start = %v, want 0", got)
	}
}

func TestModelAutoStopText(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 23, 0, 0, 0, time.Local)

	tests := []struct {
		name       string
		autoStopAt time.Time
		want       string
	}{
		{name: "none", want: "None"},
		{
			name:       "same day",
			autoStopAt: time.Date(2026, 4, 28, 23, 30, 0, 0, time.Local),
			want:       "23:30:00",
		},
		{
			name:       "different day",
			autoStopAt: time.Date(2026, 4, 29, 0, 30, 0, 0, time.Local),
			want:       "2026-04-29 00:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				startedAt:  startedAt,
				autoStopAt: tt.autoStopAt,
			}

			if got := m.autoStopText(); got != tt.want {
				t.Fatalf("autoStopText = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateHandlesWindowSizeQuitAndCompletion(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.Local)
	m := initialModel(Session{
		StartedAt:  startedAt,
		AutoStopAt: startedAt.Add(time.Hour),
	})

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Fatal("window size update should not return a command")
	}
	m = updated.(model)
	if m.width != 100 || m.height != 30 {
		t.Fatalf("size = %dx%d, want 100x30", m.width, m.height)
	}
	if m.progress.Width != 52 {
		t.Fatalf("progress width = %d, want 52", m.progress.Width)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit key should return a command")
	}
	m = updated.(model)
	if !m.quitting {
		t.Fatal("quitting should be true after quit key")
	}

	m = initialModel(Session{
		StartedAt:  startedAt,
		AutoStopAt: startedAt.Add(time.Hour),
	})
	updated, cmd = m.Update(tickMsg(startedAt.Add(time.Hour + time.Second)))
	if cmd == nil {
		t.Fatal("completed tick should return a command")
	}
	m = updated.(model)
	if !m.done {
		t.Fatal("done should be true after auto-stop time")
	}
}

func TestViewStatesAndDashboardContent(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.Local)
	m := initialModel(Session{
		StartedAt:  startedAt,
		AutoStopAt: startedAt.Add(time.Hour),
		Kind:       "Timed session",
		Label:      "Monitoring",
	})
	m.now = startedAt.Add(15 * time.Minute)

	if got := m.View(); got != "Loading..." {
		t.Fatalf("view before sizing = %q, want Loading...", got)
	}

	m.width = 100
	m.height = 30
	got := m.dashboardView()
	for _, want := range []string{"NOSLEEP", "Active Session", "Timed session", "Monitoring", "Auto-stop", "Remaining"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dashboardView missing %q in %q", want, got)
		}
	}

	m.watchMode = true
	got = m.dashboardView()
	if !strings.Contains(got, "Watching Session") {
		t.Fatalf("watch dashboard missing title in %q", got)
	}

	m.done = true
	if got := m.View(); got != "" {
		t.Fatalf("done view = %q, want empty", got)
	}
}

