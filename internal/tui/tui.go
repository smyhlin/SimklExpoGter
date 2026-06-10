package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"SimklExpoGter/internal/appsvc"
	"SimklExpoGter/internal/config"
	"SimklExpoGter/internal/exporter"
	"SimklExpoGter/internal/simkl"
)

// Service is the small application-service surface used by the terminal UI.
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
	ScheduleState() (appsvc.ScheduleState, error)
	SaveSchedule(appsvc.ScheduleSettingsInput) (config.Settings, appsvc.ScheduleState, error)
}

// Run launches a self-contained terminal UI without external dependencies.
// It is intentionally line-oriented so it works reliably over SSH, tmux,
// serial consoles and minimal Linux servers.
func Run(service Service) error {
	return run(service, os.Stdin, os.Stdout)
}

type menuItem struct {
	key         string
	title       string
	description string
	aliases     []string
}

var menuItems = []menuItem{
	{key: "1", title: "Status", description: "Config, auth, destination readiness", aliases: []string{"status"}},
	{key: "2", title: "Settings", description: "Client ID, optional secret, export path", aliases: []string{"settings"}},
	{key: "3", title: "Authenticate", description: "Simkl PIN login flow", aliases: []string{"auth", "authenticate", "login"}},
	{key: "4", title: "Easy export", description: "Run a full backup now", aliases: []string{"export", "backup"}},
	{key: "5", title: "Schedule status", description: "Show system timer and stale guard state", aliases: []string{"schedule status", "timer"}},
	{key: "6", title: "Configure recurring backup", description: "Daily/weekly timer, content, stale guard", aliases: []string{"schedule", "configure"}},
	{key: "7", title: "Clear Simkl token", description: "Remove saved Simkl access token", aliases: []string{"clear", "logout"}},
}

type screenState struct {
	output  io.Writer
	ansi    bool
	alt     bool
	left    bool
	restore func()
}

func run(service Service, input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	screen := newScreen(output)
	screen.enter()
	defer screen.leave()

	stopSignals := trapInterrupt(screen)
	defer stopSignals()

	statusLine := "Ready. Choose an action and press Enter."

	for {
		screen.renderMenu(statusLine)

		choice, err := prompt(reader, output, "Command")
		if err != nil {
			return err
		}

		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice == "" {
			continue
		}

		if choice == "q" || choice == "quit" || choice == "exit" {
			return nil
		}

		item, ok := matchMenuItem(choice)
		if !ok {
			statusLine = "Unknown command: " + choice
			continue
		}

		screen.renderAction(item.title)
		err = runMenuAction(item.key, service, reader, output)
		if err != nil {
			fmt.Fprintln(output)
			fmt.Fprintln(output, "Error:", err)
			statusLine = item.title + " failed: " + err.Error()
		} else {
			statusLine = item.title + " completed."
		}

		if err := waitForEnter(reader, output, screen); err != nil {
			return err
		}
	}
}

func newScreen(output io.Writer) *screenState {
	file, ok := output.(*os.File)
	if !ok {
		return &screenState{output: output}
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return &screenState{output: output}
	}

	restore, ansi := setupTerminalOutput(file)
	return &screenState{
		output:  output,
		ansi:    ansi,
		alt:     ansi,
		restore: restore,
	}
}

func (s *screenState) enter() {
	if !s.alt || s.left {
		return
	}
	fmt.Fprint(s.output, "\x1b[?1049h\x1b[?25h")
	s.left = false
}

func (s *screenState) leave() {
	if s.alt && !s.left {
		fmt.Fprint(s.output, "\x1b[0m\x1b[?25h\x1b[?1049l")
		s.left = true
	}
	if s.restore != nil {
		s.restore()
		s.restore = nil
	}
}

func (s *screenState) clear() {
	if s.alt {
		fmt.Fprint(s.output, "\x1b[2J\x1b[H")
	}
}

