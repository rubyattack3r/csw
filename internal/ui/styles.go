package ui

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// AppTheme defines the color palette used throughout the TUI.
type AppTheme struct {
	Primary    string
	Secondary  string
	Accent     string
	Text       string
	Subtle     string
	Error      string
	Warning    string
	Success    string
	Background string
	Surface    string
}

// TerminalTheme returns the default Cobalt Blue color scheme.
func TerminalTheme() AppTheme {
	return AppTheme{
		Primary:    "#0047AB", // Deep Cobalt
		Secondary:  "#002366", // Royal Navy
		Accent:     "#00BFFF", // Deep Sky Blue
		Text:       "#A9C9FF", // Icy Blue-White
		Subtle:     "#3B4B61", // Dim Slate
		Error:      "#ff5555", // red
		Warning:    "#f1fa8c", // yellow
		Success:    "#50fa7b", // green
		Background: "#050A10", // Darkest Navy
		Surface:    "#0A1117", // Midnight Surface
	}
}

// NewStyles initializes the TUI style tokens based on the provided theme.
func NewStyles(theme AppTheme) Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true).
			MarginLeft(1).
			MarginBottom(1),

		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)),

		Bold: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)).
			Bold(true),

		Subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Subtle)),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)),

		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Warning)),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E6ED")).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 1),

		Key: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),

		SpinnerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Bold(true),

		HighlightButton: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E6ED")).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 2).
			Bold(true),

		SelectedOption: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),

		CodeBlock: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Surface)).
			Foreground(lipgloss.Color(theme.Text)).
			Padding(1, 2).
			MarginLeft(2),

		Accent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),

		Banner: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true).
			MarginBottom(1),

		InputActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(0, 1),

		InputNormal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Subtle)).
			Padding(0, 1),
	}
}

// Styles holds the lipgloss styling definitions for all TUI components.
type Styles struct {
	Title           lipgloss.Style
	Normal          lipgloss.Style
	Bold            lipgloss.Style
	Subtle          lipgloss.Style
	Warning         lipgloss.Style
	Error           lipgloss.Style
	StatusBar       lipgloss.Style
	Key             lipgloss.Style
	SpinnerStyle    lipgloss.Style
	Success         lipgloss.Style
	HighlightButton lipgloss.Style
	SelectedOption  lipgloss.Style
	CodeBlock       lipgloss.Style
	Accent          lipgloss.Style
	Banner          lipgloss.Style
	InputActive     lipgloss.Style
	InputNormal     lipgloss.Style
}

// NewThemedProgress creates a new progress bar model styled according to the theme.
func (s Styles) NewThemedProgress(width int) progress.Model {
	theme := TerminalTheme()
	prog := progress.New(
		progress.WithGradient(theme.Secondary, theme.Primary),
		progress.WithoutPercentage(), // We prefer showing context
	)

	prog.Width = width
	return prog
}
