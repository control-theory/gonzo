package version

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	UpdateAvailable bool
	LatestVersion   string
	CurrentVersion  string
	ReleaseURL      string
	Severity        string
}

// VersionResponse represents the API response from the version check endpoint
type VersionResponse struct {
	CurrentVersion  string `json:"current_version"`
	Latest          Latest `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	Severity        string `json:"severity"`
	CacheLastUpdate string `json:"cache_last_updated"`
}

// Latest contains information about the latest release
type Latest struct {
	Tag         string            `json:"tag"`
	PublishedAt string            `json:"published_at"`
	URL         string            `json:"url"`
	Notes       string            `json:"notes"`
	Assets      map[string]string `json:"assets"`
}

// Checker handles version checking in the background
type Checker struct {
	currentVersion string
	commit         string
	updateInfo     *UpdateInfo
	checkComplete  chan bool
}

// NewChecker creates a new version checker
func NewChecker(currentVersion, commit string) *Checker {
	return &Checker{
		currentVersion: currentVersion,
		commit:         commit,
		checkComplete:  make(chan bool, 1),
	}
}

// CheckInBackground starts a background goroutine to check for updates
func (c *Checker) CheckInBackground() {
	go func() {
		defer func() {
			// Ensure we always signal completion, even if panic occurs
			select {
			case c.checkComplete <- true:
			default:
			}
		}()

		// Don't check for updates in development builds
		if c.currentVersion == "dev" || c.currentVersion == "" {
			return
		}

		// Create context with timeout for the HTTP request
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get anonymous ID for caching uniqueness
		anonID := getOrCreateAnonID()

		// Determine channel (stable for tagged releases, edge for local builds)
		channel := "stable"
		if strings.Contains(c.currentVersion, "-dirty") || strings.Contains(c.currentVersion, "-g") {
			channel = "edge"
		}

		// Build the API URL with parameters
		url := fmt.Sprintf(
			"https://gonzo-version.controltheory.com/v1/check?app=gonzo&version=%s&platform=%s&arch=%s&commit=%s&channel=%s&anon_id=%s",
			c.currentVersion,
			runtime.GOOS,
			runtime.GOARCH,
			c.commit,
			channel,
			anonID,
		)

		// Create HTTP request with context
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return // Silently ignore errors
		}

		// Make the HTTP request
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return // Silently ignore errors
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return // Silently ignore errors
		}

		// Parse JSON response
		var versionResp VersionResponse
		if err := json.Unmarshal(body, &versionResp); err != nil {
			return // Silently ignore errors
		}

		// Store update information
		c.updateInfo = &UpdateInfo{
			UpdateAvailable: versionResp.UpdateAvailable,
			LatestVersion:   strings.TrimPrefix(versionResp.Latest.Tag, "v"),
			CurrentVersion:  c.currentVersion,
			ReleaseURL:      versionResp.Latest.URL,
			Severity:        versionResp.Severity,
		}
	}()
}

// GetUpdateInfo returns the update information if available
// It waits up to 100ms for the check to complete, then returns whatever is available
func (c *Checker) GetUpdateInfo() *UpdateInfo {
	select {
	case <-c.checkComplete:
		// Check completed, return result
		return c.updateInfo
	case <-time.After(100 * time.Millisecond):
		// Don't wait too long, return whatever we have
		return c.updateInfo
	}
}

// GetUpdateInfoNonBlocking returns the update information without waiting
func (c *Checker) GetUpdateInfoNonBlocking() *UpdateInfo {
	return c.updateInfo
}

// getOrCreateAnonID creates a consistent anonymous ID based on machine characteristics
func getOrCreateAnonID() string {
	// Create a machine-specific but anonymous identifier
	// This will be the same for each machine but doesn't identify the user

	hostname, _ := os.Hostname()
	macAddr := getMACAddress()
	osArch := runtime.GOOS + "-" + runtime.GOARCH

	// Combine machine characteristics
	machineInfo := hostname + "|" + macAddr + "|" + osArch

	// Hash to create anonymous but consistent ID
	h := sha256.Sum256([]byte(machineInfo))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for shorter ID
}

// getMACAddress attempts to get a MAC address from network interfaces
func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			if iface.HardwareAddr != nil {
				return iface.HardwareAddr.String()
			}
		}
	}

	return "unknown"
}
