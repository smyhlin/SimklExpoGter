package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"
	"SimklExpoGter/internal/simkl"
	// The TUI package provides a self‑contained terminal interface for the
	// application.  It is only imported here so that the CLI can launch
	// the interactive console when requested.
	"SimklExpoGter/internal/tui"
)

const (
	ExitSuccess      = 0
	ExitUsageError   = 2
	ExitPrerequisite = 3
	ExitRuntimeError = 4
)

type Service interface {
	Path() string
	ConfigSummary() (appsvc.ConfigSummary, error)
	AuthSummary() (appsvc.AuthSummary, error)
	SaveSettings(appsvc.SaveSettingsInput) (appsvc.SaveSettingsResult, error)
	RequestDeviceCode() (simkl.DeviceCodeResponse, error)
	PollDeviceCode(string) (simkl.DeviceCodeStatusResponse, error)
	SaveAccessToken(string) (config.Settings, error)
	StandardAuthURL() (string, error)
	ExchangeOAuthCode(string) (config.Settings, error)
	ClearAccessToken() (config.Settings, error)
	RunExport(exporter.Request) (exporter.Result, error)
	RunScheduledExport(exporter.Request, string) (appsvc.ScheduledExportResult, error)
	ScheduleState() (appsvc.ScheduleState, error)
	SaveSchedule(appsvc.ScheduleSettingsInput) (config.Settings, appsvc.ScheduleState, error)
}

func IsCommand(name string) bool {
	// IsCommand returns true if the provided subcommand should trigger the
	// CLI mode.  In addition to the built‑in commands, the special "tui"
	// command launches the interactive terminal UI.
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "help", "run", "config", "auth", "schedule", "tui", "--tui", "terminal", "-h", "--help":
		return true
	default:
		return false
	}
}

func Run(args []string, stdout, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		printRootHelp(stdout)
		return ExitSuccess
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "help", "-h", "--help":
		return runHelp(args[1:], stdout, stderr)
	case "run":
		return runExportCommand(args[1:], stdout, stderr, service)
	case "config":
		return runConfigCommand(args[1:], stdout, stderr, service)
	case "auth":
		return runAuthCommand(args[1:], stdout, stderr, service)
	case "schedule":
		return runScheduleCommand(args[1:], stdout, stderr, service)
	case "tui", "--tui", "terminal":
		// Launch the interactive terminal user interface.  Any error from
		// bubbletea is printed to stderr and results in a non‑zero exit
		// status.  When the TUI exits normally we return ExitSuccess.
		if err := tui.Run(service); err != nil {
			fmt.Fprintln(stderr, err)
			return ExitRuntimeError
		}
		return ExitSuccess
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printRootHelp(stderr)
		return ExitUsageError
	}
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printRootHelp(stdout)
		return ExitSuccess
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "run":
		printRunHelp(stdout)
	case "config":
		printConfigHelp(stdout)
	case "auth":
		printAuthHelp(stdout)
	case "tui", "terminal":
		printTUIHelp(stdout)
	default:
		fmt.Fprintf(stderr, "unknown help topic %q\n\n", args[0])
		printRootHelp(stderr)
		return ExitUsageError
	}

	return ExitSuccess
}

func runConfigCommand(args []string, stdout, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		printConfigHelp(stderr)
		return ExitUsageError
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "path":
		fmt.Fprintln(stdout, service.Path())
		return ExitSuccess
	case "show":
		summary, err := service.ConfigSummary()
		if err != nil {
			return printError(stderr, err)
		}
		return printJSON(stdout, summary)
	case "set":
		return runConfigSetCommand(args[1:], stdout, stderr, service)
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n\n", args[0])
		printConfigHelp(stderr)
		return ExitUsageError
	}
}

