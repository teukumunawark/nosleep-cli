package session

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestProcessStartedAt(t *testing.T) {
	pid := os.Getpid()
	startedAt, err := ProcessStartedAt(pid)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if startedAt.IsZero() {
		t.Fatalf("expected non-zero start time")
	}
}

func TestProcessMatches(t *testing.T) {
	pid := os.Getpid()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}
	startedAt, err := ProcessStartedAt(pid)
	if err != nil {
		t.Fatalf("failed to get start time: %v", err)
	}

	tests := []struct {
		name       string
		pid        int
		executable string
		startedAt  *time.Time
		expected   bool
	}{
		{
			name:       "exact match",
			pid:        pid,
			executable: executable,
			startedAt:  &startedAt,
			expected:   true,
		},
		{
			name:       "wrong executable",
			pid:        pid,
			executable: "C:\\Windows\\System32\\cmd.exe",
			startedAt:  &startedAt,
			expected:   false,
		},
		{
			name:       "wrong start time",
			pid:        pid,
			executable: executable,
			startedAt:  func() *time.Time { t := startedAt.Add(time.Hour); return &t }(),
			expected:   false,
		},
		{
			name:       "dead pid",
			pid:        999999,
			executable: executable,
			startedAt:  &startedAt,
			expected:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches, err := ProcessMatches(tc.pid, tc.executable, tc.startedAt)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if matches != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, matches)
			}
		})
	}
}

func TestDummyProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(10 * time.Second)
	os.Exit(0)
}

func TestKillMatchingProcess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestDummyProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start dummy process: %v", err)
	}

	pid := cmd.Process.Pid
	executable, err := os.Executable()
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to get executable: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	startedAt, err := ProcessStartedAt(pid)
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to get process start time: %v", err)
	}

	matches, err := ProcessMatches(pid, executable, &startedAt)
	if err != nil || !matches {
		_ = cmd.Process.Kill()
		t.Fatalf("expected process to match (matches=%v, err=%v)", matches, err)
	}

	err = KillMatchingProcess(pid, executable, &startedAt)
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("expected no error killing process, got %v", err)
	}

	_ = cmd.Wait()

	err = KillMatchingProcess(pid, executable, &startedAt)
	if !errors.Is(err, ErrProcessNotRunning) {
		t.Errorf("expected ErrProcessNotRunning, got %v", err)
	}
}
