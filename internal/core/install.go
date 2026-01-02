package core

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"csw/internal/distro"
)

// Install orchestrates the complete installation flow.
// It handles distro detection, pre-flight checks, dependency installation,
// downloading, extraction, and license application.
func (a *App) Install(onProgress func(float64, string)) error {
	onProgress(0.05, "Detecting system distribution...")
	dInfo, err := distro.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect distro: %w", err)
	}

	expandedPath := ExpandPath(a.Config.InstallPath)

	// Pre-flight validation
	onProgress(0.06, "Validating system privileges...")
	if a.Config.Mode == ModeServer && !a.IsAdmin() {
		return fmt.Errorf("root privileges are required for Server Mode setup (installing to /opt)")
	}
	onProgress(0.08, "Checking Java version (Requires 17+)...")
	if err := a.verifyJava17(); err != nil {
		onProgress(0.08, fmt.Sprintf("Java check warning: %v", err))
	}

	// System dependencies
	onProgress(0.12, "Ensuring system dependencies are satisfied...")
	deps := distro.GetDependencies(dInfo.Type)

	if len(deps) > 0 {
		if err := a.installDependencies(onProgress, deps, *dInfo); err != nil {
			return err
		}
	} else {
		onProgress(0.20, "No additional system dependencies required.")
	}

	// Cobalt Strike Artifact Acquisition
	onProgress(0.40, "Requesting download token from Cobalt Strike server...")
	token, err := a.requestToken()
	if err != nil {
		onProgress(0.45, fmt.Sprintf("Warning: Real token request failed: %v. Trying simulation...", err))
		token = "SIMULATED_TOKEN"
	} else {
		onProgress(0.50, "Authentication sequence complete.")
	}

	artifactName := distro.GetArtifactName(dInfo.Type)
	onProgress(0.60, fmt.Sprintf("Downloading %s...", artifactName))

	tempFile := filepath.Join(os.TempDir(), artifactName)

	downloadURL := fmt.Sprintf("https://download.cobaltstrike.com/downloads/%s/latest410/%s", token, artifactName)
	if err := a.downloadFile(downloadURL, tempFile, onProgress); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Destination Preparation
	onProgress(0.70, fmt.Sprintf("Creating directory: %s", expandedPath))
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("failed to create install path: %w", err)
	}

	// Payload Extraction
	onProgress(0.80, fmt.Sprintf("Extracting %s to %s...", artifactName, expandedPath))
	if err := a.ExtractArtifact(tempFile, expandedPath, *dInfo, onProgress); err != nil {
		return fmt.Errorf("extraction failed: %v", err)
	}

	onProgress(0.87, "Extraction complete.")

	// License Initialization
	onProgress(0.90, "Applying license key via update script...")

	updateCmd := distro.GetUpdateCmd(dInfo.Type)

	onProgress(-1, fmt.Sprintf("Executing: %s [LICENSE_KEY]", updateCmd))
	out, err := a.RunCommand(onProgress, expandedPath, updateCmd, a.Config.LicenseKey)
	if err != nil {
		return fmt.Errorf("update script failed: %v. Output: %s", err, string(out))
	}

	onProgress(0.98, "License applied successfully via update script.")

	// Step 6: Server-specific post-install persistence
	if a.Config.Mode == ModeServer {
		onProgress(0.99, "Deploying full Malleable C2 profile library...")
		if err := a.deployProfiles(expandedPath); err != nil {
			onProgress(0.99, fmt.Sprintf("Warning: Failed to deploy profiles: %v", err))
		}

		onProgress(0.99, "Configuring systemd service...")
		if err := a.setupSystemdService(expandedPath, onProgress); err != nil {
			return fmt.Errorf("systemd setup failed: %v", err)
		}
	}

	onProgress(1.00, "Setup complete!")
	return nil
}

// RunCommand executes a system command and streams its output (stdout/stderr)
// back to the progress delegate. It handles specific stdin requirements for
// interactive update scripts.
func (a *App) RunCommand(onProgress func(float64, string), cwd string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)

	if cwd != "" {
		cmd.Dir = cwd
	}

	// Use distro helper to identify update script for special Stdin handling
	if distro.IsUpdateScript(name) {
		if len(args) > 0 {
			cmd.Stdin = strings.NewReader(args[0] + "\n")
			cmd.Args = []string{name} // Remove args from command line as they go to stdin
		}
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var combinedOut strings.Builder

	// Stream output
	streamer := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			onProgress(-1, line) // -1 indicates "don't update percentage, just log"
			combinedOut.WriteString(line + "\n")
		}
	}

	go streamer(stdout)
	go streamer(stderr)

	err := cmd.Wait()
	return []byte(combinedOut.String()), err
}

// IsInstalled checks if Cobalt Strike is already present at the given path
// by looking for platform-specific sentinel files.
func (a *App) IsInstalled(path string, dType distro.DistroType) bool {
	checkFile := distro.GetCheckFile(dType)
	_, err := os.Stat(filepath.Join(path, checkFile))
	return err == nil
}

// IsDirEmpty returns true if the directory at the given path contains no files.
func (a *App) IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// CleanDirectory removes all contents within the specified directory path.
func (a *App) CleanDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = dir.Close() }()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(path, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// IsAdmin checks if the current process has administrative or root privileges.
func (a *App) IsAdmin() bool {
	dInfo, _ := distro.Detect()
	return dInfo.Type.IsAdmin()
}

// GetRealHome returns the home directory of the actual user, even if running under sudo.
func GetRealHome() string {
	if runtime.GOOS != "windows" {
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			// On Linux/Unix, if run via sudo, return the home of the original user.
			if home := os.Getenv("HOME"); home != "" && home != "/root" {
				return home
			}
			// Fallback: many systems have /home/username
			userHome := filepath.Join("/home", sudoUser)
			if _, err := os.Stat(userHome); err == nil {
				return userHome
			}
			// macOS fallback
			macHome := filepath.Join("/Users", sudoUser)
			if _, err := os.Stat(macHome); err == nil {
				return macHome
			}
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

// ExpandPath resolves environment variables and home directory aliases (~/)
// in a path string. It is sudo-aware and resolves to the human user's home.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(GetRealHome(), path[2:])
	}
	return filepath.Clean(os.ExpandEnv(path))
}

// deployProfiles writes all embedded Malleable C2 profiles into a
// "profiles" subdirectory within the installation directory.
func (a *App) deployProfiles(installPath string) error {
	profilesDir := filepath.Join(installPath, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %v", err)
	}

	names := GetEmbeddedProfileNames()
	for _, name := range names {
		content, err := GetProfileContent(name)
		if err != nil {
			return err
		}
		dest := filepath.Join(profilesDir, name)
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("failed to write profile %s: %v", name, err)
		}
	}
	return nil
}

// GetLocalIPs returns a list of non-loopback IPv4 addresses found
// on the local network interfaces.
func GetLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}
