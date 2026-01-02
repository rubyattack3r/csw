package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"csw/internal/distro"
)

// ExtractArtifact provides a unified interface for extracting Cobalt Strike
// distributions (Zip, Tgz, or DMG) based on the target distribution's metadata.
func (a *App) ExtractArtifact(src, dest string, dInfo distro.Info, onProgress func(float64, string)) error {
	extractType := distro.GetExtractType(dInfo.Type)

	switch extractType {
	case distro.ExtractZip:
		return a.extractZip(src, dest)
	case distro.ExtractDmg:
		onProgress(0.85, "Mounting DMG...")
		mntPoint := filepath.Join(os.TempDir(), "cs-mnt")
		_ = os.MkdirAll(mntPoint, 0755)

		onProgress(-1, fmt.Sprintf("Executing: hdiutil attach %s ...", src))
		attachCmd := exec.Command("hdiutil", "attach", src, "-mountpoint", mntPoint, "-nobrowse", "-quiet")
		if out, err := attachCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("hdiutil attach failed: %v. Output: %s", err, string(out))
		}

		// Ensure we unmount at the end
		defer func() {
			onProgress(-1, fmt.Sprintf("Executing: hdiutil detach %s", mntPoint))
			_ = exec.Command("hdiutil", "detach", mntPoint, "-quiet").Run()
		}()

		onProgress(0.87, "Copying files from DMG...")

		// Use 'cp -R' to preserve permissions and hierarchy efficiently
		// We copy everything from the mount point to the destination
		onProgress(-1, fmt.Sprintf("Executing: cp -R %s/ %s", mntPoint, dest))
		cpCmd := exec.Command("cp", "-R", mntPoint+"/", dest)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to copy files from DMG: %v. Output: %s", err, string(out))
		}

		onProgress(0.89, "Unmounting DMG...")
		return nil
	default: // ExtractTgz
		return a.extractTgz(src, dest)
	}
}

// extractTgz handles the extraction of Gzip-compressed tarballs.
func (a *App) extractTgz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the top-level directory (Cobalt Strike tgz usually has a root folder)
		parts := strings.Split(header.Name, "/")
		if len(parts) <= 1 {
			continue // Skip the root directory itself or single files at root
		}

		relPath := filepath.Join(parts[1:]...)
		target := filepath.Join(dest, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }() // Ensure file is closed
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractZip handles the extraction of standard ZIP archives.
func (a *App) extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		// Strip top-level directory if present
		parts := strings.Split(f.Name, "/")
		if len(parts) <= 1 {
			continue
		}

		relPath := filepath.Join(parts[1:]...)
		target := filepath.Join(dest, relPath)

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			_ = srcFile.Close()
			_ = dstFile.Close()
			return err
		}

		_ = srcFile.Close()
		_ = dstFile.Close()
	}
	return nil
}
