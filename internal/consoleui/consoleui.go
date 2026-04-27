package consoleui

import (
	"fmt"
	"strings"
	"time"

	coresession "nosleep-cli/internal/session"
	"nosleep-cli/internal/timer"
)

type outputRow struct {
	Label string
	Value string
}

func PrintAlreadyRunning(state coresession.State) {
	fmt.Print(AlreadyRunningOutput(state, time.Now()))
}

func PrintStatus(state coresession.State, now time.Time) {
	fmt.Print(StatusOutput(state, now))
}

func PrintBackgroundStarted(mode, label string, autoStopAt *time.Time, startedAt time.Time) {
	fmt.Print(BackgroundStartedOutput(mode, label, autoStopAt, startedAt))
}

func BackgroundStartedOutput(mode, label string, autoStopAt *time.Time, startedAt time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Active in background"},
		{Label: "Mode", Value: coresession.ModeLabel(mode)},
	}
	if label != "" && label != "generic" {
		rows = append(rows, outputRow{Label: "Label", Value: label})
	}
	if autoStopAt == nil {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: "None"})
	} else {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: FormatClock(*autoStopAt, startedAt)})
	}

	return RenderOutput("NoSleep started", rows, []string{"nosleep status", "nosleep stop"}, nil)
}

func AlreadyRunningOutput(state coresession.State, now time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Already running"},
		{Label: "Mode", Value: coresession.ModeLabel(state.Mode)},
	}
	if state.Label != "" && state.Label != "generic" {
		rows = append(rows, outputRow{Label: "Label", Value: state.Label})
	}
	if state.AutoStopAt != nil {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: FormatClock(*state.AutoStopAt, now)})
	}

	return RenderOutput(
		"NoSleep is already running",
		rows,
		[]string{"nosleep status", "nosleep stop"},
		[]string{"Stop the active session before starting another one."},
	)
}

func StatusOutput(state coresession.State, now time.Time) string {
	rows := []outputRow{
		{Label: "Status", Value: "Active"},
		{Label: "Mode", Value: coresession.ModeLabel(state.Mode)},
	}
	if state.Label != "" && state.Label != "generic" {
		rows = append(rows, outputRow{Label: "Label", Value: state.Label})
	}
	rows = append(rows,
		outputRow{Label: "Started", Value: FormatClock(state.StartedAt, now)},
		outputRow{Label: "Elapsed", Value: timer.FormatDuration(now.Sub(state.StartedAt))},
	)
	if state.AutoStopAt != nil {
		rows = append(rows,
			outputRow{Label: "Remaining", Value: timer.FormatDuration(state.AutoStopAt.Sub(now))},
			outputRow{Label: "Auto-stop", Value: FormatClock(*state.AutoStopAt, now)},
		)
	} else {
		rows = append(rows, outputRow{Label: "Auto-stop", Value: "None"})
	}
	rows = append(rows,
		outputRow{Label: "Awake", Value: coresession.AwakeModeLabel(state.AwakeMode)},
		outputRow{Label: "PID", Value: fmt.Sprintf("%d", state.PID)},
	)

	return RenderOutput("NoSleep status", rows, []string{"nosleep stop"}, nil)
}

func StoppedOutput() string {
	return RenderOutput(
		"NoSleep stopped",
		nil,
		nil,
		[]string{"Normal Windows sleep behavior restored."},
	)
}

func RenderOutput(title string, rows []outputRow, commands []string, notes []string) string {
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n")

	if len(rows) > 0 {
		b.WriteString("\n")
		for _, row := range rows {
			b.WriteString(fmt.Sprintf("  %-10s %s\n", row.Label, row.Value))
		}
	}

	if len(notes) > 0 {
		b.WriteString("\n")
		for _, note := range notes {
			b.WriteString("  ")
			b.WriteString(note)
			b.WriteString("\n")
		}
	}

	if len(commands) > 0 {
		b.WriteString("\n")
		b.WriteString("Next:\n")
		for _, command := range commands {
			b.WriteString("  ")
			b.WriteString(command)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func FormatClock(t time.Time, reference time.Time) string {
	if timer.SameDate(t, reference) {
		return t.Format("15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}
