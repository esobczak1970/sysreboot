package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

// Each registered schedule owns a random identity and a single exclusive
// decision file. Execution and cancellation compete for the same file;
// neither operation needs to signal or trust the recorded process PID.
func newScheduleID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate schedule identity: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}

func scheduleGateName(id string) (string, error) {
	data, err := hex.DecodeString(id)
	if err != nil || len(data) != 16 || hex.EncodeToString(data) != id {
		return "", errors.New("missing or invalid schedule identity; verify the original scheduler before manual recovery")
	}
	return "sysreboot-" + id + ".decision", nil
}

// claimScheduleGate uses a caller-supplied, trusted-directory OpenFile method
// (os.Root.OpenFile in production). O_EXCL makes the empty marker itself the
// decision, so no reader can observe a partially written decision payload.
// A failed close is deliberately not success; the existing marker still
// prevents the competing operation from proceeding.
func claimScheduleGate(open func(string, int, os.FileMode) (*os.File, error), id string) (bool, error) {
	name, err := scheduleGateName(id)
	if err != nil {
		return false, err
	}
	file, err := open(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim schedule decision: %w", err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("close schedule decision: %w", err)
	}
	return true, nil
}
