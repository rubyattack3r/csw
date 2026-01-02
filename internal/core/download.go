package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// requestToken authenticates with the Cobalt Strike server using the provided
// license key and extracts a temporary download token from the response.
func (a *App) requestToken() (string, error) {
	data := url.Values{}
	data.Set("dlkey", a.Config.LicenseKey)

	// Create request with curl User-Agent
	req, err := http.NewRequest("POST", "https://download.cobaltstrike.com/download", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "curl/7.81.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content := string(body)
	if strings.Contains(content, "href=\"/downloads/") {
		start := strings.Index(content, "/downloads/") + len("/downloads/")
		end := strings.Index(content[start:], "/")
		if end != -1 {
			return content[start : start+end], nil
		}
	}

	return "", fmt.Errorf("failed to extract download token from response")
}

// downloadFile fetches the Cobalt Strike artifact from the official servers
// and streams it to a temporary file while reporting progress.
func (a *App) downloadFile(url, filepath string, onProgress func(float64, string)) error {
	// Create request with curl User-Agent for consistency
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "curl/7.81.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// Wrap reader to track progress
	counter := &WriteCounter{
		Total:      uint64(resp.ContentLength),
		onProgress: onProgress,
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	return err
}

// WriteCounter tracks the number of bytes written to a file and triggers
// progress updates for the UI.
type WriteCounter struct {
	Total      uint64
	Downloaded uint64
	onProgress func(float64, string)
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += uint64(n)
	if wc.Total > 0 {
		percent := float64(wc.Downloaded) / float64(wc.Total)
		wc.onProgress(0.60+(percent*0.1), fmt.Sprintf("Downloading... %.2f%%", percent*100))
	}
	return n, nil
}
