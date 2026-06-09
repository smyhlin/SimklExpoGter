//go:build windows

package scheduler

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type windowsManager struct{}

type queryResult struct {
	Installed  bool   `json:"installed"`
	TaskName   string `json:"taskName"`
	Status     string `json:"status"`
	NextRunAt  string `json:"nextRunAt"`
	LastRunAt  string `json:"lastRunAt"`
	LastResult string `json:"lastResult"`
}

func newManager() Manager {
	return windowsManager{}
}

func (windowsManager) Supported() bool {
	return true
}

func (windowsManager) Sync(config Config) (State, error) {
	triggerExpr := "$trigger = New-ScheduledTaskTrigger -Daily -At '" + psQuote(config.Time) + "'"
	if strings.EqualFold(config.Frequency, "weekly") {
		weeklyDays := make([]string, 0, len(config.Days))
		for _, day := range config.Days {
			weeklyDays = append(weeklyDays, dayToPowerShell(day))
		}
		triggerExpr = "$trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek " + strings.Join(weeklyDays, ",") + " -At '" + psQuote(config.Time) + "'"
	}

	script := strings.Join([]string{
		"$ErrorActionPreference = 'Stop'",
		"$action = New-ScheduledTaskAction -Execute '" + psQuote(config.ExecutablePath) + "' -Argument '" + psQuote(joinArgs(config.Arguments)) + "'",
		triggerExpr,
		"$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -StartWhenAvailable",
		"Register-ScheduledTask -TaskName '" + psQuote(config.TaskName) + "' -Description '" + psQuote(config.Description) + "' -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null",
	}, "; ")

	if _, err := runPowerShell(script); err != nil {
		return State{}, err
	}

	return windowsManager{}.Query(config.TaskName)
}

func (windowsManager) Remove(taskName string) error {
	script := "$task = Get-ScheduledTask -TaskName '" + psQuote(taskName) + "' -ErrorAction SilentlyContinue; if ($null -ne $task) { Unregister-ScheduledTask -TaskName '" + psQuote(taskName) + "' -Confirm:$false -ErrorAction Stop | Out-Null }"
	_, err := runPowerShell(script)
	return err
}

func (windowsManager) Query(taskName string) (State, error) {
	script := strings.Join([]string{
		"$task = Get-ScheduledTask -TaskName '" + psQuote(taskName) + "' -ErrorAction SilentlyContinue",
		"if ($null -eq $task) { [pscustomobject]@{ installed = $false; taskName = '" + psQuote(taskName) + "' } | ConvertTo-Json -Compress; exit 0 }",
		"$info = Get-ScheduledTaskInfo -TaskName '" + psQuote(taskName) + "'",
		"$next = ''; if ($info.NextRunTime -and $info.NextRunTime -ne [datetime]::MinValue) { $next = $info.NextRunTime.ToString('o') }",
		"$last = ''; if ($info.LastRunTime -and $info.LastRunTime -ne [datetime]::MinValue) { $last = $info.LastRunTime.ToString('o') }",
		"[pscustomobject]@{ installed = $true; taskName = $task.TaskName; status = [string]$task.State; nextRunAt = $next; lastRunAt = $last; lastResult = [string]$info.LastTaskResult } | ConvertTo-Json -Compress",
	}, "; ")

	output, err := runPowerShell(script)
	if err != nil {
		return State{}, err
	}

	var result queryResult
	if err := json.Unmarshal(output, &result); err != nil {
		return State{}, fmt.Errorf("failed to parse scheduled task status: %w", err)
	}

	return State{
		Supported:  true,
		Installed:  result.Installed,
		TaskName:   result.TaskName,
		Status:     result.Status,
		NextRunAt:  result.NextRunAt,
		LastRunAt:  result.LastRunAt,
		LastResult: result.LastResult,
	}, nil
}

func runPowerShell(script string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to manage Windows scheduled task: %s", strings.TrimSpace(string(output)))
	}
	return output, nil
}

func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func joinArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\"") {
			arg = strings.ReplaceAll(arg, `"`, `\"`)
			quoted = append(quoted, `"`+arg+`"`)
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}

func dayToPowerShell(day string) string {
	switch strings.ToLower(strings.TrimSpace(day)) {
	case "mon":
		return "Monday"
	case "tue":
		return "Tuesday"
	case "wed":
		return "Wednesday"
	case "thu":
		return "Thursday"
	case "fri":
		return "Friday"
	case "sat":
		return "Saturday"
	case "sun":
		return "Sunday"
	default:
		return "Monday"
	}
}
