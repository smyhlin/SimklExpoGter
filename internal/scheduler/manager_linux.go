//go:build linux

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	userpkg "os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type linuxManager struct{}

func newManager() Manager {
	return linuxManager{}
}

func (linuxManager) Supported() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func (linuxManager) Sync(config Config) (State, error) {
	if !(linuxManager{}).Supported() {
		return State{Supported: false, TaskName: config.TaskName}, fmt.Errorf("systemd user timers are not available on this system")
	}
	if strings.TrimSpace(config.ExecutablePath) == "" {
		return State{}, fmt.Errorf("missing executable path for scheduled backup")
	}

	unitName := sanitizedUnitName(config.TaskName)
	userDir, err := userSystemdDir()
	if err != nil {
		return State{}, err
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return State{}, err
	}

	servicePath := filepath.Join(userDir, unitName+".service")
	timerPath := filepath.Join(userDir, unitName+".timer")
	wrapperPath := filepath.Join(userDir, unitName+"-run.sh")

	serviceUnit := buildServiceUnit(config, wrapperPath)
	timerUnitColdStart := buildTimerUnit(config, false)
	timerUnitPersistent := buildTimerUnit(config, true)
	wrapperScript := buildWrapperScript(config)

	// Stop the previous timer first. This keeps reconfiguration idempotent and
	// avoids a stale active unit from racing while the files are rewritten.
	_, _ = runSystemctl("stop", unitName+".timer")

	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0o700); err != nil {
		return State{}, err
	}
	if err := os.WriteFile(servicePath, []byte(serviceUnit), 0o644); err != nil {
		return State{}, err
	}

	// First activation deliberately starts with Persistent=false. A persistent
	// calendar timer can immediately trigger a catch-up job when the timer is
	// started. That is useful after real downtime, but surprising during setup:
	// a failed backup can make "schedule enable" look like the timer itself
	// failed. After the timer is active, the file is rewritten with
	// Persistent=true so future missed runs are still caught.
	if err := os.WriteFile(timerPath, []byte(timerUnitColdStart), 0o644); err != nil {
		return State{}, err
	}
	if _, err := runSystemctl("daemon-reload"); err != nil {
		return State{}, err
	}
	if _, err := runSystemctl("enable", unitName+".timer"); err != nil {
		return State{}, err
	}
	if _, err := runSystemctl("start", unitName+".timer"); err != nil {
		return State{}, decorateStartError(unitName, err)
	}

	if err := os.WriteFile(timerPath, []byte(timerUnitPersistent), 0o644); err != nil {
		return State{}, err
	}
	if _, err := runSystemctl("daemon-reload"); err != nil {
		return State{}, err
	}

	state, err := linuxManager{}.Query(config.TaskName)
	if err != nil {
		return state, err
	}
	if lingerMessage := ensureCurrentUserLinger(); lingerMessage != "" {
		if state.Message != "" {
			state.Message += " "
		}
		state.Message += lingerMessage
	}
	return state, nil
}

func (linuxManager) Remove(taskName string) error {
	if !(linuxManager{}).Supported() {
		return nil
	}
	unitName := sanitizedUnitName(taskName)
	_, _ = runSystemctl("disable", "--now", unitName+".timer")
	userDir, err := userSystemdDir()
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(userDir, unitName+".timer"))
	_ = os.Remove(filepath.Join(userDir, unitName+".service"))
	_ = os.Remove(filepath.Join(userDir, unitName+"-run.sh"))
	_, _ = runSystemctl("daemon-reload")
	return nil
}

func (linuxManager) Query(taskName string) (State, error) {
	state := State{Supported: (linuxManager{}).Supported(), TaskName: taskName}
	if !state.Supported {
		state.Message = "systemd user timers are not available on this system"
		return state, nil
	}

	unitName := sanitizedUnitName(taskName)
	timerName := unitName + ".timer"
	serviceName := unitName + ".service"

	if _, err := runSystemctl("is-enabled", "--quiet", timerName); err != nil {
		state.Installed = false
		state.Message = "Timer is not installed."
		return state, nil
	}
	state.Installed = true

	timerProps, err := systemctlShow(timerName, "ActiveState", "NextElapseUSecRealtime", "LastTriggerUSec")
	if err != nil {
		return state, err
	}
	serviceProps, _ := systemctlShow(serviceName, "ActiveState", "Result")

	state.Status = timerProps["ActiveState"]
	state.NextRunAt = timerProps["NextElapseUSecRealtime"]
	state.LastRunAt = timerProps["LastTriggerUSec"]
	state.LastResult = serviceProps["Result"]
	if serviceProps["ActiveState"] != "" && state.LastResult == "" {
		state.LastResult = serviceProps["ActiveState"]
	}
	state.Message = "Recurring backup is synced with a systemd user timer."
	return state, nil
}

func buildServiceUnit(config Config, wrapperPath string) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=" + firstNonEmpty(config.Description, "Recurring backup for "+config.TaskName),
		"Wants=network-online.target",
		"After=network-online.target",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=" + systemdPath(wrapperPath),
		"",
	}, "\n")
}

