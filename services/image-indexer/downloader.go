package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

// validateURL checks if the URL is safe to use (prevents SSRF attacks)
func validateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Only allow HTTP and HTTPS protocols
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported protocol: %s", parsedURL.Scheme)
	}

	// Block private/internal IP ranges and localhost
	host := strings.ToLower(parsedURL.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("localhost access not allowed")
	}

	// Block private IP ranges (simplified check)
	if strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.") ||
		strings.HasPrefix(host, "169.254.") {
		return fmt.Errorf("private IP range access not allowed")
	}

	return nil
}

// DownloadFile downloads a file from a given url and returns a file reader and its mime type
func DownloadFile(url string) (io.Reader, string, int, error) {
	// Validate URL to prevent SSRF attacks
	if err := validateURL(url); err != nil {
		return nil, "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := http.Get(url) // #nosec G107 -- URL is validated above
	if err != nil {
		return nil, "", 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", 0, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	fileHeader := make([]byte, 512)
	if n, err := resp.Body.Read(fileHeader); err != nil {
		if errors.Is(err, io.EOF) {
			// the file size is smaller than the sample bytes (512)
			fileHeader = fileHeader[:n]
		} else {
			return nil, "", 0, fmt.Errorf("%d bytes read. error: %s", n, err.Error())
		}
	}
	mimeType := mimetype.Detect(fileHeader).String()

	file := bytes.NewBuffer(fileHeader)
	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, "", 0, err
	}

	log.Debug("file downloaded",
		zap.String("download_url", url),
		zap.String("mimeType", mimeType),
		zap.Int("file_size", file.Len()))

	return file, mimeType, file.Len(), nil
}
