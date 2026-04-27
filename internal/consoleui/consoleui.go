package consoleui

import (
	"fmt"
	"strings"
	"time"

	coresession "nosleep-cli/internal/session"
	"nosleep-cli/internal/timer"

	"github.com/charmbracelet/lipgloss"
)

type outputRow struct {
	Label string
	Value string
}

var (
	accentColor = lipgloss.Color("214")
	greenColor  = lipgloss.Color("42")
	mutedColor  = lipgloss.Color("244")
	whiteColor  = lipgloss.Color("15")

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 4).
			Width(60)

	titleStyle = lipgloss.NewStyle().
			Foreground(whiteColor).
			Bold(true)

	kickerStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginRight(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(whiteColor)

	activeValueStyle = lipgloss.NewStyle().
			Foreground(greenColor).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)
)

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
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		kickerStyle.Render("NOSLEEP"),
		titleStyle.Render(title),
	)

	var rowStrs []string
	for _, row := range rows {
		rowStrs = append(rowStrs, renderRow(row.Label, row.Value))
	}

	var footerLines []string
	for _, note := range notes {
		footerLines = append(footerLines, note)
	}
	if len(commands) > 0 {
		if len(footerLines) > 0 {
			footerLines = append(footerLines, "")
		}
		footerLines = append(footerLines, "Next:")
		for _, cmd := range commands {
			footerLines = append(footerLines, "  "+cmd)
		}
	}

	var parts []string
	parts = append(parts, header, "")
	if len(rowStrs) > 0 {
		parts = append(parts, strings.Join(rowStrs, "\n"))
	}
	if len(footerLines) > 0 {
		parts = append(parts, footerStyle.Render(strings.Join(footerLines, "\n")))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return cardStyle.Render(body) + "\n"
}

func renderRow(label, value string) string {
	vStyle := valueStyle
	if strings.Contains(value, "Active") || value == "Already running" {
		vStyle = activeValueStyle
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render(label),
		vStyle.Render(value),
	)
}

func FormatClock(t time.Time, reference time.Time) string {
	if timer.SameDate(t, reference) {
		return t.Format("15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}
