// sysreboot is an enhanced reboot tool with smart capabilities.
// Author: Eric Sobczak
// License: MIT
// Repository: https://github.com/esobczak1970/sysreboot
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Constants for application metadata
const (
	appName    = "sysreboot"
	appVersion = "0.2.0"
)

// Enumeration for index mapping of the flags (must match appFlags order)
const (
	cancelIndex = iota
	confirmIndex
	confirmTimeoutIndex
	delayIndex
	haltIndex
	messageIndex
	poweroffIndex
	rebootIndex
	shutdownIndex
	statusIndex
	timeIndex
	verboseIndex
	versionIndex
)

// flagData defines the structure for command-line flag information.
type flagData struct {
	longName   string      // Long form of the flag.
	shortName  string      // Short form of the flag (single letter).
	value      interface{} // Variable that stores the flag's value.
	defaultVal interface{} // Default value of the flag.
	usage      string      // Description of the flag.
}

// appFlags holds the configuration for all command-line flags.
// IMPORTANT: Order must match the index constants above
var appFlags = []flagData{
	{"cancel", "x", new(bool), false, "Cancel a pending scheduled action."},
	{"confirm", "c", new(bool), false, "Require confirmation before performing the action."},
	{"confirm-timeout", "ct", new(int), 10, "Confirmation timeout in seconds."},
	{"delay", "d", new(int), 0, "Delay in minutes before performing the action."},
	{"halt", "h", new(bool), false, "Halt the machine."},
	{"message", "m", new(string), "", "Message to send to all users before performing the action."},
	{"poweroff", "p", new(bool), false, "Power-off the machine."},
	{"reboot", "r", new(bool), true, "Reboot the machine (default action)."},
	{"shutdown", "s", new(bool), false, "Shutdown the machine (alias for poweroff)."},
	{"status", "st", new(bool), false, "Show status of pending scheduled actions."},
	{"time", "t", new(string), "", "Specific time for the action in HH:MM format (24-hour)."},
	{"verbose", "vb", new(bool), false, "Output more information."},
	{"version", "v", new(bool), false, "Show application version."},
}

var (
	logFile   string      // Path to the log file.
	logger    *log.Logger // Logger instance for the application.
	logWriter io.WriteCloser
)

// Constants for PID file management
const (
	pidFileDir    = "/var/run" // Primary location for PID files (Unix)
	pidFileDirAlt = "/tmp"     // Fallback location for PID files
	pidFileName   = "sysreboot.pid"
)

