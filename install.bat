@echo off
REM RedPaper Installation Script
REM This script installs the wallpaper changer and sets up the scheduled task

echo RedPaper Installation
echo =====================

REM Check if running as administrator
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Administrator privileges detected.
) else (
    echo This script requires administrator privileges.
    echo Please run as administrator.
    pause
    exit /b 1
)

REM Set installation directory
set INSTALL_DIR=%PROGRAMFILES%\RedPaper

echo Installing to: %INSTALL_DIR%

REM Create installation directory
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
    mkdir "%INSTALL_DIR%\logs"
    mkdir "%INSTALL_DIR%\data"
)

REM Copy executable
echo Copying executable...
copy "redpaper.exe" "%INSTALL_DIR%\" >nul
if errorlevel 1 (
    echo Error copying executable
    pause
    exit /b 1
)

REM Create batch file for scheduled task
echo Creating run script...
(
echo @echo off
echo "%INSTALL_DIR%\redpaper.exe" ^>^> "%INSTALL_DIR%\logs\redpaper.log" 2^>^&1
echo exit /b %%errorlevel%%
) > "%INSTALL_DIR%\run_redpaper.bat"

REM Create PowerShell script for task creation
echo Creating task setup script...
(
echo $taskName = "RedPaper"
echo $taskPath = "%INSTALL_DIR%\run_redpaper.bat"
echo.
echo # Remove existing task if it exists
echo if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
echo     Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
echo     Write-Host "Removed existing task: $taskName"
echo }
echo.
echo # Create new scheduled task
echo $action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument "/c `"$taskPath`""
echo.
echo # Create triggers separately (cannot combine with +=)
echo $triggerStartup = New-ScheduledTaskTrigger -AtStartup
echo $triggerHourly = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Hours 1) -RepetitionDuration (New-TimeSpan -Days 365)
echo.
echo # Use correct LogonType
echo $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive
echo $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
echo.
echo # Create the task with multiple triggers
echo $task = New-ScheduledTask -Action $action -Trigger @($triggerStartup, $triggerHourly) -Principal $principal -Settings $settings
echo Register-ScheduledTask -TaskName $taskName -InputObject $task
echo Write-Host "Successfully created scheduled task: $taskName"
) > "%INSTALL_DIR%\create_task.ps1"

REM Execute PowerShell script to create task
echo Setting up scheduled task...
powershell.exe -ExecutionPolicy Bypass -File "%INSTALL_DIR%\create_task.ps1"
if errorlevel 1 (
    echo PowerShell method failed, trying schtasks fallback...
    schtasks /create /tn "RedPaper" /tr "%INSTALL_DIR%\run_redpaper.bat" /sc hourly /f
    if errorlevel 1 (
        echo Error creating scheduled task with both methods
        echo You can create it manually using Task Scheduler
    ) else (
        echo Scheduled task created successfully using schtasks
    )
) else (
    echo Scheduled task created successfully using PowerShell
)

REM Create desktop shortcut
echo Creating desktop shortcut...
set SCRIPT_DIR=%~dp0
set SHORTCUT_PATH=%USERPROFILE%\Desktop\RedPaper.lnk
powershell.exe -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%SHORTCUT_PATH%'); $Shortcut.TargetPath = '%INSTALL_DIR%\redpaper.exe'; $Shortcut.Save()"

REM Create registry entries for Add/Remove Programs
echo Creating registry entries...
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v DisplayName /t REG_SZ /d "RedPaper" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v UninstallString /t REG_SZ /d "%INSTALL_DIR%\Uninstall.bat" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v DisplayVersion /t REG_SZ /d "1.0.0" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v Publisher /t REG_SZ /d "Your Company" /f

REM Copy uninstall script
copy "uninstall.bat" "%INSTALL_DIR%\" >nul

echo.
echo Installation completed successfully!
echo.
echo The wallpaper changer has been installed and a scheduled task has been created.
echo The task will run every hour and check if 24 hours have passed since the last wallpaper change.
echo.
echo You can also run it manually from the desktop shortcut.
echo.
pause
