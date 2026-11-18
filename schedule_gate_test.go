package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestScheduleGateExactlyOneDecision(t *testing.T) {
	dir := t.TempDir()
	open := func(name string, flags int, mode os.FileMode) (*os.File, error) {
		return os.OpenFile(filepath.Join(dir, name), flags, mode)
	}
	for range 128 {
		id, err := newScheduleID()
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		outcomes := make(chan bool, 2)
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				won, err := claimScheduleGate(open, id)
				if err != nil {
					t.Error(err)
				}
				outcomes <- won
			}()
		}
		close(start)
		wg.Wait()
		close(outcomes)
		wins := 0
		for won := range outcomes {
			if won {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("decision wins = %d, want exactly one", wins)
		}
	}
}

func TestScheduleGateRejectsLegacyAndInvalidIdentity(t *testing.T) {
	open := func(string, int, os.FileMode) (*os.File, error) {
		t.Fatal("invalid schedule must not touch the filesystem")
		return nil, errors.New("unexpected open")
	}
	for _, id := range []string{"", "../outside", "0000", "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", "gggggggggggggggggggggggggggggggg"} {
		if won, err := claimScheduleGate(open, id); won || err == nil {
			t.Fatalf("invalid identity %q accepted", id)
		}
	}
}

func TestScheduleGateRejectsErrorsAndDoesNotReuseOldIdentity(t *testing.T) {
	first, err := newScheduleID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newScheduleID()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	open := func(name string, flags int, mode os.FileMode) (*os.File, error) {
		return os.OpenFile(filepath.Join(dir, name), flags, mode)
	}
	for _, id := range []string{first, second} {
		if won, err := claimScheduleGate(open, id); !won || err != nil {
			t.Fatalf("distinct schedule did not win: %v, %v", won, err)
		}
		if won, err := claimScheduleGate(open, id); won || err != nil {
			t.Fatalf("repeated decision was accepted: %v, %v", won, err)
		}
	}
	deny := func(string, int, os.FileMode) (*os.File, error) { return nil, os.ErrPermission }
	if won, err := claimScheduleGate(deny, second); won || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("filesystem error accepted: %v, %v", won, err)
	}
}
