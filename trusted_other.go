//go:build !windows

package main

import "fmt"

func trustedWindowsSystemDirectory() (string, error) {
	return "", fmt.Errorf("Windows system directory is unavailable")
}
