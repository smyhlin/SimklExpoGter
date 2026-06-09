@echo off
setlocal enabledelayedexpansion
set APP_NAME=SimklExpoGter
set VERSION=%VERSION%
if "%VERSION%"=="" set VERSION=0.1.0
set DIST_DIR=%~dp0dist

if "%~1"=="" goto help
if /I "%~1"=="help" goto help
if /I "%~1"=="-h" goto help
if /I "%~1"=="--help" goto help
if /I "%~1"=="bootstrap" goto bootstrap
if /I "%~1"=="check" goto check
if /I "%~1"=="test" goto check
if /I "%~1"=="frontend" goto frontend
if /I "%~1"=="windows" goto windows
if /I "%~1"=="windows-cli" goto windows_cli
if /I "%~1"=="release" goto release_windows
if /I "%~1"=="release-windows" goto release_windows
if /I "%~1"=="clean" goto clean

echo Unknown command: %~1
exit /b 2

:bootstrap
shift /1
call "%~dp0scripts\bootstrap.bat" %*
exit /b %ERRORLEVEL%

:frontend
pushd "%~dp0frontend"
call npm ci || exit /b 1
call npm run build || exit /b 1
popd
exit /b 0

:check
call "%~f0" frontend || exit /b 1
go test ./... || exit /b 1
exit /b 0

:resolve_wails
where wails >nul 2>nul
if errorlevel 1 (
  for /f "delims=" %%G in ('go env GOPATH') do set GOPATH_VALUE=%%G
  set WAILS_BIN=!GOPATH_VALUE!\bin\wails.exe
) else (
  set WAILS_BIN=wails
)
exit /b 0

:windows
call "%~f0" frontend || exit /b 1
call :resolve_wails || exit /b 1
!WAILS_BIN! build -clean -platform windows/amd64 || exit /b 1
if not exist "%DIST_DIR%\windows" mkdir "%DIST_DIR%\windows"
copy /Y "%~dp0build\bin\%APP_NAME%.exe" "%DIST_DIR%\windows\%APP_NAME%.exe" >nul
exit /b 0

:windows_cli
if not exist "%~dp0frontend\dist" mkdir "%~dp0frontend\dist"
if not exist "%~dp0frontend\dist\.gitkeep" type nul > "%~dp0frontend\dist\.gitkeep"
if not exist "%DIST_DIR%\windows" mkdir "%DIST_DIR%\windows"
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -tags cli -trimpath -ldflags "-s -w" -o "%DIST_DIR%\windows\%APP_NAME%-cli.exe" ./ || exit /b 1
exit /b 0

:release_windows
call "%~f0" windows || exit /b 1
call "%~f0" windows-cli || exit /b 1

if not exist "%DIST_DIR%\release" mkdir "%DIST_DIR%\release"

powershell -NoProfile -ExecutionPolicy Bypass -Command "Compress-Archive -Force -Path '%DIST_DIR%\windows\%APP_NAME%.exe' -DestinationPath '%DIST_DIR%\release\%APP_NAME%-windows-gui-amd64-v%VERSION%.zip'" || exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -Command "Compress-Archive -Force -Path '%DIST_DIR%\windows\%APP_NAME%-cli.exe' -DestinationPath '%DIST_DIR%\release\%APP_NAME%-windows-cli-amd64-v%VERSION%.zip'" || exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -Command "Get-FileHash '%DIST_DIR%\release\*' -Algorithm SHA256 | ForEach-Object { $_.Hash + '  ' + (Split-Path $_.Path -Leaf) } | Set-Content '%DIST_DIR%\release\SHA256SUMS-windows.txt'" || exit /b 1

dir "%DIST_DIR%\release"
exit /b 0

:clean
rmdir /S /Q "%DIST_DIR%" 2>nul
rmdir /S /Q "%~dp0build\bin" 2>nul
rmdir /S /Q "%~dp0frontend\dist" 2>nul
rmdir /S /Q "%~dp0.build" 2>nul
exit /b 0

:help
echo Usage: build.bat ^<command^>
echo.
echo Commands:
echo   bootstrap        Run Windows bootstrap
echo   check            Build frontend and run Go tests
echo   frontend         Build frontend only
echo   windows          Build Windows GUI binary with Wails
echo   windows-cli      Build Windows CLI/TUI-only binary
echo   release-windows  Build Windows release zip files
echo   clean            Remove generated outputs
exit /b 0
