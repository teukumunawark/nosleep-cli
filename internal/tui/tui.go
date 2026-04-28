package tui

import (
	"strings"
	"time"

	"nosleep-cli/internal/timer"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	animationFPS = 10
)

var (
	accentColor = lipgloss.Color("214")
	greenColor  = lipgloss.Color("42")
	yellowColor = lipgloss.Color("220")
	redColor    = lipgloss.Color("196")
	mutedColor  = lipgloss.Color("244")
	whiteColor  = lipgloss.Color("15")

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 4).
			Width(60)

	kickerStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginRight(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(whiteColor).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(whiteColor)

	helpStyle = lipgloss.NewStyle().
			MarginTop(1)
)

type tickMsg time.Time

type keyMap struct {
	Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "stop nosleep"),
	),
}

var watchKeys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "close monitor"),
	),
}

type Session struct {
	Duration   time.Duration
	StartedAt  time.Time
	AutoStopAt time.Time
	Kind       string
	Label      string
	WatchMode  bool
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
	watchMode  bool
	spinner    spinner.Model
	progress   progress.Model
	help       help.Model
}

func initialModel(session Session) model {
	now := time.Now()
	kind := session.Kind
	if kind == "" {
		kind = "Open-ended"
	}

	startedAt := session.StartedAt
	if startedAt.IsZero() {
		startedAt = now
	}

	autoStopAt := session.AutoStopAt
	duration := session.Duration
	if !autoStopAt.IsZero() {
		duration = autoStopAt.Sub(startedAt)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
		progress.WithWidth(40),
	)

	return model{
		duration:   duration,
		kind:       kind,
		label:      session.Label,
		indefinite: autoStopAt.IsZero() && duration == 0,
		startedAt:  startedAt,
		autoStopAt: autoStopAt,
		now:        now,
		spinner:    s,
		progress:   p,
		help:       help.New(),
		watchMode:  session.WatchMode,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.spinner.Tick)
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
		m.progress.Width = minInt(msg.Width-20, 52)
		return m, nil

	case tea.KeyMsg:
		k := keys
		if m.watchMode {
			k = watchKeys
		}
		if key.Matches(msg, k.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		m.now = time.Time(msg)

		if !m.indefinite && m.remaining() <= 0 {
			m.done = true
			return m, tea.Quit
		}

		return m, tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if pm, ok := newModel.(progress.Model); ok {
			m.progress = pm
		}
		return m, cmd
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

	content := m.dashboardView()

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) dashboardView() string {
	headerTitle := "Active Session"
	if m.watchMode {
		headerTitle = "Watching Session"
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		m.spinner.View(),
		kickerStyle.Render("NOSLEEP"),
		titleStyle.Render(headerTitle),
	)

	rows := []string{
		m.renderRow("Mode", m.kind),
		m.renderRow("Label", m.displayLabel()),
		m.renderRow("Started", m.startedAt.Format("15:04:05")),
		m.renderRow("Elapsed", timer.FormatDuration(m.elapsed())),
	}

	if !m.indefinite {
		rows = append(rows, m.renderRow("Auto-stop", m.autoStopText()))
		rows = append(rows, m.renderRow("Remaining", m.remainingStyled()))
		rows = append(rows, "")
		rows = append(rows, m.progress.ViewAs(m.percentDone()))
	} else {
		rows = append(rows, m.renderRow("Auto-stop", "None"))
	}

	activeKeys := keys
	if m.watchMode {
		activeKeys = watchKeys
	}
	helpView := helpStyle.Render(m.help.View(activeKeys))

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(rows, "\n"),
		helpView,
	)

	return cardStyle.Render(body)
}

func (m model) renderRow(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render(label),
		valueStyle.Render(value),
	)
}

func (m model) displayLabel() string {
	if m.isGenericLabel() {
		return "Default"
	}
	return m.label
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

func (m model) percentDone() float64 {
	if m.indefinite || m.duration == 0 {
		return 0
	}
	return float64(m.elapsed()) / float64(m.duration)
}

func (m model) remainingStyled() string {
	rem := m.remaining()
	remStr := timer.FormatDuration(rem)

	percent := 1.0 - m.percentDone()

	style := lipgloss.NewStyle().Bold(true)
	if percent < 0.2 {
		style = style.Foreground(redColor)
	} else if percent < 0.5 {
		style = style.Foreground(yellowColor)
	} else {
		style = style.Foreground(greenColor)
	}

	return style.Render(remStr)
}

func (m model) autoStopText() string {
	if m.autoStopAt.IsZero() {
		return "None"
	}
	if timer.SameDate(m.startedAt, m.autoStopAt) {
		return m.autoStopAt.Format("15:04:05")
	}
	return m.autoStopAt.Format("2006-01-02 15:04:05")
}

func (m model) isGenericLabel() bool {
	l := strings.TrimSpace(strings.ToLower(m.label))
	return l == "" || l == "generic"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Start(session Session) error {
	p := tea.NewProgram(initialModel(session), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
