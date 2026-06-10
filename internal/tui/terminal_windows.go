//go:build windows

package tui

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	enableProcessedOutput           = 0x0001
	enableVirtualTerminalProcessing = 0x0004
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func setupTerminalOutput(file *os.File) (func(), bool) {
	handle := syscall.Handle(file.Fd())

	var originalMode uint32
	ok, _, _ := procGetConsoleMode.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&originalMode)),
	)
	if ok == 0 {
		return func() {}, false
	}

	// Do not set DISABLE_NEWLINE_AUTO_RETURN here.
	// The TUI uses normal fmt.Fprintln calls. On Windows, disabling newline
	// auto-return makes '\n' move down without returning to column 0, which
	// turns the dashboard into a diagonal/stair-step layout.
	nextMode := originalMode | enableProcessedOutput | enableVirtualTerminalProcessing
	ok, _, _ = procSetConsoleMode.Call(uintptr(handle), uintptr(nextMode))
	if ok == 0 {
		return func() {}, false
	}

	restore := func() {
		procSetConsoleMode.Call(uintptr(handle), uintptr(originalMode))
	}
	return restore, true
}
