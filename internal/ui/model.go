package ui

import (
	"fmt"
	"runtime"
	"strings"

	"csw/internal/core"
	"csw/internal/distro"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the various view stages of the installer TUI.
type State int

const (
	StateWelcome State = iota
	StateSelectMode
	StateForm
	StateConfirmOverwrite
	StateConfirmDetails
	StateInstalling
	StateDone
	StateError
)

// Model is the main Bubble Tea model for the installer UI. It tracks
// application state, user inputs, and installation progress.
type Model struct {
	App    *core.App
	Styles Styles
	State  State

	DistroInfo *distro.Info
	Err        error

	Inputs     []textinput.Model
	FocusIndex int

	Progress    progress.Model
	LogMessages []string
	Viewport    viewport.Model
	ShowFullLog bool

	IsAdmin bool

	LocalIPs []string
	IPIndex  int

	ProfileNames []string
	ProfileIndex int

	ShowSecrets bool
}

// NewModel creates and initializes a new Model instance with the provided App core.
func NewModel(app *core.App) Model {
	styles := NewStyles(TerminalTheme())

	// Initialize inputs
	inputs := make([]textinput.Model, 5)

	// License Key
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Enter License Key"
	inputs[0].Focus()
	inputs[0].Prompt = "License: "
	inputs[0].EchoMode = textinput.EchoPassword
	inputs[0].EchoCharacter = '•'
	inputs[0].Width = 60
	inputs[0].CharLimit = 64

	// Install Path
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "/opt/cobaltstrike"
	inputs[1].Prompt = "Path:    "
	inputs[1].Width = 60
	inputs[1].CharLimit = 256

	// Server IP
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Server IP (e.g. 1.2.3.4)"
	inputs[2].Prompt = "IP:      "
	inputs[2].Width = 60
	inputs[2].CharLimit = 64

	// TeamServer Password
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "TeamServer Password"
	inputs[3].Prompt = "Pass:    "
	inputs[3].EchoMode = textinput.EchoPassword
	inputs[3].EchoCharacter = '•'
	inputs[3].Width = 60
	inputs[3].CharLimit = 64

	// Malleable Profile
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Select Malleable Profile"
	inputs[4].Prompt = "Profile: "
	inputs[4].Width = 60
	inputs[4].CharLimit = 128

	dInfo, _ := distro.Detect()

	vp := viewport.New(80, 10)
	vp.Style = styles.CodeBlock

	return Model{
		App:          app,
		Styles:       styles,
		State:        StateWelcome,
		Inputs:       inputs,
		DistroInfo:   dInfo,
		Progress:     styles.NewThemedProgress(40),
		LogMessages:  []string{},
		Viewport:     vp,
		IsAdmin:      app.IsAdmin(),
		LocalIPs:     append([]string{""}, core.GetLocalIPs()...),
		IPIndex:      0,
		ProfileNames: core.GetEmbeddedProfileNames(),
		ProfileIndex: 0,
	}
}

func (m Model) getNumInputs() int {
	if m.App.Config.Mode == core.ModeServer {
		return 5
	}
	return 2
}

// Init performs initial TUI commands.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all incoming TUI messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.State {
		case StateWelcome:
			if msg.String() == "enter" {
				m.State = StateSelectMode
				return m, nil
			}
		case StateSelectMode:
			if msg.String() == "1" { // Client
				_ = m.App.SetMode(core.ModeClient)
				m.Inputs[1].SetValue(m.App.Config.InstallPath)
				m.Inputs[1].SetCursor(0)
				m.State = StateForm
				return m, nil
			} else if msg.String() == "2" { // Server
				if runtime.GOOS != "linux" {
					m.Err = fmt.Errorf("server mode (TeamServer) is only supported on Linux; Windows and macOS environments are not supported")
					m.State = StateError
					return m, nil
				}
				err := m.App.SetMode(core.ModeServer)
				if err != nil {
					m.Err = err
					m.State = StateError
				} else {
					m.Inputs[1].SetValue(m.App.Config.InstallPath)
					m.Inputs[1].SetCursor(0)
					m.State = StateForm
				}
				return m, nil
			}
		case StateForm:
			if cmd := m.handleFormKeys(msg); cmd != nil {
				return m, cmd
			}
		case StateError:
			if msg.String() == "backspace" || msg.String() == "esc" {
				m.State = StateSelectMode
				return m, nil
			}
			if msg.String() == "q" {
				return m, tea.Quit
			}
		case StateConfirmOverwrite:
			if msg.String() == "y" || msg.String() == "Y" {
				// Clean directory and proceed
				path := core.ExpandPath(m.App.Config.InstallPath)
				if err := m.App.CleanDirectory(path); err != nil {
					m.Err = fmt.Errorf("failed to clean directory: %w", err)
					m.State = StateError
					return m, nil
				}
				return m, m.transitionFromForm()
			} else if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" {
				m.State = StateForm
				return m, nil
			}
		case StateConfirmDetails:
			return m, m.handleConfirmationKeys(msg)
		case StateDone:
			if msg.String() == "q" {
				return m, tea.Quit
			}
		}

		// Always handle global quit keys unless in state that overrides them
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.State == StateInstalling || m.State == StateDone || m.State == StateError {
			if msg.String() == "l" {
				m.ShowFullLog = !m.ShowFullLog
				return m, nil
			}
		}

	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd

	case installProgressMsg:
		m.LogMessages = append(m.LogMessages, msg.Log)
		m.Viewport.SetContent(strings.Join(m.LogMessages, "\n"))
		m.Viewport.GotoBottom()

		cmds := []tea.Cmd{waitForProgress(msg.Channel)}
		if msg.Percent >= 0 {
			cmds = append(cmds, m.Progress.SetPercent(msg.Percent))
		}
		return m, tea.Batch(cmds...)

	case installDoneMsg:
		m.LogMessages = append(m.LogMessages, "Setup finished successfully!")
		m.State = StateDone
		return m, m.Progress.SetPercent(1.0)

	case errMsg:
		m.Err = msg.err
		m.State = StateError
		return m, nil
	}

	// Handle Input updates if in Form state
	if m.State == StateForm {
		cmds := make([]tea.Cmd, len(m.Inputs))
		for i := range m.Inputs {
			m.Inputs[i], cmds[i] = m.Inputs[i].Update(msg)
		}
		return m, tea.Batch(cmds...)
	}

	if m.State == StateInstalling || m.State == StateDone || m.State == StateError {
		if m.ShowFullLog {
			var cmd tea.Cmd
			m.Viewport, cmd = m.Viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) handleFormKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "shift+tab", "up", "down":
		// Cycle focus
		s := msg.String()
		if s == "up" || s == "shift+tab" {
			m.FocusIndex--
		} else {
			m.FocusIndex++
		}

		numInputs := m.getNumInputs()
		if m.FocusIndex > numInputs {
			m.FocusIndex = 0
		} else if m.FocusIndex < 0 {
			m.FocusIndex = numInputs
		}

		cmds := make([]tea.Cmd, len(m.Inputs))
		for i := 0; i < len(m.Inputs); i++ {
			if i == m.FocusIndex {
				cmds[i] = m.Inputs[i].Focus()
				m.Inputs[i].PromptStyle = m.Styles.Accent
				m.Inputs[i].TextStyle = m.Styles.Normal
			} else {
				m.Inputs[i].Blur()
				m.Inputs[i].PromptStyle = m.Styles.Subtle
				m.Inputs[i].TextStyle = m.Styles.Subtle
			}
		}
		return tea.Batch(cmds...)

	case "left", "right":
		if m.FocusIndex == 2 && len(m.LocalIPs) > 0 {
			if msg.String() == "left" {
				m.IPIndex--
				if m.IPIndex < 0 {
					m.IPIndex = len(m.LocalIPs) - 1
				}
			} else {
				m.IPIndex++
				if m.IPIndex >= len(m.LocalIPs) {
					m.IPIndex = 0
				}
			}
			m.Inputs[2].SetValue(m.LocalIPs[m.IPIndex])
			m.Inputs[2].SetCursor(len(m.Inputs[2].Value()))
			return nil // Important: return nil to stop event propagation (prevents cursor shift)
		}
		// Cycle Profiles if focused on Profile input (index 4)
		if m.FocusIndex == 4 && len(m.ProfileNames) > 0 {
			if msg.String() == "left" {
				m.ProfileIndex--
				if m.ProfileIndex < 0 {
					m.ProfileIndex = len(m.ProfileNames) - 1
				}
			} else {
				m.ProfileIndex++
				if m.ProfileIndex >= len(m.ProfileNames) {
					m.ProfileIndex = 0
				}
			}
			m.Inputs[4].SetValue(m.ProfileNames[m.ProfileIndex])
			m.Inputs[4].SetCursor(len(m.Inputs[4].Value()))
			return nil
		}

	case "enter":
		if m.FocusIndex >= m.getNumInputs()-1 {
			// Submit form
			m.App.Config.LicenseKey = m.Inputs[0].Value()
			m.App.Config.InstallPath = m.Inputs[1].Value()

			path := core.ExpandPath(m.App.Config.InstallPath)
			empty, err := m.App.IsDirEmpty(path)
			if err != nil {
				m.Err = fmt.Errorf("failed to check if directory is empty: %w", err)
				m.State = StateError
				return nil
			}

			if !empty {
				m.State = StateConfirmOverwrite
				return nil
			}

			return m.transitionFromForm()
		}

		// Move to next input
		m.FocusIndex++
		if m.FocusIndex >= len(m.Inputs) {
			m.FocusIndex = 0
		}
		cmds := make([]tea.Cmd, len(m.Inputs))
		for i := range m.Inputs {
			if i == m.FocusIndex {
				cmds[i] = m.Inputs[i].Focus()
			} else {
				m.Inputs[i].Blur()
			}
		}
		return tea.Batch(cmds...)
	}
	return nil
}

