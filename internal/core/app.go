package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type InstallMode int

const (
	ModeClient InstallMode = iota
	ModeServer
)

// Config holds the application's configuration state, including license keys,
// installation paths, and the targeted installation mode.
type Config struct {
	LicenseKey      string
	InstallPath     string
	Mode            InstallMode
	ServerIP        string
	ServerPassword  string
	SelectedProfile string
}

// App serves as the central orchestration engine, holding configuration and
// providing methods for the installation lifecycle.
type App struct {
	Config  *Config
	Version string
}

// NewApp initializes a new App instance with default configuration settings.
func NewApp() *App {
	return &App{
		Config: &Config{
			Mode: ModeClient,
		},
	}
}

// SetMode configures the installation mode (Client or Server) and initializes
// default installation paths based on the current operating system.
func (a *App) SetMode(mode InstallMode) error {
	if mode == ModeServer && runtime.GOOS != "linux" {
		return fmt.Errorf("server mode is only supported on Linux")
	}

	a.Config.Mode = mode

	// Set default path
	home := GetRealHome()

	switch mode {
	case ModeClient:
		switch runtime.GOOS {
		case "windows":
			a.Config.InstallPath = filepath.Join(os.Getenv("APPDATA"), "CobaltStrike")
		case "darwin":
			a.Config.InstallPath = filepath.Join(home, "Applications", "CobaltStrike")
		default:
			a.Config.InstallPath = filepath.Join(home, ".local", "share", "cobaltstrike")
		}
	case ModeServer:
		a.Config.InstallPath = "/opt/cobaltstrike"
	}

	return nil
}