func (s *screenState) renderMenu(statusLine string) {
	s.clear()
	fmt.Fprintf(s.output, "%s  %s\n", s.bold("SimklExpoGter"), s.dim("TUI workspace · type number/name · q quits"))
	fmt.Fprintln(s.output, "──────────────────────────────────────────────────────────────────────────────")
	fmt.Fprintln(s.output)
	fmt.Fprintln(s.output, s.bold("Actions"))
	for _, item := range menuItems {
		fmt.Fprintf(s.output, "  %s  %-28s %s\n", s.cyan(item.key), item.title, s.dim(item.description))
	}
	fmt.Fprintln(s.output)
	fmt.Fprintln(s.output, "──────────────────────────────────────────────────────────────────────────────")
	if strings.TrimSpace(statusLine) != "" {
		fmt.Fprintln(s.output, s.dim(statusLine))
	}
	fmt.Fprintln(s.output)
}

func (s *screenState) renderAction(title string) {
	s.clear()
	fmt.Fprintln(s.output, s.bold(title))
	fmt.Fprintln(s.output, "──────────────────────────────────────────────────────────────────────────────")
	fmt.Fprintln(s.output)
}

func (s *screenState) bold(value string) string {
	return s.wrap(value, "1")
}

func (s *screenState) dim(value string) string {
	return s.wrap(value, "2")
}

func (s *screenState) cyan(value string) string {
	return s.wrap(value, "36")
}