// transitionFromForm validates the input fields and moves the state
// to the final confirmation view.
func (m *Model) transitionFromForm() tea.Cmd {
	// 1. Validation: Ensure no active inputs are empty
	numInputs := m.getNumInputs()
	for i := 0; i < numInputs; i++ {
		val := strings.TrimSpace(m.Inputs[i].Value())
		if val == "" {
			label := strings.TrimSpace(strings.TrimRight(m.Inputs[i].Prompt, ":"))
			m.Err = fmt.Errorf("%s cannot be empty", label)
			return nil
		}
	}
	m.Err = nil // Clear any validation errors

	// 2. Populate config
	m.App.Config.LicenseKey = m.Inputs[0].Value()
	m.App.Config.InstallPath = m.Inputs[1].Value()

	if m.App.Config.Mode == core.ModeServer {
		m.App.Config.ServerIP = m.Inputs[2].Value()
		m.App.Config.ServerPassword = m.Inputs[3].Value()
		m.App.Config.SelectedProfile = m.Inputs[4].Value()
	}

	m.State = StateConfirmDetails
	return nil
}

// handleConfirmationKeys manages keyboard input for the final review screen.
// 'y' starts the install, 'n' returns to the form, and 'v' toggles secrets.
func (m *Model) handleConfirmationKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.State = StateInstalling
		return m.startInstall()
	case "n", "N", "esc":
		m.State = StateForm
		return nil
	case "v", "V":
		m.ShowSecrets = !m.ShowSecrets
		return nil
	}
	return nil
}