func runConfigSetCommand(args []string, stdout, stderr io.Writer, service Service) int {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var clientID string
	var clientSecret string
	var secretAlias string
	var output string
	var backupStorage string
	var telegramBotToken string
	var telegramChatID string
	var telegramThreadID string
	var telegramCaption string

	fs.StringVar(&clientID, "client-id", "", "Persist the Simkl client ID.")
	fs.StringVar(&clientSecret, "client-secret", "", "Persist the Simkl client secret.")
	fs.StringVar(&secretAlias, "secret", "", "Alias for --client-secret.")
	fs.StringVar(&output, "output", "", "Persist the default export directory.")
	fs.StringVar(&backupStorage, "backup-storage", "", "Persist backup storage: local, gdrive, telegram.")
	fs.StringVar(&telegramBotToken, "telegram-bot-token", "", "Persist Telegram bot token.")
	fs.StringVar(&telegramChatID, "telegram-chat-id", "", "Persist Telegram chat ID.")
	fs.StringVar(&telegramThreadID, "telegram-thread-id", "", "Persist Telegram forum topic/thread ID.")
	fs.StringVar(&telegramCaption, "telegram-caption", "", "Persist Telegram backup caption.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printConfigSetHelp(stdout)
			return ExitSuccess
		}
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr)
		printConfigSetHelp(stderr)
		return ExitUsageError
	}

	resolvedSecret, err := resolveAliasedValue(clientSecret, secretAlias, "--client-secret", "--secret")
	if err != nil {
		return printError(stderr, err)
	}

	visited := visitedFlags(fs)
	input := appsvc.SaveSettingsInput{
		ClientID:           clientID,
		ClientSecret:       resolvedSecret,
		ExportDirectory:     output,
		BackupStorage:       backupStorage,
		TelegramBotToken:    telegramBotToken,
		TelegramChatID:      telegramChatID,
		TelegramThreadID:    telegramThreadID,
		TelegramCaption:     telegramCaption,
		SetClientID:         visited["client-id"],
		SetClientSecret:     visited["client-secret"] || visited["secret"],
		SetExportDirectory:  visited["output"],
		SetBackupStorage:    visited["backup-storage"],
		SetTelegramBotToken: visited["telegram-bot-token"],
		SetTelegramChatID:   visited["telegram-chat-id"],
		SetTelegramThreadID: visited["telegram-thread-id"],
		SetTelegramCaption:  visited["telegram-caption"],
	}

	if !input.SetClientID && !input.SetClientSecret && !input.SetExportDirectory && !input.SetBackupStorage && !input.SetTelegramBotToken && !input.SetTelegramChatID && !input.SetTelegramThreadID && !input.SetTelegramCaption {
		return printError(stderr, newUsageError("config set requires at least one flag"))
	}

	if _, err := service.SaveSettings(input); err != nil {
		return printError(stderr, err)
	}

	fmt.Fprintln(stdout, "Settings updated.")
	return ExitSuccess
}

func runAuthCommand(args []string, stdout, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		printAuthHelp(stderr)
		return ExitUsageError
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "login", "pin", "device", "url":
		if strings.EqualFold(strings.TrimSpace(args[0]), "url") {
			fmt.Fprintln(stderr, "Simkl OAuth URL auth requires an exact registered redirect_uri. Using PIN login instead, which is correct for CLI/TUI.")
		}
		return runAuthPINCommand(args[1:], stdout, stderr, service)
	case "exchange":
		return runAuthExchangeCommand(args[1:], stdout, stderr, service)
	case "oauth-url":
		authURL, err := service.StandardAuthURL()
		if err != nil {
			return printError(stderr, err)
		}
		fmt.Fprintln(stdout, authURL)
		return ExitSuccess
	case "status":
		summary, err := service.AuthSummary()
		if err != nil {
			return printError(stderr, err)
		}
		return printJSON(stdout, summary)
	case "clear":
		if _, err := service.ClearAccessToken(); err != nil {
			return printError(stderr, err)
		}
		fmt.Fprintln(stdout, "Access token cleared.")
		return ExitSuccess
	default:
		fmt.Fprintf(stderr, "unknown auth command %q\n\n", args[0])
		printAuthHelp(stderr)
		return ExitUsageError
	}
}

