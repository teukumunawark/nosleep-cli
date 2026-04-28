package worker

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	coresession "nosleep-cli/internal/session"
)

func TestBackgroundWorkerArgs(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 123, time.UTC)
	autoStopAt := startedAt.Add(2 * time.Hour)

	tests := []struct {
		name       string
		autoStopAt *time.Time
		want       []string
	}{
		{
			name: "open ended",
			want: []string{
				"start",
				"--background-worker",
				"--started-at", "2026-04-28T09:00:00.000000123Z",
				"--session-mode", coresession.ModeOpenEnded,
				"--mode", "Reading",
			},
		},
		{
			name:       "auto stop",
			autoStopAt: &autoStopAt,
			want: []string{
				"start",
				"--background-worker",
				"--started-at", "2026-04-28T09:00:00.000000123Z",
				"--session-mode", coresession.ModeTimed,
				"--mode", "Monitoring",
				"--auto-stop-at", "2026-04-28T11:00:00.000000123Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := coresession.ModeOpenEnded
			label := "Reading"
			if tt.autoStopAt != nil {
				mode = coresession.ModeTimed
				label = "Monitoring"
			}

			got := backgroundWorkerArgs(startedAt, tt.autoStopAt, mode, label)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("args = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseBackgroundWorkerTimes(t *testing.T) {
	startedAt := "2026-04-28T09:00:00.000000123Z"
	autoStopAt := "2026-04-28T11:00:00.000000123Z"

	gotStartedAt, gotAutoStopAt, err := parseBackgroundWorkerTimes(startedAt, autoStopAt)
	if err != nil {
		t.Fatalf("parse times: %v", err)
	}
	if gotStartedAt.Format(time.RFC3339Nano) != startedAt {
		t.Fatalf("startedAt = %s, want %s", gotStartedAt.Format(time.RFC3339Nano), startedAt)
	}
	if gotAutoStopAt == nil {
		t.Fatal("autoStopAt should not be nil")
	}
	if gotAutoStopAt.Format(time.RFC3339Nano) != autoStopAt {
		t.Fatalf("autoStopAt = %s, want %s", gotAutoStopAt.Format(time.RFC3339Nano), autoStopAt)
	}
}

func TestParseBackgroundWorkerTimesAllowsMissingAutoStop(t *testing.T) {
	startedAt := "2026-04-28T09:00:00Z"

	gotStartedAt, gotAutoStopAt, err := parseBackgroundWorkerTimes(startedAt, "")
	if err != nil {
		t.Fatalf("parse times: %v", err)
	}
	if gotStartedAt.Format(time.RFC3339Nano) != startedAt {
		t.Fatalf("startedAt = %s, want %s", gotStartedAt.Format(time.RFC3339Nano), startedAt)
	}
	if gotAutoStopAt != nil {
		t.Fatalf("autoStopAt = %v, want nil", gotAutoStopAt)
	}
}

func TestParseBackgroundWorkerTimesRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name         string
		startedAtStr string
		autoStopStr  string
		wantText     string
	}{
		{
			name:         "invalid start time",
			startedAtStr: "not-a-time",
			wantText:     "parse background worker start time",
		},
		{
			name:         "invalid auto stop time",
			startedAtStr: "2026-04-28T09:00:00Z",
			autoStopStr:  "not-a-time",
			wantText:     "parse background worker auto-stop time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseBackgroundWorkerTimes(tt.startedAtStr, tt.autoStopStr)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("error = %q, want text %q", err, tt.wantText)
			}
			var parseErr *time.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, want wrapped time.ParseError", err)
			}
		})
	}
}

func TestNormalizeMode(t *testing.T) {
	if got := normalizeMode(""); got != coresession.ModeOpenEnded {
		t.Fatalf("empty mode = %q, want %q", got, coresession.ModeOpenEnded)
	}
	if got := normalizeMode(coresession.ModeTimed); got != coresession.ModeTimed {
		t.Fatalf("timed mode = %q, want %q", got, coresession.ModeTimed)
	}
}

func TestBackgroundWorkerState(t *testing.T) {
	startedAt := time.Date(2026, 4, 28, 9, 0, 0, 0, time.UTC)
	processStartedAt := startedAt.Add(-1 * time.Second)
	autoStopAt := startedAt.Add(30 * time.Minute)

	got := backgroundWorkerState(
		1234,
		startedAt,
		processStartedAt,
		&autoStopAt,
		`C:\Tools\nosleep\nosleep.exe`,
		coresession.ModeTimed,
		"Monitoring",
	)

	if got.PID != 1234 {
		t.Fatalf("pid = %d, want 1234", got.PID)
	}
	if !got.StartedAt.Equal(startedAt) {
		t.Fatalf("startedAt = %v, want %v", got.StartedAt, startedAt)
	}
	if got.ProcessStartedAt == nil || !got.ProcessStartedAt.Equal(processStartedAt) {
		t.Fatalf("processStartedAt = %v, want %v", got.ProcessStartedAt, processStartedAt)
	}
	if got.Mode != coresession.ModeTimed {
		t.Fatalf("mode = %q, want %q", got.Mode, coresession.ModeTimed)
	}
	if got.AwakeMode != coresession.AwakeModeSystemDisplay {
		t.Fatalf("awakeMode = %q, want %q", got.AwakeMode, coresession.AwakeModeSystemDisplay)
	}
	if got.AutoStopAt == nil || !got.AutoStopAt.Equal(autoStopAt) {
		t.Fatalf("autoStopAt = %v, want %v", got.AutoStopAt, autoStopAt)
	}
	if got.Executable != `C:\Tools\nosleep\nosleep.exe` {
		t.Fatalf("executable = %q", got.Executable)
	}
	if got.Label != "Monitoring" {
		t.Fatalf("label = %q, want Monitoring", got.Label)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("state should validate: %v", err)
	}
}

