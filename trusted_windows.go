package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// kernel32 is a Windows KnownDLL; query the OS instead of trusting SystemRoot.
func trustedWindowsSystemDirectory() (string, error) {
	buffer := make([]uint16, 32768)
	n, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW").Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if n == 0 {
		return "", fmt.Errorf("GetSystemDirectoryW: %w", err)
	}
	if n >= uintptr(len(buffer)) {
		return "", fmt.Errorf("Windows system directory exceeds buffer")
	}
	return syscall.UTF16ToString(buffer[:n]), nil
}
