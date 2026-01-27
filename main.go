// sysreboot is an enhanced reboot tool with smart capabilities.
// Author: Eric Sobczak
// License: MIT
// Repository: https://github.com/esobczak1970/sysreboot
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Constants for application metadata
const (
	appName    = "sysreboot"
	appVersion = "0.1.3"
)

// Enumeration for index mapping of the flags (must match appFlags order)
const (
	confirmIndex = iota
	confirmTimeoutIndex
	delayIndex
	haltIndex
	messageIndex
	poweroffIndex
	rebootIndex
	shutdownIndex
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
	{"confirm", "c", new(bool), false, "Require confirmation before performing the action."},
	{"confirm-timeout", "ct", new(int), 10, "Confirmation timeout in seconds."},
	{"delay", "d", new(int), 0, "Delay in minutes before performing the action."},
	{"halt", "h", new(bool), false, "Halt the machine."},
	{"message", "m", new(string), "", "Message to send to all users before performing the action."},
	{"poweroff", "p", new(bool), false, "Power-off the machine."},
	{"reboot", "r", new(bool), true, "Reboot the machine (default action)."},
	{"shutdown", "s", new(bool), false, "Shutdown the machine (alias for poweroff)."},
	{"time", "t", new(string), "", "Specific time for the action in HH:MM format (24-hour)."},
	{"verbose", "vb", new(bool), false, "Output more information."},
	{"version", "v", new(bool), false, "Show application version."},
}

var (
	logFile   string      // Path to the log file.
	logger    *log.Logger // Logger instance for the application.
	logWriter io.WriteCloser
)

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

	time.Sleep(durationUntilReboot) // Wait until the specified time.
	executeAction(action, message, confirmation)
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
		logger.Printf("%s scheduled in %d minutes.\n", action, delay)
		fmt.Printf("%s scheduled in %d minutes.\n", action, delay)
		time.Sleep(time.Duration(delay) * time.Minute)
	}

	executeAction(action, message, confirmation)
}
