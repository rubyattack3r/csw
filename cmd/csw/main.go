package main

import (
	"fmt"
	"os"

	"csw/internal/core"
	"csw/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// Version is the application version, injected at build time via ldflags.
var Version = "dev"

func main() {
	app := core.NewApp()
	app.Version = Version

	if !app.IsAdmin() {
		fmt.Println("ERROR: Administrative privileges required.")
		fmt.Println("Please restart the application as an Administrator (Windows) or as root (Linux).")
		fmt.Println("\nPress Enter to exit...")
		var input string
		_, _ = fmt.Scanln(&input)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewModel(app))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting CSW: %v\n", err)
		os.Exit(1)
	}
}