// viewConfirmDetails renders the read-only summary of the installation parameters.
func (m *Model) viewConfirmDetails() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title.Render("Review Installation Details"))
	b.WriteString("\n\n")

	mask := func(s string) string {
		if m.ShowSecrets {
			return s
		}
		if s == "" {
			return "(empty)"
		}
		return strings.Repeat("•", 12)
	}

	type summaryItem struct {
		Label  string
		Value  string
		Secret bool
	}

	items := []summaryItem{
		{"License Key", m.App.Config.LicenseKey, true},
		{"Install Path", m.App.Config.InstallPath, false},
	}

	if m.App.Config.Mode == core.ModeServer {
		items = append(items, summaryItem{"Server IP", m.App.Config.ServerIP, false})
		items = append(items, summaryItem{"Server Pass", m.App.Config.ServerPassword, true})
		items = append(items, summaryItem{"C2 Profile", m.App.Config.SelectedProfile, false})
	}

	for _, item := range items {
		val := item.Value
		if item.Secret {
			val = mask(val)
		}
		b.WriteString(fmt.Sprintf("%-15s: %s\n", m.Styles.Accent.Render(item.Label), m.Styles.Normal.Render(val)))
	}

	b.WriteString("\n")
	b.WriteString(m.Styles.Accent.Render("Ready to deploy Cobalt Strike? [y/N]") + "\n\n")

	b.WriteString(
		m.Styles.Subtle.Render("Press ") + m.Styles.Accent.Render("'v'") + m.Styles.Subtle.Render(" to toggle secrets visibility.") + "\n" +
			m.Styles.Subtle.Render("Press ") + m.Styles.Accent.Render("'y'") + m.Styles.Subtle.Render(" to confirm, ") + m.Styles.Accent.Render("'n'") + m.Styles.Subtle.Render(" to go back."),
	)

	return b.String()
}

