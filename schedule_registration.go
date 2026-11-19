package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const schedulePollInterval = 250 * time.Millisecond

// claimRegisteredSchedule arbitrates both execution and cancellation against
// the same registration. A late canceller must not report success for a
// schedule already removed or replaced by a different registration.
func claimRegisteredSchedule(id string) (bool, error) {
	if _, err := scheduleGateName(id); err != nil {
		return false, err
	}
	current, err := readPIDFile()
	if err != nil || current.ScheduleID != id {
		return false, errors.New("schedule registration changed or disappeared; decision not confirmed")
	}
	root, err := os.OpenRoot(filepath.Dir(getPIDFilePath()))
	if err != nil {
		return false, fmt.Errorf("open schedule directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	won, err := claimScheduleGate(root.OpenFile, id)
	if err != nil || !won {
		return won, err
	}
	current, err = readPIDFile()
	if err != nil || current.ScheduleID != id {
		return false, errors.New("schedule registration changed or disappeared after decision; verify the scheduler")
	}
	return true, nil
}

func scheduleDecisionExists(id string) (bool, error) {
	name, err := scheduleGateName(id)
	if err != nil {
		return false, err
	}
	root, err := os.OpenRoot(filepath.Dir(getPIDFilePath()))
	if err != nil {
		return false, fmt.Errorf("open schedule directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	// Even a malformed marker or symlink means a decision cannot safely be
	// claimed. Do not follow a symlink or delete a marker here.
	_, err = root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect schedule decision: %w", err)
	}
	return true, nil
}

// waitForScheduleDecision returns true only to the scheduler that wins the
// exclusive decision. A cancellation marker is checked while waiting and
// enforced atomically at the deadline, regardless of the polling interval.
func waitForScheduleDecision(ctx context.Context, duration time.Duration, id string) (bool, error) {
	if ctx.Err() != nil {
		return false, nil
	}
	if _, err := scheduleGateName(id); err != nil {
		return false, err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	ticker := time.NewTicker(schedulePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, nil
		case <-ticker.C:
			decided, err := scheduleDecisionExists(id)
			if err != nil || decided {
				return false, err
			}
		case <-timer.C:
			if ctx.Err() != nil {
				return false, nil
			}
			won, err := claimRegisteredSchedule(id)
			return won && ctx.Err() == nil, err
		}
	}
}

// Only the scheduler may clear its own registration. Remove the PID record
// first; retaining the marker on failure prevents a second decision.
func cleanupScheduleRegistration(id string) {
	name, err := scheduleGateName(id)
	if err != nil {
		return
	}
	current, err := readPIDFile()
	if err != nil || current.ScheduleID != id {
		return
	}
	root, err := os.OpenRoot(filepath.Dir(getPIDFilePath()))
	if err != nil {
		logger.Printf("Warning: unable to open schedule directory for cleanup: %v", err)
		return
	}
	defer func() { _ = root.Close() }()
	if err := root.Remove(filepath.Base(getPIDFilePath())); err != nil {
		logger.Printf("Warning: unable to remove schedule registration: %v", err)
		return
	}
	if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Printf("Warning: unable to remove schedule decision: %v", err)
	}
}
