//go:build !windows

package tui

import "os"

func setupTerminalOutput(_ *os.File) (func(), bool) {
	return func() {}, true
}