// View renders the current state of the TUI as a string.
func (m Model) View() string {
	switch m.State {
	case StateWelcome:
		return m.viewWelcome()
	case StateSelectMode:
		return m.viewSelectMode()
	case StateForm:
		return m.viewForm()
	case StateConfirmOverwrite:
		return m.viewConfirmOverwrite()
	case StateConfirmDetails:
		return m.viewConfirmDetails()
	case StateInstalling:
		return m.viewInstalling()
	case StateDone:
		return m.viewDone()
	case StateError:
		return m.viewError()
	default:
		return "Unknown State"
	}
}

func (m Model) viewSelectMode() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title.Render("Select Installation Mode") + "\n\n")

	opt1 := m.Styles.Accent.Render("1. Client Setup") + m.Styles.Subtle.Render(" (User-local path)")
	opt2 := m.Styles.Accent.Render("2. Server Setup") + m.Styles.Subtle.Render(" (System-wide path, Linux only)")

	b.WriteString(opt1 + "\n")
	b.WriteString(opt2 + "\n\n")

	b.WriteString(m.Styles.Subtle.Render("Press [1] or [2] to continue..."))

	return b.String()
}

func (m Model) viewError() string {
	var b strings.Builder
	b.WriteString(m.Styles.Error.Render("ERROR") + "\n\n")
	b.WriteString(m.Styles.Normal.Render(m.Err.Error()) + "\n\n")

	if m.ShowFullLog {
		b.WriteString(m.Styles.Subtle.Render("Installation Log Audit (Arrows/PgUp/PgDn to scroll, 'l' to return)") + "\n")
		b.WriteString(m.Viewport.View() + "\n")
	} else if len(m.LogMessages) > 0 {
		// Show last 3 logs for context
		start := len(m.LogMessages) - 3
		if start < 0 {
			start = 0
		}
		for _, log := range m.LogMessages[start:] {
			b.WriteString(m.Styles.Subtle.Render(log) + "\n")
		}
		b.WriteString("\n" + m.Styles.Subtle.Render("Press 'l' for full audit, [Backspace] to go back, [q] to quit."))
	} else {
		b.WriteString(m.Styles.Subtle.Render("Press [Backspace] to go back, [q] to quit."))
	}

	return b.String()
}

func (m Model) viewWelcome() string {
	dInfoStr := "Unknown Distro"
	if m.DistroInfo != nil {
		dInfoStr = fmt.Sprintf("%s %s", m.DistroInfo.ID, m.DistroInfo.Version)
	}

	statusLine := m.Styles.Subtle.Render("Detected System: " + dInfoStr)
	if runtime.GOOS == "windows" {
		adminStatus := m.Styles.Success.Render(" (Administrator)")
		if !m.IsAdmin {
			adminStatus = m.Styles.Warning.Render(" (Non-Admin)")
		}
		statusLine += adminStatus
	}

	// Simple, elegant banner without a box
	bannerStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(TerminalTheme().Primary)).
		PaddingLeft(2).
		MarginBottom(1)

	banner := bannerStyle.Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color(TerminalTheme().Accent)).Bold(true).Render("CSW") + "\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color(TerminalTheme().Subtle)).Faint(true).Render(fmt.Sprintf("Cobalt Strike Wizard (%s)", m.App.Version)),
	)

	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		banner,
		statusLine,
		m.Styles.Subtle.Render("Press [Enter] to start setup..."),
	)
}

