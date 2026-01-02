package distro

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetDependencies returns the list of system package names required by
// Cobalt Strike for the given distribution type.
func GetDependencies(dType DistroType) []string {
	switch dType {
	case Debian:
		return []string{"build-essential", "openjdk-17-jdk"}
	case RedHat:
		return []string{"gcc", "java-17-openjdk"}
	case Arch:
		return []string{"base-devel", "jdk17-openjdk"}
	case Windows:
		return []string{"Microsoft.OpenJDK.21"}
	case MacOS:
		return []string{"openjdk"}
	default:
		return []string{}
	}
}

// GetInstallCommand returns the base package manager command for installing
// software on the given distribution.
func GetInstallCommand(dType DistroType) string {
	switch dType {
	case Debian:
		return "apt-get install -y"
	case RedHat:
		return "dnf install -y"
	case Arch:
		return "pacman -S --noconfirm"
	case Windows:
		return "winget install"
	case MacOS:
		return "brew install"
	default:
		return "echo 'Manual install required for:'"
	}
}

// GetArtifactName returns the filename of the Cobalt Strike distribution
// archive (Zip, Tgz, DMG) corresponding to the platform.
func GetArtifactName(dType DistroType) string {
	switch dType {
	case Windows:
		return "cobaltstrike-dist-windows.zip"
	case MacOS:
		return "cobaltstrike-dist-mac.dmg"
	default:
		return "cobaltstrike-dist-linux.tgz"
	}
}

// GetUpdateCmd returns the platform-specific command to execute the
// Cobalt Strike license update script.
func GetUpdateCmd(dType DistroType) string {
	switch dType {
	case Windows:
		return ".\\update.bat"
	default:
		return "./update"
	}
}

type ExtractType int

const (
	ExtractTgz ExtractType = iota
	ExtractZip
	ExtractDmg
)

// GetExtractType determines the required extraction method (Zip, Tgz, DMG)
// for the given distribution's artifact.
func GetExtractType(dType DistroType) ExtractType {
	switch dType {
	case Windows:
		return ExtractZip
	case MacOS:
		return ExtractDmg
	default:
		return ExtractTgz
	}
}

// GetCheckFile returns the name of a sentinel file used to verify if
// Cobalt Strike is already installed in a directory.
func GetCheckFile(dType DistroType) string {
	if dType == Windows {
		return "cobaltstrike.exe"
	}
	return "cobaltstrike.auth"
}

// GetDependencyCommand constructs the full, platform-specific command line
// (executable + arguments) to install the provided list of missing dependencies.
func GetDependencyCommand(dType DistroType, missingDeps []string) (string, []string) {
	instCmd := GetInstallCommand(dType)
	parts := strings.Fields(instCmd)
	cmdBase := parts[0]
	args := parts[1:]

	if dType == Windows {
		// For Windows, we use winget directly. It handles multiple packages in one call.
		return "winget", append([]string{"install", "-e", "-h", "--accept-package-agreements", "--accept-source-agreements"}, missingDeps...)
	}

	return cmdBase, append(args, missingDeps...)
}

// IsAdmin reports whether the current process is running with
// administrative/root privileges on the host system.
func (dType DistroType) IsAdmin() bool {
	if runtime.GOOS == "windows" {
		// Simplified check for Windows admin - checking if we can write to system dir
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return os.Geteuid() == 0
}

// IsUpdateScript returns true if the given command name refers to a
// Cobalt Strike license update script.
func IsUpdateScript(name string) bool {
	base := filepath.Base(name)
	return base == "update" || base == "update.bat"
}
