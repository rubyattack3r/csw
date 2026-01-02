package core

import (
	"fmt"
	"os/exec"
	"strings"

	"csw/internal/distro"
)

// installDependencies handles the system-level installation of required packages.
// It is idempotent and gracefully handles cases where packages are already installed.
func (a *App) installDependencies(onProgress func(float64, string), missingDeps []string, dInfo distro.Info) error {
	cmdBase, args := distro.GetDependencyCommand(dInfo.Type, missingDeps)

	cmdStr := fmt.Sprintf("%s %s", cmdBase, strings.Join(args, " "))
	onProgress(-1, fmt.Sprintf("Executing: %s", cmdStr))

	out, err := a.RunCommand(onProgress, "", cmdBase, args...)
	if err != nil {
		// Winget often returns non-zero codes for "already installed" or "no upgrade found".
		// We treat these as a success for our "ensuring" step.
		outStr := strings.ToLower(string(out))
		if strings.Contains(outStr, "no available upgrade found") ||
			strings.Contains(outStr, "already installed") ||
			strings.Contains(outStr, "newer package versions are available") {
			onProgress(0.22, "System dependencies are already satisfied.")
			return nil
		}
		return fmt.Errorf("dependency install failed: %v. Output: %s", err, string(out))
	}
	return nil
}

// verifyJava17 performs a pre-flight check to ensure a compatible Java runtime
// (version 17 or higher) is available in the system PATH.
func (a *App) verifyJava17() error {
	cmd := exec.Command("java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("java not found")
	}

	outStr := string(output)
	if !strings.Contains(outStr, "version \"17") && !strings.Contains(outStr, "version \"18") &&
		!strings.Contains(outStr, "version \"19") && !strings.Contains(outStr, "version \"20") &&
		!strings.Contains(outStr, "version \"21") {
		return fmt.Errorf("java 17 or higher is required")
	}
	return nil
}
