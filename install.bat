@echo off
REM RedPaper Installation Script

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
)

REM Copy executable
echo Copying executable...
copy "redpaper.exe" "%INSTALL_DIR%\" >nul
if errorlevel 1 (
    echo Error copying executable
    pause
    exit /b 1
)

REM Copy uninstall script
copy "uninstall.bat" "%INSTALL_DIR%\" >nul

REM Remove any old scheduled task from previous installs
schtasks /delete /tn "RedPaper" /f >nul 2>&1

REM Create registry entries for Add/Remove Programs
echo Creating registry entries...
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v DisplayName /t REG_SZ /d "RedPaper" /f >nul
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v UninstallString /t REG_SZ /d "%INSTALL_DIR%\uninstall.bat" /f >nul
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v DisplayVersion /t REG_SZ /d "1.0.0" /f >nul
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /v Publisher /t REG_SZ /d "RedPaper" /f >nul

REM Create desktop shortcut
echo Creating desktop shortcut...
set SHORTCUT_PATH=%USERPROFILE%\Desktop\RedPaper.lnk
powershell.exe -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%SHORTCUT_PATH%'); $Shortcut.TargetPath = '%INSTALL_DIR%\redpaper.exe'; $Shortcut.IconLocation = '%INSTALL_DIR%\redpaper.exe,0'; $Shortcut.Save()"

REM Launch RedPaper (it will appear in the system tray)
echo Launching RedPaper...
start "" "%INSTALL_DIR%\redpaper.exe"

echo.
echo Installation completed successfully!
echo.
echo RedPaper is now running in the system tray (bottom-right of taskbar).
echo Right-click the tray icon to change settings and enable "Start with Windows".
echo.
pause
