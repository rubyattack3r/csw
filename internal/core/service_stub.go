//go:build !linux

package core

// setupSystemdService is a no-op for non-Linux platforms as systemd is Linux-specific.
func (a *App) setupSystemdService(installPath string, onProgress func(float64, string)) error {
	onProgress(-1, "Skipping systemd setup (non-Linux platform)")
	return nil
}