func (s *screenState) wrap(value string, code string) string {
	if !s.ansi {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func trapInterrupt(screen *screenState) func() {
	if !screen.alt {
		return func() {}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		screen.leave()
		os.Exit(130)
	}()

	return func() {
		signal.Stop(signals)
	}
}

func matchMenuItem(choice string) (menuItem, bool) {
	for _, item := range menuItems {
		if choice == item.key || choice == strings.ToLower(item.title) {
			return item, true
		}
		for _, alias := range item.aliases {
			if choice == alias {
				return item, true
			}
		}
	}
	return menuItem{}, false
}

func runMenuAction(key string, service Service, reader *bufio.Reader, output io.Writer) error {
	switch key {
	case "1":
		return showStatus(service, output)
	case "2":
		return editSettings(service, reader, output)
	case "3":
		return authenticate(service, reader, output)
	case "4":
		return runEasyExport(service, output)
	case "5":
		return showScheduleStatus(service, output)
	case "6":
		return configureSchedule(service, reader, output)
	case "7":
		if _, err := service.ClearAccessToken(); err != nil {
			return err
		}
		fmt.Fprintln(output, "Access token cleared.")
		return nil
	default:
		fmt.Fprintln(output, "Unknown option.")
		return nil
	}
}

func waitForEnter(reader *bufio.Reader, output io.Writer, screen *screenState) error {
	fmt.Fprintln(output)
	fmt.Fprint(output, screen.dim("Press Enter to return to the dashboard..."))
	_, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	return nil
}

func showStatus(service Service, output io.Writer) error {
	configSummary, err := service.ConfigSummary()
	if err != nil {
		return err
	}
	authSummary, err := service.AuthSummary()
	if err != nil {
		return err
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Config:", configSummary.ConfigPath)
	fmt.Fprintln(output, "Client ID:", boolLabel(authSummary.ClientIDConfigured))
	fmt.Fprintln(output, "Client secret:", boolLabel(authSummary.ClientSecretConfigured))
	fmt.Fprintln(output, "Access token:", boolLabel(authSummary.AccessTokenConfigured))
	fmt.Fprintln(output, "Headless export ready:", boolLabel(authSummary.ReadyForHeadlessRun))
	fmt.Fprintln(output, "Destination:", firstNonEmpty(configSummary.ExportDirectory, configSummary.SuggestedExportDirectory))
	return nil
}

func editSettings(service Service, reader *bufio.Reader, output io.Writer) error {
	current, err := service.ConfigSummary()
	if err != nil {
		return err
	}

	clientID, err := promptDefault(reader, output, "Simkl client ID", current.ClientID)
	if err != nil {
		return err
	}
	clientSecret, err := prompt(reader, output, "Simkl client secret (leave blank to keep current)")
	if err != nil {
		return err
	}
	exportDir, err := promptDefault(reader, output, "Export directory", firstNonEmpty(current.ExportDirectory, current.SuggestedExportDirectory))
	if err != nil {
		return err
	}

	_, err = service.SaveSettings(appsvc.SaveSettingsInput{
		ClientID:           clientID,
		ClientSecret:       clientSecret,
		ExportDirectory:    exportDir,
		SetClientID:        true,
		SetClientSecret:    strings.TrimSpace(clientSecret) != "",
		SetExportDirectory: true,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(output, "Settings saved.")
	return nil
}

func authenticate(service Service, reader *bufio.Reader, output io.Writer) error {
	deviceCode, err := service.RequestDeviceCode()
	if err != nil {
		return err
	}

	verificationURL := strings.TrimSpace(deviceCode.VerificationURL)
	if verificationURL == "" {
		verificationURL = "https://simkl.com/pin/"
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Simkl PIN login")
	fmt.Fprintln(output, "Open this URL in your browser:")
	fmt.Fprintln(output, verificationURL)
	fmt.Fprintln(output, "Enter this code:")
	fmt.Fprintln(output, strings.TrimSpace(deviceCode.UserCode))

	answer, err := promptDefault(reader, output, "Poll until approved? [Y/n]", "Y")
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(answer), "n") || strings.EqualFold(strings.TrimSpace(answer), "no") {
		fmt.Fprintln(output, "Login left pending. Run Authenticate again and approve the displayed PIN.")
		return nil
	}

	interval := deviceCode.Interval
	if interval <= 0 {
		interval = 5
	}
	expiresIn := deviceCode.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 900
	}

	fmt.Fprintf(output, "Waiting for approval for up to %d seconds", expiresIn)
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			fmt.Fprintln(output)
			fmt.Fprintln(output, "Simkl PIN login expired. Restart authentication and approve the new code.")
			return nil
		}

		time.Sleep(time.Duration(interval) * time.Second)
		status, err := service.PollDeviceCode(deviceCode.UserCode)
		if err != nil {
			fmt.Fprintln(output)
			return err
		}

		if strings.EqualFold(strings.TrimSpace(status.Result), "OK") && strings.TrimSpace(status.AccessToken) != "" {
			if _, err := service.SaveAccessToken(status.AccessToken); err != nil {
				fmt.Fprintln(output)
				return err
			}
			fmt.Fprintln(output)
			fmt.Fprintln(output, "Access token saved.")
			return nil
		}

		message := strings.ToLower(strings.TrimSpace(status.Message))
		if strings.Contains(message, "slow") {
			interval += 5
		}
		fmt.Fprint(output, ".")
	}
}

func runEasyExport(service Service, output io.Writer) error {
	fmt.Fprintln(output, "Running export...")
	result, err := service.RunExport(exporter.Request{})
	if err != nil {
		return err
	}
	printExportResult(output, result)
	return nil
}

func showScheduleStatus(service Service, output io.Writer) error {
	state, err := service.ScheduleState()
	if err != nil {
		return err
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Scheduler supported:", boolLabel(state.Supported))
	fmt.Fprintln(output, "Enabled:", boolLabel(state.Enabled))
	fmt.Fprintln(output, "Installed:", boolLabel(state.Installed))
	fmt.Fprintln(output, "Frequency:", state.Frequency)
	fmt.Fprintln(output, "Time:", state.Time)
	fmt.Fprintln(output, "Max backup age:", firstNonEmpty(state.MaxBackupAge, "24h"))
	fmt.Fprintln(output, "Stale guard:", boolLabel(state.RunIfBackupIsStale))
	if state.LastSuccessfulBackupAt != "" {
		fmt.Fprintln(output, "Last successful backup:", state.LastSuccessfulBackupAt, state.LastSuccessfulBackupKind)
	}
	if state.Message != "" {
		fmt.Fprintln(output, "Message:", state.Message)
	}
	return nil
}

func configureSchedule(service Service, reader *bufio.Reader, output io.Writer) error {
	state, err := service.ScheduleState()
	if err != nil {
		return err
	}

	enableValue, err := promptDefault(reader, output, "Enable recurring backup? (y/n)", yesNo(state.Enabled))
	if err != nil {
		return err
	}
	enabled := parseYes(enableValue)
	if !enabled {
		if _, _, err := service.SaveSchedule(appsvc.ScheduleSettingsInput{Enabled: false}); err != nil {
			return err
		}
		fmt.Fprintln(output, "Recurring backup disabled.")
		return nil
	}

	frequency, err := promptDefault(reader, output, "Frequency (daily/weekly)", firstNonEmpty(state.Frequency, "daily"))
	if err != nil {
		return err
	}
	runTime, err := promptDefault(reader, output, "Run time (HH:MM)", firstNonEmpty(state.Time, "02:00"))
	if err != nil {
		return err
	}
	days, err := promptDefault(reader, output, "Weekly days (mon,tue,wed)", strings.Join(state.Days, ","))
	if err != nil {
		return err
	}
	format, err := promptDefault(reader, output, "Output format (csv/json/both)", firstNonEmpty(state.OutputFormat, exporter.FormatCSV))
	if err != nil {
		return err
	}
	fieldMode, err := promptDefault(reader, output, "Field mode (all/compact)", firstNonEmpty(state.FieldMode, exporter.FieldModeAll))
	if err != nil {
		return err
	}
	content, err := promptDefault(reader, output, "Content (shows,movies,anime)", strings.Join(state.Content, ","))
	if err != nil {
		return err
	}
	activityValue, err := promptDefault(reader, output, "Use activity check? (y/n)", yesNo(state.UseActivityCheck))
	if err != nil {
		return err
	}
	staleDefault := yesNo(state.RunIfBackupIsStale)
	if !state.Enabled && strings.TrimSpace(state.MaxBackupAge) == "" {
		staleDefault = "y"
	}
	staleValue, err := promptDefault(reader, output, "Use stale backup guard? (y/n)", staleDefault)
	if err != nil {
		return err
	}
	maxAge, err := promptDefault(reader, output, "Maximum backup age (12h,24h,3d,1w)", firstNonEmpty(state.MaxBackupAge, "24h"))
	if err != nil {
		return err
	}

	_, nextState, err := service.SaveSchedule(appsvc.ScheduleSettingsInput{
		Enabled:            true,
		Frequency:          frequency,
		Time:               runTime,
		Days:               splitCSV(days),
		OutputFormat:       format,
		FieldMode:          fieldMode,
		Content:            splitCSV(content),
		UseActivityCheck:   parseYes(activityValue),
		MaxBackupAge:       maxAge,
		RunIfBackupIsStale: parseYes(staleValue),
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(output, "Recurring backup saved.")
	if nextState.Message != "" {
		fmt.Fprintln(output, nextState.Message)
	}
	return nil
}

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	fmt.Fprintf(output, "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptDefault(reader *bufio.Reader, output io.Writer, label, fallback string) (string, error) {
	if strings.TrimSpace(fallback) != "" {
		fmt.Fprintf(output, "%s [%s]: ", label, fallback)
	} else {
		fmt.Fprintf(output, "%s: ", label)
	}
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(fallback), nil
	}
	return value, nil
}

func printExportResult(output io.Writer, result exporter.Result) {
	fmt.Fprintf(output, "Exported %d items to %s\n", result.ItemCounts["all"], firstNonEmpty(result.DestinationLabel, result.OutputDirectory))
	for _, file := range result.Files {
		fmt.Fprintf(output, "- %s (%s/%s/%s, %d rows)\n", filepathBase(file.Path), file.Format, file.MediaType, file.Kind, file.Rows)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(output, "Warning: %s\n", warning)
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
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func yesNo(value bool) string {
	if value {
		return "y"
	}
	return "n"
}

func parseYes(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1", "on", "enabled":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func filepathBase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "/")
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}
