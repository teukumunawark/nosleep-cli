package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const animationFPS = 4

var (
	appStyle = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{})

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

	kickerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

type tickMsg time.Time

const (
	SessionOpenEnded = "Open-ended"
	SessionTimed     = "Timed session"
	SessionUntil     = "Until time"
)

type Session struct {
	Duration   time.Duration
	AutoStopAt time.Time
	Kind       string
	Label      string
}

type model struct {
	width      int
	height     int
	duration   time.Duration
	kind       string
	label      string
	indefinite bool
	startedAt  time.Time
	autoStopAt time.Time
	now        time.Time
	quitting   bool
	done       bool
	frame      int
}

func initialModel(session Session) model {
	now := time.Now()
	kind := session.Kind
	if kind == "" {
		kind = SessionOpenEnded
	}
	startedAt := now
	autoStopAt := session.AutoStopAt
	duration := session.Duration
	if !autoStopAt.IsZero() {
		duration = autoStopAt.Sub(startedAt)
	}

	return model{
		duration:   duration,
		kind:       kind,
		label:      session.Label,
		indefinite: autoStopAt.IsZero() && duration == 0,
		startedAt:  now,
		autoStopAt: autoStopAt,
		now:        now,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/animationFPS, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		m.now = time.Time(msg)
		m.frame++

		if !m.indefinite && m.remaining() <= 0 {
			m.done = true
			return m, tea.Quit
		}

		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting || m.done {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.width < 36 || m.height < 10 {
		return appStyle.Width(m.width).Height(m.height).Render(m.compactView())
	}

	contentWidth := minInt(maxInt(32, m.width-4), 72)
	body := m.normalView(contentWidth)

	return appStyle.Width(m.width).Height(m.height).Render(
		lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body),
	)
}

func (m model) compactView() string {
	lines := []string{
		"NOSLEEP",
		m.displayTime(),
	}

	if m.width >= 20 {
		lines = append(lines, trimToWidth("Mode: "+m.kind, m.width))
	}

	if m.height >= 5 {
		lines = append(lines, trimToWidth(m.compactMeta(), m.width))
	}

	return strings.Join(lines, "\n")
}

func (m model) normalView(width int) string {
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		kickerStyle.Render("NOSLEEP"),
		titleStyle.Render("Preventing sleep during passive work"),
	)

	rows := []string{
		m.row("Mode", m.kind),
		m.row("Started", m.startedAt.Format("15:04:05")),
		m.row("Elapsed", formatDuration(m.elapsed())),
	}
	if !m.indefinite {
		rows = append(rows, m.row("Remaining", formatDuration(m.remaining())))
	}
	rows = append(rows,
		m.row("Auto-stop", m.autoStopText()),
		m.row("Awake", "System + Display"),
	)

	if !m.isGenericLabel() {
		rows = append(rows, m.row("Label", m.label))
	}

	footerLines := []string{}
	if m.indefinite {
		footerLines = append(footerLines, "Tip: use --duration 1h or --until 18:00")
	}
	footerLines = append(footerLines, "Press Q, ESC, or CTRL+C to stop")

	parts := []string{header, "", strings.Join(rows, "\n"), "", footerStyle.Render(strings.Join(footerLines, "\n"))}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return cardStyle.Width(width).Render(content)
}

func (m model) elapsed() time.Duration {
	if m.now.Before(m.startedAt) {
		return 0
	}
	return m.now.Sub(m.startedAt)
}

func (m model) remaining() time.Duration {
	if m.indefinite {
		return 0
	}

	remaining := m.duration - m.elapsed()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (m model) displayTime() string {
	if m.indefinite {
		return formatDuration(m.elapsed())
	}
	return formatDuration(m.remaining())
}

func (m model) compactMeta() string {
	return m.kind
}

func (m model) autoStopText() string {
	if m.autoStopAt.IsZero() {
		return "None"
	}
	if sameDate(m.startedAt, m.autoStopAt) {
		return m.autoStopAt.Format("15:04:05")
	}
	return m.autoStopAt.Format("2006-01-02 15:04:05")
}

func (m model) row(label, value string) string {
	return fmt.Sprintf("%-11s %s", label, value)
}

func (m model) isGenericLabel() bool {
	return strings.TrimSpace(strings.ToLower(m.label)) == "" || strings.TrimSpace(strings.ToLower(m.label)) == "generic"
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func Start(session Session) error {
	p := tea.NewProgram(initialModel(session), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
