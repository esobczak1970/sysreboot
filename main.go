// sysreboot is an enhanced reboot tool with smart capabilities.
// Author: Eric Sobczak
// License: MIT
// Repository: https://github.com/esobczak1970/sysreboot
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Constants for application metadata
const (
	appName    = "sysreboot"
	appVersion = "0.1.2"
)

// Enumeration for index mapping of the flags
const (
	haltIndex = iota
	poweroffIndex
	rebootIndex
	delayIndex
	messageIndex
	confirmIndex
	confirmTimeoutIndex
	verboseIndex
	timeIndex
	versionIndex
	shutdownIndex
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
var appFlags = []flagData{
	// Flags are organized alphabetically by longName for readability.
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
	logFile string      // Path to the log file.
	logger  *log.Logger // Logger instance for the application.
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
	logger = log.New(file, appName+": ", log.Ldate|log.Ltime|log.Lshortfile)

	// Override the default flag usage message with a custom one.
	flag.Usage = customUsage
}

func getLogFileDirectory() string {
	// Get the appropriate log file directory based on the operating system.
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
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
		return fmt.Errorf("invalid time format: %v", err)
	}

	// Calculate how long to wait until the specified time.
	now := time.Now()
	durationUntilReboot := time.Until(now.Truncate(24 * time.Hour).Add(time.Hour*time.Duration(rebootTime.Hour()) + time.Minute*time.Duration(rebootTime.Minute())))
	if durationUntilReboot < 0 {
		durationUntilReboot += 24 * time.Hour // Schedule for the next day if time is in the past.
	}

	logger.Printf("%s scheduled at %s (in %s).\n", action, rebootTime.Format("15:04"), durationUntilReboot)
	fmt.Printf("%s scheduled at %s (in %s).\n", action, rebootTime.Format("15:04"), durationUntilReboot)

	time.Sleep(durationUntilReboot) // Wait until the specified time.
	executeAction(action, message, confirmation)
	return nil
}

func sendWallMessage(message string) {
	// Send a message to all users on the system using the 'wall' command (Unix-like systems only).
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		if *(appFlags[verboseIndex].value.(*bool)) {
			logger.Println("Wall message feature is not supported on this OS.")
		}
		return
	}

	if *(appFlags[verboseIndex].value.(*bool)) {
		logger.Println("Sending wall message.")
	}
	cmd := exec.Command("wall", message)
	err := cmd.Run()
	if err != nil {
		logger.Printf("Failed to send wall message: %v\n", err)
	}
}

func executeAction(action string, message string, confirmation bool) {
	// Perform the requested action after optional confirmation and message broadcasting.
	if confirmation && !confirmAction() {
		fmt.Println("Action cancelled.")
		logger.Println("Action cancelled by user.")
		return
	}

	if message != "" {
		sendWallMessage(message)
	}

	logVerbose("Executing " + action + " action.")
	executeSystemCommand(action)
}

func confirmAction() bool {
	// Prompt the user for confirmation before proceeding with an action.
	fmt.Println("Are you sure you want to proceed with the action? (y/n)")
	timer := time.NewTimer(time.Duration(getFlagInt(confirmTimeoutIndex)) * time.Second)
	responseChan := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		responseChan <- response
	}()

	select {
	case <-timer.C:
		fmt.Println("\nConfirmation timer expired, proceeding with action.")
		return true
	case response := <-responseChan:
		timer.Stop()
		return response[0] == 'y' || response[0] == 'Y'
	}
}

func executeSystemCommand(action string) {
	// Execute the system command associated with the specified action.
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("systemctl", action)
	case "windows":
		if action == "reboot" {
			cmd = exec.Command("shutdown", "/r", "/t", "0")
		} else if action == "poweroff" {
			cmd = exec.Command("shutdown", "/s", "/t", "0")
		}
	case "darwin":
		if action == "reboot" {
			cmd = exec.Command("sudo", "shutdown", "-r", "now")
		} else if action == "poweroff" {
			cmd = exec.Command("sudo", "shutdown", "-h", "now")
		} else if action == "halt" {
			cmd = exec.Command("sudo", "halt")
		}
	default:
		logger.Printf("Unsupported action or OS: %s on %s", action, runtime.GOOS)
		return
	}

	if err := cmd.Run(); err != nil {
		logger.Printf("Failed to execute %s: %v\n", action, err)
	} else {
		logger.Printf("%s action executed successfully.\n", action)
	}
}

func getFlagInt(index int) int {
	// Retrieve an integer value from the appFlags based on the index.
	return *(appFlags[index].value.(*int))
}

func logVerbose(message string) {
	// Log a message if verbose output is enabled.
	if *(appFlags[verboseIndex].value.(*bool)) {
		logger.Println(message)
	}
}

func main() {
	// Parse the command-line flags.
	flag.Parse()

	// Display version information if the version flag is set and exit.
	if *appFlags[versionIndex].value.(*bool) {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Determine the action to take based on flags provided by the user.
	action := "reboot" // Default action is to reboot.
	if *(appFlags[haltIndex].value.(*bool)) {
		action = "halt"
	} else if *(appFlags[poweroffIndex].value.(*bool)) || *(appFlags[shutdownIndex].value.(*bool)) { // Modified line
		action = "poweroff"
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
