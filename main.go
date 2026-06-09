//go:build !cli

package main

import (
	"embed"
	"fmt"
	"os"
	"runtime"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	service, err := appsvc.New(appsvc.DefaultAppName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitRuntimeError)
	}

	args := os.Args
	forceGUI := hasFlag(args[1:], "--gui")
	args = removeFlag(args, "--gui")

	if shouldRunCLI(args) {
		os.Exit(cli.Run(args[1:], os.Stdout, os.Stderr, service))
	}

	if !forceGUI && shouldFallbackToTUI() {
		fmt.Fprintln(os.Stderr, "No Linux GUI environment detected, starting terminal UI. Use --gui to force GUI startup.")
		os.Exit(cli.Run([]string{"tui"}, os.Stdout, os.Stderr, service))
	}

	app := NewApp(service)

	err = wails.Run(&options.App{
		Title:     appsvc.DefaultAppName,
		Width:     1440,
		Height:    980,
		MinWidth:  1180,
		MinHeight: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		if !forceGUI && canRunTUI() {
			fmt.Fprintln(os.Stderr, "GUI startup failed, starting terminal UI instead:", err)
			os.Exit(cli.Run([]string{"tui"}, os.Stdout, os.Stderr, service))
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cli.ExitRuntimeError)
	}
}

func shouldRunCLI(args []string) bool {
	return len(args) > 1 && cli.IsCommand(args[1])
}

func shouldFallbackToTUI() bool {
	return runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" && canRunTUI()
}

func canRunTUI() bool {
	stdin, err := os.Stdin.Stat()
	if err != nil || stdin.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stdout, err := os.Stdout.Stat()
	return err == nil && stdout.Mode()&os.ModeCharDevice != 0
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func removeFlag(args []string, flag string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == flag {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}
