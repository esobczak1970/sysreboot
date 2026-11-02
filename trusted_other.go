//go:build !windows

package main

import "fmt"

var trustedWindowsSystemDirectory = func() (string, error) {
	return "", fmt.Errorf("windows system directory is unavailable")
}