func runAuthPINCommand(args []string, stdout, stderr io.Writer, service Service) int {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var noPoll bool
	fs.BoolVar(&noPoll, "no-poll", false, "Print the PIN code and exit instead of waiting for approval.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthPINHelp(stdout)
			return ExitSuccess
		}
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr)
		printAuthPINHelp(stderr)
		return ExitUsageError
	}

	deviceCode, err := service.RequestDeviceCode()
	if err != nil {
		return printError(stderr, err)
	}

	verificationURL := strings.TrimSpace(deviceCode.VerificationURL)
	if verificationURL == "" {
		verificationURL = "https://simkl.com/pin/"
	}

	fmt.Fprintln(stdout, "Simkl PIN login")
	fmt.Fprintf(stdout, "Open: %s\n", verificationURL)
	fmt.Fprintf(stdout, "Enter code: %s\n", strings.TrimSpace(deviceCode.UserCode))

	if noPoll {
		fmt.Fprintln(stdout, "Run this command again without --no-poll after approving, or use the TUI login.")
		return ExitSuccess
	}

	interval := deviceCode.Interval
	if interval <= 0 {
		interval = 5
	}
	expiresIn := deviceCode.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 900
	}

	fmt.Fprintf(stdout, "Waiting for approval for up to %d seconds", expiresIn)
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			fmt.Fprintln(stdout)
			return printError(stderr, newUsageError("Simkl PIN login expired; run auth login again"))
		}

		time.Sleep(time.Duration(interval) * time.Second)
		status, err := service.PollDeviceCode(deviceCode.UserCode)
		if err != nil {
			fmt.Fprintln(stdout)
			return printError(stderr, err)
		}

		if strings.EqualFold(strings.TrimSpace(status.Result), "OK") && strings.TrimSpace(status.AccessToken) != "" {
			if _, err := service.SaveAccessToken(status.AccessToken); err != nil {
				fmt.Fprintln(stdout)
				return printError(stderr, err)
			}
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Access token saved.")
			return ExitSuccess
		}

		message := strings.ToLower(strings.TrimSpace(status.Message))
		if strings.Contains(message, "slow") {
			interval += 5
		}
		fmt.Fprint(stdout, ".")
	}
}

func runScheduleCommand(args []string, stdout, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		printScheduleHelp(stderr)
		return ExitUsageError
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "status":
		state, err := service.ScheduleState()
		if err != nil {
			return printError(stderr, err)
		}
		return printJSON(stdout, state)
	case "enable":
		return runScheduleEnableCommand(args[1:], stdout, stderr, service)
	case "disable":
		if _, _, err := service.SaveSchedule(appsvc.ScheduleSettingsInput{Enabled: false}); err != nil {
			return printError(stderr, err)
		}
		fmt.Fprintln(stdout, "Recurring backup disabled.")
		return ExitSuccess
	case "linger":
		return runScheduleLingerCommand(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown schedule command %q\n\n", args[0])
		printScheduleHelp(stderr)
		return ExitUsageError
	}
}

