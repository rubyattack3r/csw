package distro

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// DistroType represents the category of the operating system distribution
// (e.g., Debian, RedHat, Windows).
type DistroType int

const (
	Unknown DistroType = iota
	Debian
	RedHat
	Arch
	Windows
	MacOS
)

// Info contains metadata about the detected operating system, including its
// ID, version string, and normalized distro type.
type Info struct {
	ID      string
	Version string
	Type    DistroType
}

// Detect identifies the current operating system distribution and version.
// It handles Windows, macOS, and major Linux distributions by inspecting
// /etc/os-release or using runtime information.
func Detect() (*Info, error) {
	if runtime.GOOS == "windows" {
		return &Info{
			ID:      "windows",
			Version: "unknown",
			Type:    Windows,
		}, nil
	}

	if runtime.GOOS == "darwin" {
		return &Info{
			ID:      "macos",
			Version: "unknown",
			Type:    MacOS,
		}, nil
	}

	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info := &Info{Type: Unknown}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], "\"")

		switch key {
		case "ID":
			info.ID = value
			switch value {
			case "debian", "ubuntu", "kali":
				info.Type = Debian
			case "fedora", "centos", "rhel":
				info.Type = RedHat
			case "arch", "manjaro":
				info.Type = Arch
			}
		case "VERSION_ID":
			info.Version = value
		}
	}

	return info, nil
}
