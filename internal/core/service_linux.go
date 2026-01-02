//go:build linux

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// setupSystemdService configures and starts a systemd service for the TeamServer.
// It detects if systemd is available and handles nested binary paths common in modern CS packages.
func (a *App) setupSystemdService(installPath string, onProgress func(float64, string)) error {
	// Step 1: Check if systemd is even available
	if _, err := exec.LookPath("systemctl"); err != nil {
		onProgress(-1, "systemctl not found, skipping systemd service setup")
		return nil
	}

	// Step 2: Detect teamserver binary location (root vs nested server/ dir)
	binaryPath := filepath.Join(installPath, "teamserver")
	workingDir := installPath

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		serverBinaryPath := filepath.Join(installPath, "server", "teamserver")
		if _, err := os.Stat(serverBinaryPath); err == nil {
			binaryPath = serverBinaryPath
			workingDir = filepath.Join(installPath, "server")
		}
	}

	// Step 3: Generate service unit content
	serviceContent := fmt.Sprintf(`[Unit]
Description=Cobalt Strike TeamServer
After=network.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s %s %s %s/profiles/%s
Restart=always

[Install]
WantedBy=multi-user.target
`, workingDir, binaryPath, a.Config.ServerIP, a.Config.ServerPassword, installPath, a.Config.SelectedProfile)

	servicePath := "/etc/systemd/system/cobaltstrike.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	// Step 4: Reload systemd and lifecycle the service
	if _, err := a.RunCommand(onProgress, "", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload failed: %v", err)
	}
	if _, err := a.RunCommand(onProgress, "", "systemctl", "enable", "cobaltstrike"); err != nil {
		return fmt.Errorf("systemctl enable failed: %v", err)
	}
	if _, err := a.RunCommand(onProgress, "", "systemctl", "restart", "cobaltstrike"); err != nil {
		return fmt.Errorf("systemctl restart failed: %v", err)
	}

	return nil
}
