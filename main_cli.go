//go:build cli

package main

import (
	"fmt"
	"os"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/cli"
)

func main() {
	service, err := appsvc.New(appsvc.DefaultAppName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitRuntimeError)
	}

	if len(os.Args) == 1 {
		if canRunTerminal() {
			os.Exit(cli.Run([]string{"tui"}, os.Stdout, os.Stderr, service))
		}
		os.Exit(cli.Run([]string{"help"}, os.Stdout, os.Stderr, service))
	}

	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, service))
}

func canRunTerminal() bool {
	stdin, err := os.Stdin.Stat()
	if err != nil || stdin.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stdout, err := os.Stdout.Stat()
	return err == nil && stdout.Mode()&os.ModeCharDevice != 0
}