func runScheduleEnableCommand(args []string, stdout, stderr io.Writer, service Service) int {
	fs := flag.NewFlagSet("schedule enable", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var frequency string
	var scheduleTime string
	var daysRaw string
	var outputFormat string
	var fieldMode string
	var contentRaw string
	var activityCheck bool
	var maxBackupAge string
	var noStaleGuard bool

	fs.StringVar(&frequency, "frequency", "daily", "Schedule frequency: daily or weekly.")
	fs.StringVar(&scheduleTime, "time", "02:00", "Run time in HH:MM 24-hour format.")
	fs.StringVar(&daysRaw, "days", "mon", "Comma-separated weekly days: mon,tue,wed,thu,fri,sat,sun.")
	fs.StringVar(&outputFormat, "format", exporter.FormatCSV, "Output format: csv, json, or both.")
	fs.StringVar(&fieldMode, "field-mode", exporter.FieldModeAll, "Field density: all or compact.")
	fs.StringVar(&contentRaw, "content", "shows,movies,anime", "Comma-separated media types.")
	fs.BoolVar(&activityCheck, "activity-check", false, "Use /sync/activities before exporting.")
	fs.StringVar(&maxBackupAge, "max-backup-age", "24h", "Stale threshold: 12h, 24h, 3d, 1w.")
	fs.BoolVar(&noStaleGuard, "no-stale-guard", false, "Always run when the OS scheduler triggers.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printScheduleEnableHelp(stdout)
			return ExitSuccess
		}
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr)
		printScheduleEnableHelp(stderr)
		return ExitUsageError
	}

	content, err := appsvc.NormalizeCLITypes(splitCSV(contentRaw))
	if err != nil {
		return printError(stderr, err)
	}

	_, state, err := service.SaveSchedule(appsvc.ScheduleSettingsInput{
		Enabled:            true,
		Frequency:          strings.TrimSpace(frequency),
		Time:               strings.TrimSpace(scheduleTime),
		Days:               splitCSV(daysRaw),
		OutputFormat:       strings.TrimSpace(outputFormat),
		FieldMode:          strings.TrimSpace(fieldMode),
		Content:            content,
		UseActivityCheck:   activityCheck,
		MaxBackupAge:       strings.TrimSpace(maxBackupAge),
		RunIfBackupIsStale: !noStaleGuard,
	})
	if err != nil {
		return printError(stderr, err)
	}

	fmt.Fprintln(stdout, "Recurring backup enabled.")
	if state.Message != "" {
		fmt.Fprintln(stdout, state.Message)
	}
	return ExitSuccess
}

func runScheduleLingerCommand(stdout, stderr io.Writer) int {
	if runtime.GOOS != "linux" {
		fmt.Fprintln(stderr, "linger is only supported on Linux")
		return ExitUsageError
	}
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		fmt.Fprintln(stderr, "unable to determine current user")
		return ExitRuntimeError
	}
	cmd := exec.Command("loginctl", "enable-linger", user)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(stderr, "failed to enable lingering: %s\n", strings.TrimSpace(string(output)))
		return ExitRuntimeError
	}
	fmt.Fprintf(stdout, "Enabled lingering for %s. systemd user timers can run after logout and boot.\n", user)
	return ExitSuccess
}

func runAuthExchangeCommand(args []string, stdout, stderr io.Writer, service Service) int {
	fs := flag.NewFlagSet("auth exchange", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var code string
	fs.StringVar(&code, "code", "", "OAuth code copied from the Simkl approval page.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthExchangeHelp(stdout)
			return ExitSuccess
		}
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr)
		printAuthExchangeHelp(stderr)
		return ExitUsageError
	}

	if strings.TrimSpace(code) == "" {
		return printError(stderr, newUsageError("auth exchange requires --code"))
	}

	if _, err := service.ExchangeOAuthCode(code); err != nil {
		return printError(stderr, err)
	}

	fmt.Fprintln(stdout, "Access token saved.")
	return ExitSuccess
}