// ScheduleInfo holds information about a scheduled action
type ScheduleInfo struct {
	PID           int       `json:"pid"`
	Action        string    `json:"action"`
	ScheduledTime string    `json:"scheduled_time,omitempty"`
	Delay         int       `json:"delay_minutes,omitempty"`
	Message       string    `json:"message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func init() {
	// Initialize command-line flags based on appFlags configuration.
	for _, fd := range appFlags {
		switch v := fd.value.(type) {
		case *bool:
			flag.BoolVar(v, fd.longName, fd.defaultVal.(bool), fd.usage)
			flag.BoolVar(v, fd.shortName, fd.defaultVal.(bool), fd.usage+" (short form)")
		case *int:
			flag.IntVar(v, fd.longName, fd.defaultVal.(int), fd.usage)
			flag.IntVar(v, fd.shortName, fd.defaultVal.(int), fd.usage+" (short form)")
		case *string:
			flag.StringVar(v, fd.longName, fd.defaultVal.(string), fd.usage)
			flag.StringVar(v, fd.shortName, fd.defaultVal.(string), fd.usage+" (short form)")
		}
	}

	// Set up the log file location and initialize the logger.
	logFile = filepath.Join(getLogFileDirectory(), appName+".log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	logWriter = file
	logger = log.New(file, appName+": ", log.Ldate|log.Ltime|log.Lshortfile)

	// Override the default flag usage message with a custom one.
	flag.Usage = customUsage
}

func getLogFileDirectory() string {
	// Get the appropriate log file directory based on the operating system.
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = "."
		}
		return appData
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return homeDir
}

func customUsage() {
	// Display custom usage information for the application.
	fmt.Fprintf(os.Stderr, "%s: Enhanced reboot tool with smart capabilities.\n\n", appName)
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", appName)
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s --reboot --delay 5 --message \"System will reboot in 5 minutes!\"\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --poweroff --confirm\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --shutdown --confirm\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --halt --verbose\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --reboot --time \"23:30\"\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --status                    # Show pending scheduled actions\n", appName)
	fmt.Fprintf(os.Stderr, "  %s --cancel                    # Cancel pending scheduled action\n", appName)
}

func scheduleAtSpecificTime(timeStr string, action string, message string, confirmation bool) error {
	// Schedule an action (reboot, shutdown, etc.) to occur at a specific time.
	rebootTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return fmt.Errorf("invalid time format (use HH:MM): %v", err)
	}

	// Calculate how long to wait until the specified time.
	now := time.Now()
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), rebootTime.Hour(), rebootTime.Minute(), 0, 0, now.Location())
	durationUntilReboot := targetTime.Sub(now)

	if durationUntilReboot < 0 {
		durationUntilReboot += 24 * time.Hour // Schedule for the next day if time is in the past.
	}

	logger.Printf("%s scheduled at %s (in %s).\n", action, rebootTime.Format("15:04"), durationUntilReboot)
	fmt.Printf("%s scheduled at %s (in %s).\n", action, rebootTime.Format("15:04"), durationUntilReboot)
	fmt.Println("Press Ctrl+C to cancel, or use 'sysreboot --cancel' from another terminal.")

	// Set up signal handling for cancellation
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for either the scheduled time or a signal
	timer := time.NewTimer(durationUntilReboot)
	select {
	case <-timer.C:
		// Time reached, proceed with action
		executeAction(action, message, confirmation)
	case sig := <-sigChan:
		// Signal received, cancel action
		timer.Stop()
		fmt.Printf("\nReceived signal %v, cancelling scheduled action.\n", sig)
		logger.Printf("Cancelled %s action due to signal %v\n", action, sig)
		return nil
	}

	return nil
}

func sendWallMessage(message string) error {
	// Send a message to all users on the system using the 'wall' command (Unix-like systems only).
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		logVerbose("Wall message feature is not supported on this OS.")
		return fmt.Errorf("wall not supported on %s", runtime.GOOS)
	}

	logVerbose("Sending wall message: " + message)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wall", message)
	if err := cmd.Run(); err != nil {
		logger.Printf("Failed to send wall message: %v\n", err)
		return err
	}

	return nil
}

func executeAction(action string, message string, confirmation bool) {
	// Perform the requested action after optional confirmation and message broadcasting.
	if confirmation && !confirmAction() {
		fmt.Println("Action cancelled.")
		logger.Println("Action cancelled by user.")
		return
	}

	if message != "" {
		if err := sendWallMessage(message); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to send message: %v\n", err)
		}
	}

	logVerbose("Executing " + action + " action.")
	if err := executeSystemCommand(action); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to execute %s: %v\n", action, err)
		logger.Printf("Failed to execute %s: %v\n", action, err)
		os.Exit(1)
	}
}

func confirmAction() bool {
	// Prompt the user for confirmation before proceeding with an action.
	fmt.Print("Are you sure you want to proceed with the action? (y/n): ")
	timeout := getFlagInt(confirmTimeoutIndex)
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	responseChan := make(chan string, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			responseChan <- ""
			return
		}
		responseChan <- strings.TrimSpace(response)
	}()

	select {
	case <-timer.C:
		fmt.Println("\nConfirmation timeout expired. Action cancelled for safety.")
		logger.Println("Confirmation timeout expired. Action cancelled.")
		return false
	case response := <-responseChan:
		timer.Stop()
		if len(response) == 0 {
			return false
		}
		confirmed := response[0] == 'y' || response[0] == 'Y'
		if !confirmed {
			logger.Println("User declined confirmation.")
		}
		return confirmed
	}
}

func executeSystemCommand(action string) error {
	// Execute the system command associated with the specified action.
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		// Use systemctl for modern Linux systems
		cmd = exec.Command("systemctl", action)
	case "windows":
		switch action {
		case "reboot":
			cmd = exec.Command("shutdown", "/r", "/t", "0")
		case "poweroff", "halt":
			cmd = exec.Command("shutdown", "/s", "/t", "0")
		default:
			return fmt.Errorf("unsupported action for Windows: %s", action)
		}
	case "darwin":
		// Check if running with appropriate privileges
		if os.Geteuid() != 0 {
			return fmt.Errorf("root privileges required (run with sudo)")
		}

		switch action {
		case "reboot":
			cmd = exec.Command("shutdown", "-r", "now")
		case "poweroff":
			cmd = exec.Command("shutdown", "-h", "now")
		case "halt":
			cmd = exec.Command("halt")
		default:
			return fmt.Errorf("unsupported action for macOS: %s", action)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}

	logger.Printf("%s action executed successfully.\n", action)
	return nil
}

func getFlagInt(index int) int {
	// Retrieve an integer value from the appFlags based on the index.
	return *(appFlags[index].value.(*int))
}

func logVerbose(message string) {
	// Log a message if verbose output is enabled.
	if *(appFlags[verboseIndex].value.(*bool)) {
		logger.Println(message)
		fmt.Println(message)
	}
}

// getPIDFilePath returns the path to the PID file
func getPIDFilePath() string {
	// Try primary location first
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(pidFileDir); err == nil {
			return filepath.Join(pidFileDir, pidFileName)
		}
	}
	// Fallback to /tmp or TEMP directory
	tempDir := pidFileDirAlt
	if runtime.GOOS == "windows" {
		tempDir = os.Getenv("TEMP")
		if tempDir == "" {
			tempDir = os.TempDir()
		}
	}
	return filepath.Join(tempDir, pidFileName)
}

// writePIDFile writes the current process PID and schedule info to a file
func writePIDFile(scheduleInfo ScheduleInfo) error {
	pidPath := getPIDFilePath()
	scheduleInfo.PID = os.Getpid()
	scheduleInfo.CreatedAt = time.Now()

	data, err := json.Marshal(scheduleInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule info: %v", err)
	}

	if err := os.WriteFile(pidPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	logVerbose(fmt.Sprintf("Created PID file: %s", pidPath))
	return nil
}

// readPIDFile reads the PID file and returns schedule information
func readPIDFile() (*ScheduleInfo, error) {
	pidPath := getPIDFilePath()

	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no scheduled action found")
		}
		return nil, fmt.Errorf("failed to read PID file: %v", err)
	}

	var info ScheduleInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse PID file: %v", err)
	}

	return &info, nil
}

// removePIDFile removes the PID file
func removePIDFile() error {
	pidPath := getPIDFilePath()
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %v", err)
	}
	logVerbose(fmt.Sprintf("Removed PID file: %s", pidPath))
	return nil
}

// processExists checks if a process with the given PID exists
func processExists(pid int) bool {
	// On Unix systems, we can send signal 0 to check if process exists
	if runtime.GOOS != "windows" {
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}

	// On Windows, try to open the process
	// This is a simplified check
	process, err := os.FindProcess(pid)
	return err == nil && process != nil
}

// cancelScheduledAction cancels a pending scheduled action
func cancelScheduledAction() error {
	info, err := readPIDFile()
	if err != nil {
		return err
	}

	// Check if the process still exists
	if !processExists(info.PID) {
		// Process doesn't exist, just remove the stale PID file
		if err := removePIDFile(); err != nil {
			return fmt.Errorf("removed stale PID file, but encountered error: %v", err)
		}
		return fmt.Errorf("no active scheduled action found (stale PID file removed)")
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}

	if err := process.Signal(os.Interrupt); err != nil {
		// Try SIGKILL if SIGTERM fails
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %v", err)
		}
	}

	// Remove the PID file
	if err := removePIDFile(); err != nil {
		logger.Printf("Warning: failed to remove PID file: %v", err)
	}

	fmt.Printf("Cancelled scheduled %s action (PID: %d)\n", info.Action, info.PID)
	logger.Printf("Cancelled scheduled %s action (PID: %d)\n", info.Action, info.PID)

	return nil
}

// showScheduleStatus displays the status of any pending scheduled actions
func showScheduleStatus() error {
	info, err := readPIDFile()
	if err != nil {
		fmt.Println("No scheduled actions pending.")
		return nil
	}

	// Check if process is still running
	if !processExists(info.PID) {
		fmt.Println("No active scheduled actions (stale PID file found).")
		_ = removePIDFile()
		return nil
	}

	fmt.Println("Scheduled Action Status:")
	fmt.Println("========================")
	fmt.Printf("Action:       %s\n", info.Action)
	fmt.Printf("PID:          %d\n", info.PID)

	if info.ScheduledTime != "" {
		fmt.Printf("Scheduled At: %s\n", info.ScheduledTime)
	}
	if info.Delay > 0 {
		fmt.Printf("Delay:        %d minutes\n", info.Delay)
	}
	if info.Message != "" {
		fmt.Printf("Message:      %s\n", info.Message)
	}

	fmt.Printf("Created:      %s\n", info.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func cleanup() {
	// Close log file handle
	if logWriter != nil {
		logWriter.Close()
	}
}

func main() {
	defer cleanup()

	// Parse the command-line flags.
	flag.Parse()

	// Display version information if the version flag is set and exit.
	if *appFlags[versionIndex].value.(*bool) {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Handle status flag
	if *appFlags[statusIndex].value.(*bool) {
		if err := showScheduleStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle cancel flag
	if *appFlags[cancelIndex].value.(*bool) {
		if err := cancelScheduledAction(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Validate that user has appropriate privileges on Unix systems
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "Error: Root privileges required. Please run with sudo.\n")
		logger.Println("Attempted to run without root privileges.")
		os.Exit(1)
	}

	// Determine the action to take based on flags provided by the user.
	action := "reboot" // Default action is to reboot.
	conflictingFlags := 0

	if *(appFlags[haltIndex].value.(*bool)) {
		action = "halt"
		conflictingFlags++
	}
	if *(appFlags[poweroffIndex].value.(*bool)) {
		action = "poweroff"
		conflictingFlags++
	}
	if *(appFlags[shutdownIndex].value.(*bool)) {
		action = "poweroff"
		conflictingFlags++
	}
	if *(appFlags[rebootIndex].value.(*bool)) && conflictingFlags > 0 {
		conflictingFlags++
	}

	if conflictingFlags > 1 {
		fmt.Fprintf(os.Stderr, "Error: Multiple conflicting actions specified. Choose only one.\n")
		os.Exit(1)
	}

	// Handle scheduled time if provided.
	if *(appFlags[timeIndex].value.(*string)) != "" {
		handleScheduledTime(*(appFlags[timeIndex].value.(*string)), action)
		return
	}

	// Proceed with a delayed action if a delay is specified.
	handleDelay(*(appFlags[delayIndex].value.(*int)), action)
}

// handleScheduledTime schedules an action at a specific time.
func handleScheduledTime(timeStr, action string) {
	message := *(appFlags[messageIndex].value.(*string))
	confirmation := *(appFlags[confirmIndex].value.(*bool))

	// Write PID file for scheduled action
	scheduleInfo := ScheduleInfo{
		Action:        action,
		ScheduledTime: timeStr,
		Message:       message,
	}

	if err := writePIDFile(scheduleInfo); err != nil {
		logger.Printf("Warning: failed to write PID file: %v\n", err)
	}

	// Ensure PID file is removed when done
	defer func() { _ = removePIDFile() }()

	// Attempt to schedule and handle errors if any.
	if err := scheduleAtSpecificTime(timeStr, action, message, confirmation); err != nil {
		logger.Printf("Error scheduling action: %v\n", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// handleDelay sets a delay before executing an action.
func handleDelay(delay int, action string) {
	message := *(appFlags[messageIndex].value.(*string))
	confirmation := *(appFlags[confirmIndex].value.(*bool))

	// Log and wait if a delay is set, then execute the action.
	if delay > 0 {
		// Write PID file for delayed action
		scheduleInfo := ScheduleInfo{
			Action:  action,
			Delay:   delay,
			Message: message,
		}

		if err := writePIDFile(scheduleInfo); err != nil {
			logger.Printf("Warning: failed to write PID file: %v\n", err)
		}

		// Ensure PID file is removed when done
		defer func() { _ = removePIDFile() }()

		// Set up signal handling for cancellation
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		logger.Printf("%s scheduled in %d minutes.\n", action, delay)
		fmt.Printf("%s scheduled in %d minutes.\n", action, delay)
		fmt.Println("Press Ctrl+C to cancel, or use 'sysreboot --cancel' from another terminal.")

		// Wait for either the delay to expire or a signal
		timer := time.NewTimer(time.Duration(delay) * time.Minute)
		select {
		case <-timer.C:
			// Delay completed, proceed with action
		case sig := <-sigChan:
			// Signal received, cancel action
			timer.Stop()
			fmt.Printf("\nReceived signal %v, cancelling scheduled action.\n", sig)
			logger.Printf("Cancelled %s action due to signal %v\n", action, sig)
			return
		}
	}

	executeAction(action, message, confirmation)
}
