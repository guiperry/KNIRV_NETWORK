package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
)

// ProgressBar is a component that displays a progress bar
type ProgressBar struct {
	progress progress.Model
	styles   ui.Styles
	width    int
	percent  float64
	label    string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(styles ui.Styles, width int) ProgressBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	return ProgressBar{
		progress: p,
		styles:   styles,
		width:    width,
		percent:  0,
		label:    "",
	}
}

// SetPercent sets the progress percentage
func (p *ProgressBar) SetPercent(percent float64) {
	p.percent = percent
}

// SetLabel sets the progress label
func (p *ProgressBar) SetLabel(label string) {
	p.label = label
}

// View renders the progress bar
func (p ProgressBar) View() string {
	if p.label != "" {
		return fmt.Sprintf("%s\n%s", p.label, p.progress.ViewAs(p.percent))
	}
	return p.progress.ViewAs(p.percent)
}

// Spinner is a component that displays a spinner
type Spinner struct {
	spinner spinner.Model
	styles  ui.Styles
	label   string
}

// NewSpinner creates a new spinner
func NewSpinner(styles ui.Styles) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Theme.Primary)

	return Spinner{
		spinner: s,
		styles:  styles,
		label:   "",
	}
}

// SetLabel sets the spinner label
func (s *Spinner) SetLabel(label string) {
	s.label = label
}

// Tick advances the spinner animation
func (s *Spinner) Tick() tea.Cmd {
	return s.spinner.Tick
}

// View renders the spinner
func (s Spinner) View() string {
	if s.label != "" {
		return fmt.Sprintf("%s %s", s.spinner.View(), s.label)
	}
	return s.spinner.View()
}

// LoadingMessage is a component that displays a loading message with a spinner
type LoadingMessage struct {
	spinner Spinner
	styles  ui.Styles
	message string
}

// NewLoadingMessage creates a new loading message
func NewLoadingMessage(styles ui.Styles, message string) LoadingMessage {
	return LoadingMessage{
		spinner: NewSpinner(styles),
		styles:  styles,
		message: message,
	}
}

// SetMessage sets the loading message
func (l *LoadingMessage) SetMessage(message string) {
	l.message = message
}

// Tick advances the spinner animation
func (l *LoadingMessage) Tick() tea.Cmd {
	return l.spinner.Tick()
}

// View renders the loading message
func (l LoadingMessage) View() string {
	return fmt.Sprintf("%s %s", l.spinner.View(), l.message)
}

// TimedProgressBar is a component that displays a progress bar with time estimation
type TimedProgressBar struct {
	progress    progress.Model
	styles      ui.Styles
	width       int
	percent     float64
	label       string
	startTime   time.Time
	elapsedTime time.Duration
	showETA     bool
}

// NewTimedProgressBar creates a new timed progress bar
func NewTimedProgressBar(styles ui.Styles, width int) TimedProgressBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	return TimedProgressBar{
		progress:  p,
		styles:    styles,
		width:     width,
		percent:   0,
		label:     "",
		startTime: time.Now(),
		showETA:   true,
	}
}

// SetPercent sets the progress percentage
func (p *TimedProgressBar) SetPercent(percent float64) {
	p.percent = percent
	p.elapsedTime = time.Since(p.startTime)
}

// SetLabel sets the progress label
func (p *TimedProgressBar) SetLabel(label string) {
	p.label = label
}

// SetShowETA sets whether to show the ETA
func (p *TimedProgressBar) SetShowETA(showETA bool) {
	p.showETA = showETA
}

// View renders the timed progress bar
func (p TimedProgressBar) View() string {
	progressView := p.progress.ViewAs(p.percent)

	if !p.showETA {
		if p.label != "" {
			return fmt.Sprintf("%s\n%s", p.label, progressView)
		}
		return progressView
	}

	// Calculate ETA
	var etaString string
	if p.percent > 0 {
		totalTime := float64(p.elapsedTime) / p.percent
		remainingTime := time.Duration(totalTime - float64(p.elapsedTime))
		etaString = fmt.Sprintf("ETA: %s", formatDuration(remainingTime))
	} else {
		etaString = "ETA: calculating..."
	}

	// Format elapsed time
	elapsedString := fmt.Sprintf("Elapsed: %s", formatDuration(p.elapsedTime))

	// Combine views
	if p.label != "" {
		return fmt.Sprintf("%s\n%s\n%s %s", p.label, progressView, elapsedString, etaString)
	}
	return fmt.Sprintf("%s\n%s %s", progressView, elapsedString, etaString)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// MultiProgressBar is a component that displays multiple progress bars
type MultiProgressBar struct {
	bars   []ProgressBar
	styles ui.Styles
	width  int
	labels []string
}

// NewMultiProgressBar creates a new multi progress bar
func NewMultiProgressBar(styles ui.Styles, width int, count int) MultiProgressBar {
	bars := make([]ProgressBar, count)
	labels := make([]string, count)

	for i := 0; i < count; i++ {
		bars[i] = NewProgressBar(styles, width)
		labels[i] = fmt.Sprintf("Task %d", i+1)
	}

	return MultiProgressBar{
		bars:   bars,
		styles: styles,
		width:  width,
		labels: labels,
	}
}

// SetPercent sets the progress percentage for a specific bar
func (m *MultiProgressBar) SetPercent(index int, percent float64) {
	if index >= 0 && index < len(m.bars) {
		m.bars[index].SetPercent(percent)
	}
}

// SetLabel sets the label for a specific bar
func (m *MultiProgressBar) SetLabel(index int, label string) {
	if index >= 0 && index < len(m.bars) {
		m.labels[index] = label
		m.bars[index].SetLabel(label)
	}
}

// View renders the multi progress bar
func (m MultiProgressBar) View() string {
	var sb strings.Builder

	for i, bar := range m.bars {
		bar.SetLabel(m.labels[i])
		sb.WriteString(bar.View())
		if i < len(m.bars)-1 {
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}