func runExportCommand(args []string, stdout, stderr io.Writer, service Service) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var fieldMode string
	var modeAlias string
	var typesRaw string
	var contentAlias string
	var statusesRaw string
	var extended string
	var outputFormat string
	var grouping string
	var output string
	var filenamePrefix string
	var dateFrom string
	var episodeFiles bool
	var memos bool
	var nextWatchInfo bool
	var episodeWatchedAt bool
	var activityCheck bool
	var scheduled bool
	var maxBackupAge string

	fs.StringVar(&fieldMode, "field-mode", "", "Field density: all or compact.")
	fs.StringVar(&modeAlias, "mode", "", "Alias for --field-mode.")
	fs.StringVar(&typesRaw, "types", "", "Comma-separated media types.")
	fs.StringVar(&contentAlias, "content", "", "Alias for --types.")
	fs.StringVar(&statusesRaw, "status", "", "Comma-separated statuses.")
	fs.StringVar(&extended, "extended", "full", "Extended Simkl payload mode.")
	fs.StringVar(&outputFormat, "format", exporter.FormatCSV, "Output format: csv, json, or both.")
	fs.StringVar(&grouping, "grouping", exporter.GroupingSeparate, "Grouping: single-file or separate-files.")
	fs.StringVar(&output, "output", "", "Export directory override for this run.")
	fs.StringVar(&filenamePrefix, "filename-prefix", "simkl-export", "Filename prefix for exported files.")
	fs.StringVar(&dateFrom, "date-from", "", "Incremental export timestamp.")
	fs.BoolVar(&episodeFiles, "episode-files", true, "Include episode-level export files.")
	fs.BoolVar(&memos, "memos", true, "Include memos in the export request.")
	fs.BoolVar(&nextWatchInfo, "next-watch-info", true, "Include next-watch info in the export request.")
	fs.BoolVar(&episodeWatchedAt, "episode-watched-at", true, "Include episode watched-at timestamps.")
	fs.BoolVar(&activityCheck, "activity-check", false, "Use /sync/activities before exporting.")
	fs.BoolVar(&scheduled, "scheduled", false, "Run as a scheduled backup and skip when the last successful backup is still fresh.")
	fs.StringVar(&maxBackupAge, "max-backup-age", "", "Override scheduled stale threshold, e.g. 12h, 24h, 3d, 1w.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printRunHelp(stdout)
			return ExitSuccess
		}
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr)
		printRunHelp(stderr)
		return ExitUsageError
	}

	resolvedFieldMode, err := resolveAliasedValue(fieldMode, modeAlias, "--field-mode", "--mode")
	if err != nil {
		return printError(stderr, err)
	}
	resolvedTypesRaw, err := resolveAliasedValue(typesRaw, contentAlias, "--types", "--content")
	if err != nil {
		return printError(stderr, err)
	}

	types, err := appsvc.NormalizeCLITypes(splitCSV(resolvedTypesRaw))
	if err != nil {
		return printError(stderr, err)
	}
	statuses, err := appsvc.NormalizeCLIStatuses(splitCSV(statusesRaw))
	if err != nil {
		return printError(stderr, err)
	}

	request := exporter.Request{
		Types:                types,
		Statuses:             statuses,
		DateFrom:             strings.TrimSpace(dateFrom),
		Extended:             strings.TrimSpace(extended),
		EpisodeWatchedAt:     episodeWatchedAt,
		IncludeMemos:         memos,
		IncludeNextWatchInfo: nextWatchInfo,
		OutputFormat:         strings.TrimSpace(outputFormat),
		FieldMode:            strings.TrimSpace(resolvedFieldMode),
		Grouping:             strings.TrimSpace(grouping),
		IncludeEpisodeFiles:  episodeFiles,
		UseActivityCheck:     activityCheck,
		ExportDirectory:      strings.TrimSpace(output),
		FilenamePrefix:       strings.TrimSpace(filenamePrefix),
	}

	if scheduled {
		result, err := service.RunScheduledExport(request, strings.TrimSpace(maxBackupAge))
		if err != nil {
			return printError(stderr, err)
		}
		printScheduledRunResult(stdout, result)
		return ExitSuccess
	}

	result, err := service.RunExport(request)
	if err != nil {
		return printError(stderr, err)
	}

	printRunResult(stdout, result)
	return ExitSuccess
}

func printScheduledRunResult(stdout io.Writer, scheduled appsvc.ScheduledExportResult) {
	if scheduled.Skipped {
		fmt.Fprintf(stdout, "Skipped scheduled backup: %s\n", scheduled.Reason)
		if scheduled.LastBackupAt != "" {
			fmt.Fprintf(stdout, "Last successful backup: %s\n", scheduled.LastBackupAt)
		}
		if scheduled.MaxBackupAge != "" {
			fmt.Fprintf(stdout, "Max backup age: %s\n", scheduled.MaxBackupAge)
		}
		return
	}

	if scheduled.Ran {
		fmt.Fprintln(stdout, "Scheduled backup ran.")
		printRunResult(stdout, scheduled.Result)
		return
	}

	fmt.Fprintln(stdout, "Scheduled backup did not run.")
}

