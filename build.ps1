param(
    [string]$Command = "windows",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Rest
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
& "$Root\build.bat" $Command @Rest
exit $LASTEXITCODE