func (m Model) viewForm() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title.Render("Configuration Details") + "\n\n")

	numInputs := m.getNumInputs()
	for i := 0; i < numInputs; i++ {
		b.WriteString(m.Inputs[i].View())
		if i == 2 && m.FocusIndex == 2 && len(m.LocalIPs) > 0 {
			b.WriteString(m.Styles.Subtle.Render("  ← Use Arrows to Cycle Local IPs →"))
		}
		if i == 4 && m.FocusIndex == 4 && len(m.ProfileNames) > 0 {
			b.WriteString(m.Styles.Subtle.Render("  ← Use Arrows to Cycle Profiles →"))
		}
		if i < numInputs-1 {
			b.WriteRune('\n')
		}
	}

	var button string
	if m.FocusIndex == numInputs {
		button = m.Styles.SelectedOption.Render("\n\n[ Submit ]")
	} else {
		button = m.Styles.Subtle.Render("\n\n[ Submit (Enter) ]")
	}
	b.WriteString(button)

	return b.String()
}

func (m Model) viewInstalling() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title.Render("Installing Cobalt Strike") + "\n\n")

	if m.ShowFullLog {
		b.WriteString(m.Styles.Subtle.Render("Full Installation Log (Arrows/PgUp/PgDn to scroll, 'l' to return)") + "\n")
		b.WriteString(m.Viewport.View() + "\n")
	} else {
		b.WriteString(m.Progress.View() + "\n\n")

		// Show last 5 logs for context
		start := len(m.LogMessages) - 5
		if start < 0 {
			start = 0
		}
		for _, log := range m.LogMessages[start:] {
			b.WriteString(m.Styles.Subtle.Render(log) + "\n")
		}
		b.WriteString("\n" + m.Styles.Subtle.Render("Press 'l' for full log scroll view"))
	}

	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(m.Styles.Success.Render("Installation Complete!") + "\n\n")

	if m.ShowFullLog {
		b.WriteString(m.Styles.Subtle.Render("Full Installation Log (Arrows/PgUp/PgDn to scroll, 'l' to return)") + "\n")
		b.WriteString(m.Viewport.View() + "\n")
	} else {
		b.WriteString(m.Styles.Normal.Render("Cobalt Strike is ready at: "+m.App.Config.InstallPath) + "\n\n")

		// Show last 5 logs for context
		start := len(m.LogMessages) - 5
		if start < 0 {
			start = 0
		}
		for _, log := range m.LogMessages[start:] {
			b.WriteString(m.Styles.Subtle.Render(log) + "\n")
		}
		b.WriteString("\n" + m.Styles.Subtle.Render("Press 'l' for full log scroll view, 'q' to quit"))
	}

	return b.String()
}

func (m Model) viewConfirmOverwrite() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title.Render("Directory Occupied") + "\n\n")

	msg := fmt.Sprintf("The directory %s is not empty.\nExisting files may be overwritten or deleted.", m.App.Config.InstallPath)
	b.WriteString(m.Styles.Normal.Render(msg) + "\n\n")

	b.WriteString(m.Styles.Accent.Render("Do you want to clear this directory and proceed? [y/N]") + "\n\n")
	b.WriteString(m.Styles.Subtle.Render("Press 'y' to confirm, 'n' or [Esc] to go back."))
	return b.String()
}

// startInstall initiates the installation process in a separate goroutine.
func (m Model) startInstall() tea.Cmd {
	ch := make(chan installProgressMsg)
	return tea.Batch(
		func() tea.Msg {
			go func() {
				err := m.App.Install(func(percent float64, log string) {
					ch <- installProgressMsg{Percent: percent, Log: log, Channel: ch}
				})
				if err != nil {
					ch <- installProgressMsg{Err: err, Channel: ch}
				}
				close(ch)
			}()
			return nil
		},
		waitForProgress(ch),
	)
}

// waitForProgress is a command that waits for a message on the progress channel.
func waitForProgress(ch chan installProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return installDoneMsg{}
		}
		if msg.Err != nil {
			return errMsg{msg.Err}
		}
		return msg
	}
}

type installProgressMsg struct {
	Percent float64
	Log     string
	Err     error
	Channel chan installProgressMsg
}

type installDoneMsg struct{}
type errMsg struct{ err error }
