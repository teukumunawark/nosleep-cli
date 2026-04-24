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

	timerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

type tickMsg time.Time

type model struct {
	width      int
	height     int
	duration   time.Duration
	mode       string
	indefinite bool
	startedAt  time.Time
	now        time.Time
	quitting   bool
	done       bool
	frame      int
}

func initialModel(d time.Duration, mode string) model {
	now := time.Now()
	return model{
		duration:   d,
		mode:       mode,
		indefinite: d == 0,
		startedAt:  now,
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

	contentWidth := min(max(32, m.width-4), 72)
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

	if m.width >= 20 && !m.isGenericMode() {
		lines = append(lines, trimToWidth("Mode: "+m.mode, m.width))
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
		mutedStyle.Render(m.headerMeta(width)),
	)

	timerBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		mutedStyle.Render(m.timerLabel()),
		timerStyle.Render(m.displayTime()),
		mutedStyle.Render(m.timerMeta()),
		mutedStyle.Render(m.protectionMeta()),
	)

	modeLine := ""
	if !m.isGenericMode() {
		modeLine = modeStyle.Render("Mode: " + m.mode)
	}

	footer := footerStyle.Render("Q / ESC / CTRL+C to exit")

	parts := []string{header, "", timerBlock, ""}
	if modeLine != "" {
		parts = append(parts, modeLine, "")
	}
	parts = append(parts, footer)

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

func (m model) progressRatio() float64 {
	if m.indefinite || m.duration <= 0 {
		return 0
	}

	ratio := float64(m.elapsed()) / float64(m.duration)
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func (m model) displayTime() string {
	if m.indefinite {
		return formatDuration(m.elapsed())
	}
	return formatDuration(m.remaining())
}

func (m model) timerLabel() string {
	if m.indefinite {
		return "Elapsed"
	}
	return "Remaining"
}

func (m model) timerMeta() string {
	if m.indefinite {
		return "Running until you stop it"
	}
	return fmt.Sprintf("%.0f%% complete", m.progressRatio()*100)
}

func (m model) protectionMeta() string {
	if m.indefinite {
		return "System and display stay awake"
	}
	return "Sleep prevention is active"
}

func (m model) headerMeta(width int) string {
	meta := fmt.Sprintf("%s  |  %s", m.sessionKind(), m.now.Format("15:04:05"))
	return trimToWidth(meta, width)
}

func (m model) compactMeta() string {
	if m.indefinite {
		return "Open-ended session"
	}
	return "Timed session"
}

func (m model) sessionKind() string {
	if m.indefinite {
		return "Open-ended"
	}
	return "Timed"
}

func (m model) isGenericMode() bool {
	return strings.TrimSpace(strings.ToLower(m.mode)) == "" || strings.TrimSpace(strings.ToLower(m.mode)) == "generic"
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Start(duration time.Duration, mode string) error {
	p := tea.NewProgram(initialModel(duration, mode), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
