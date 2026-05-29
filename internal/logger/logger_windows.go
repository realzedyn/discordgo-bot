//go:build windows

package logger

import (
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

const (
	enableVirtualTerminalProcessing = 0x0004
)

func enableWindowsANSI() {
	stdout := syscall.Handle(syscall.Stdout)

	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(uintptr(stdout), uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		return
	}

	mode |= enableVirtualTerminalProcessing
	procSetConsoleMode.Call(uintptr(stdout), uintptr(mode))
}