func printRunResult(stdout io.Writer, result exporter.Result) {
	fmt.Fprintf(stdout, "Exported %d items to %s\n", result.ItemCounts["all"], result.OutputDirectory)
	fmt.Fprintf(stdout, "Exported at: %s\n", result.ExportedAt)
	fmt.Fprintln(stdout, "Files:")
	for _, file := range result.Files {
		fmt.Fprintf(stdout, "- %s (%s / %s / %s, %d rows)\n", file.Path, file.Format, file.MediaType, file.Kind, file.Rows)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(stdout, "Warnings:")
		for _, warning := range result.Warnings {
			fmt.Fprintf(stdout, "- %s\n", warning)
		}
	}
}

func printJSON(stdout io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	if err := encoder.Encode(value); err != nil {
		return ExitRuntimeError
	}
	return ExitSuccess
}

func printError(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, err)
	var localUsageErr *usageError
	switch {
	case appsvc.IsUsageError(err):
		return ExitUsageError
	case errors.As(err, &localUsageErr):
		return ExitUsageError
	case appsvc.IsPrerequisiteError(err):
		return ExitPrerequisite
	default:
		return ExitRuntimeError
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return items
}

func resolveAliasedValue(primary, alias, primaryName, aliasName string) (string, error) {
	primary = strings.TrimSpace(primary)
	alias = strings.TrimSpace(alias)
	if primary != "" && alias != "" && primary != alias {
		return "", newUsageError(fmt.Sprintf("%s and %s must match when both are provided", primaryName, aliasName))
	}
	if primary != "" {
		return primary, nil
	}
	return alias, nil
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func newUsageError(message string) error {
	return fmt.Errorf("%w", &usageError{message: message})
}

type usageError struct {
	message string
}

func (e *usageError) Error() string {
	return e.message
}

func printRootHelp(w io.Writer) {
	// Print generic usage information.  We avoid hard‑coding a Windows‑only
	// `.exe` suffix here so that the help output remains valid on Linux and
	// macOS.  Replace `SimklExpoGter` with your binary name if it differs.
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter help")
	fmt.Fprintln(w, "  SimklExpoGter run [flags]")
	fmt.Fprintln(w, "  SimklExpoGter config path")
	fmt.Fprintln(w, "  SimklExpoGter config show")
	fmt.Fprintln(w, "  SimklExpoGter config set [flags]")
	fmt.Fprintln(w, "  SimklExpoGter auth login")
	fmt.Fprintln(w, "  SimklExpoGter auth exchange --code <code>")
	fmt.Fprintln(w, "  SimklExpoGter auth status")
	fmt.Fprintln(w, "  SimklExpoGter auth clear")
	fmt.Fprintln(w, "  SimklExpoGter schedule status")
	fmt.Fprintln(w, "  SimklExpoGter schedule enable [flags]")
	fmt.Fprintln(w, "  SimklExpoGter schedule disable")
	fmt.Fprintln(w, "  SimklExpoGter schedule linger")
	fmt.Fprintln(w, "  SimklExpoGter tui")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  SimklExpoGter config set --client-id abc123 --secret shhh --output ./exports")
	fmt.Fprintln(w, "  SimklExpoGter auth login")
	fmt.Fprintln(w, "  SimklExpoGter auth exchange --code ABCD-1234")
	fmt.Fprintln(w, "  SimklExpoGter run --mode all --content anime,movie,series --output ./exports")
	fmt.Fprintln(w, "  SimklExpoGter schedule enable --frequency daily --time 02:00 --max-backup-age 24h")
}

func printRunHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter run [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --field-mode, --mode         all|compact (default all)")
	fmt.Fprintln(w, "  --types, --content           Comma-separated media types (anime,movie,series)")
	fmt.Fprintln(w, "  --status                     Comma-separated statuses")
	fmt.Fprintln(w, "  --extended                   full|full_anime_seasons|simkl_ids_only|ids_only")
	fmt.Fprintln(w, "  --format                     csv|json|both (default csv)")
	fmt.Fprintln(w, "  --grouping                   single-file|separate-files (default separate-files)")
	fmt.Fprintln(w, "  --output                     Export directory override for this run")
	fmt.Fprintln(w, "  --filename-prefix            Filename prefix (default simkl-export)")
	fmt.Fprintln(w, "  --date-from                  Incremental export timestamp")
	fmt.Fprintln(w, "  --episode-files              Include episode files (default true)")
	fmt.Fprintln(w, "  --memos                      Include memos (default true)")
	fmt.Fprintln(w, "  --next-watch-info            Include next-watch info (default true)")
	fmt.Fprintln(w, "  --episode-watched-at         Include episode watched-at timestamps (default true)")
	fmt.Fprintln(w, "  --activity-check             Use activity check before export (default false)")
	fmt.Fprintln(w, "  --scheduled                  Run as scheduled backup and skip if backup is fresh")
	fmt.Fprintln(w, "  --max-backup-age             Override stale threshold: 12h, 24h, 3d, 1w")
}

func printConfigHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter config path")
	fmt.Fprintln(w, "  SimklExpoGter config show")
	fmt.Fprintln(w, "  SimklExpoGter config set [flags]")
}

func printConfigSetHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter config set --client-id <id> [--client-secret <secret>] [--output <dir>]")
	fmt.Fprintln(w, "  SimklExpoGter config set --backup-storage telegram --telegram-bot-token <token> --telegram-chat-id <id>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --client-id                  Persist the Simkl client ID")
	fmt.Fprintln(w, "  --client-secret, --secret    Persist the Simkl client secret")
	fmt.Fprintln(w, "  --output                     Persist the default export directory")
	fmt.Fprintln(w, "  --backup-storage             Persist backup storage: local, gdrive, telegram")
	fmt.Fprintln(w, "  --telegram-bot-token         Persist Telegram bot token")
	fmt.Fprintln(w, "  --telegram-chat-id           Persist Telegram chat ID")
	fmt.Fprintln(w, "  --telegram-thread-id         Persist Telegram forum topic/thread ID")
	fmt.Fprintln(w, "  --telegram-caption           Persist Telegram backup caption")
}

func printAuthHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter auth login")
	fmt.Fprintln(w, "  SimklExpoGter auth pin")
	fmt.Fprintln(w, "  SimklExpoGter auth status")
	fmt.Fprintln(w, "  SimklExpoGter auth clear")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  auth login uses the Simkl PIN flow and does not require client_secret or redirect_uri.")
	fmt.Fprintln(w, "  auth url is kept as a compatibility alias for auth login.")
	fmt.Fprintln(w, "  oauth-url and exchange are only for web apps with a registered redirect_uri.")
}

func printAuthPINHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter auth login [--no-poll]")
}

func printAuthExchangeHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter auth exchange --code <code>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "OAuth code exchange requires the redirect_uri registered in Simkl developer settings.")
}

func printScheduleHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter schedule status")
	fmt.Fprintln(w, "  SimklExpoGter schedule enable [flags]")
	fmt.Fprintln(w, "  SimklExpoGter schedule disable")
	fmt.Fprintln(w, "  SimklExpoGter schedule linger")
}

func printScheduleEnableHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter schedule enable [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --frequency                 daily|weekly")
	fmt.Fprintln(w, "  --time                      HH:MM in 24-hour format")
	fmt.Fprintln(w, "  --days                      Weekly days, comma-separated")
	fmt.Fprintln(w, "  --format                    csv|json|both")
	fmt.Fprintln(w, "  --field-mode                all|compact")
	fmt.Fprintln(w, "  --content                   shows,movies,anime")
	fmt.Fprintln(w, "  --activity-check            Use /sync/activities before export")
	fmt.Fprintln(w, "  --max-backup-age            Run only when last backup is older: 12h, 24h, 3d, 1w")
	fmt.Fprintln(w, "  --no-stale-guard            Always run when scheduler triggers")
}

func printTUIHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  SimklExpoGter tui")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Interactive terminal mode for guiless sessions.")
	fmt.Fprintln(w, "Keys: arrows to move, Enter to select/save, Tab to switch fields, Esc to go back, q or Ctrl+C to quit.")
}