func buildWrapperScript(config Config) string {
	args := append([]string{config.ExecutablePath}, config.Arguments...)
	quotedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		quotedArgs = append(quotedArgs, shellQuote(arg))
	}

	return strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"exec " + strings.Join(quotedArgs, " "),
		"",
	}, "\n")
}

func buildTimerUnit(config Config, persistent bool) string {
	lines := []string{
		"[Unit]",
		"Description=" + firstNonEmpty(config.Description, "Recurring backup for "+config.TaskName),
		"",
		"[Timer]",
	}
	for _, calendar := range onCalendarLines(config) {
		lines = append(lines, "OnCalendar="+calendar)
	}
	lines = append(lines,
		"Persistent="+strconv.FormatBool(persistent),
		"Unit="+sanitizedUnitName(config.TaskName)+".service",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	)
	return strings.Join(lines, "\n")
}

func onCalendarLines(config Config) []string {
	timeValue := strings.TrimSpace(config.Time)
	if timeValue == "" {
		timeValue = "02:00"
	}
	if len(timeValue) == len("15:04") {
		timeValue += ":00"
	}

	if strings.EqualFold(config.Frequency, "weekly") {
		days := append([]string(nil), config.Days...)
		if len(days) == 0 {
			days = []string{"mon"}
		}
		sort.Strings(days)
		calendars := make([]string, 0, len(days))
		for _, day := range days {
			calendars = append(calendars, systemdDay(day)+" *-*-* "+timeValue)
		}
		return calendars
	}

	return []string{"*-*-* " + timeValue}
}

func systemdDay(day string) string {
	switch strings.ToLower(strings.TrimSpace(day)) {
	case "mon":
		return "Mon"
	case "tue":
		return "Tue"
	case "wed":
		return "Wed"
	case "thu":
		return "Thu"
	case "fri":
		return "Fri"
	case "sat":
		return "Sat"
	case "sun":
		return "Sun"
	default:
		return "Mon"
	}
}

func userSystemdDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "systemd", "user"), nil
}

func sanitizedUnitName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "SimklExpoGterRecurringBackup"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '@' || r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-.")
}

func systemdPath(value string) string {
	value = filepath.Clean(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		`\`, `\\`,
		" ", `\x20`,
		"\t", `\t`,
		"\n", "",
	)
	return replacer.Replace(value)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func decorateStartError(unitName string, startErr error) error {
	timerName := unitName + ".timer"
	serviceName := unitName + ".service"
	details := []string{startErr.Error()}

	if output, err := runSystemctl("status", timerName, "--no-pager", "--full"); err == nil && strings.TrimSpace(string(output)) != "" {
		details = append(details, "timer status:\n"+strings.TrimSpace(string(output)))
	}
	if output, err := runSystemctl("status", serviceName, "--no-pager", "--full"); err == nil && strings.TrimSpace(string(output)) != "" {
		details = append(details, "service status:\n"+strings.TrimSpace(string(output)))
	}

	return fmt.Errorf("%s", strings.Join(details, "\n\n"))
}

func ensureCurrentUserLinger() string {
	if _, err := exec.LookPath("loginctl"); err != nil {
		return "loginctl is not available, so user lingering could not be enabled automatically."
	}

	userName := currentUserName()
	if userName == "" {
		return "Could not determine the current user for loginctl enable-linger."
	}

	if isUserLingerEnabled(userName) {
		return "User lingering is enabled, so the timer can run after logout and boot."
	}

	cmd := exec.Command("loginctl", "enable-linger", userName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "Timer is installed, but automatic loginctl enable-linger failed: " + message + ". Run: loginctl enable-linger \"$USER\""
	}

	if isUserLingerEnabled(userName) {
		return "User lingering was enabled automatically, so the timer can run after logout and boot."
	}

	return "loginctl enable-linger was run, but lingering could not be verified. Check with: loginctl show-user \"$USER\" -p Linger"
}

func isUserLingerEnabled(userName string) bool {
	output, err := exec.Command("loginctl", "show-user", userName, "-p", "Linger", "--value").CombinedOutput()
	if err != nil {
		output, err = exec.Command("loginctl", "show-user", userName, "-p", "Linger").CombinedOutput()
		if err != nil {
			return false
		}
	}
	value := strings.TrimSpace(string(output))
	return strings.EqualFold(value, "yes") || strings.EqualFold(value, "Linger=yes")
}

func currentUserName() string {
	if userName := strings.TrimSpace(os.Getenv("USER")); userName != "" {
		return userName
	}
	if current, err := userpkg.Current(); err == nil {
		if userName := strings.TrimSpace(current.Username); userName != "" {
			if slash := strings.LastIndexAny(userName, `\\/`); slash >= 0 && slash+1 < len(userName) {
				return userName[slash+1:]
			}
			return userName
		}
	}
	return ""
}

func systemctlShow(unit string, properties ...string) (map[string]string, error) {
	args := []string{"show", unit}
	for _, property := range properties {
		args = append(args, "-p", property)
	}
	output, err := runSystemctl(args...)
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = value
		}
	}
	return values, nil
}

func runSystemctl(args ...string) ([]byte, error) {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl --user %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return output, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
