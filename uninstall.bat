@echo off
REM RedPaper Uninstallation Script

echo RedPaper Uninstallation
echo =======================

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

echo Removing from: %INSTALL_DIR%

REM Stop running instance
echo Stopping RedPaper if running...
taskkill /f /im redpaper.exe >nul 2>&1

REM Remove startup registry key
echo Removing startup entry...
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "RedPaper" /f >nul 2>&1

REM Remove old scheduled task if it exists
schtasks /delete /tn "RedPaper" /f >nul 2>&1

REM Ask user about removing data
echo.
set /p choice="Do you want to remove all downloaded wallpapers and settings? (Y/N): "
if /i "%choice%"=="Y" goto :removeData
if /i "%choice%"=="y" goto :removeData
goto :skipData

:removeData
echo Removing user data...
if exist "%LOCALAPPDATA%\RedPaper" (
    rmdir /s /q "%LOCALAPPDATA%\RedPaper"
)
if exist "%USERPROFILE%\Pictures\redpaper_wallpapers" (
    rmdir /s /q "%USERPROFILE%\Pictures\redpaper_wallpapers"
)
echo User data removed.
goto :continue

:skipData
echo Keeping user data.

:continue
REM Remove files
echo Removing application files...
if exist "%INSTALL_DIR%" (
    rmdir /s /q "%INSTALL_DIR%"
)

REM Remove desktop shortcut
echo Removing desktop shortcut...
if exist "%USERPROFILE%\Desktop\RedPaper.lnk" (
    del "%USERPROFILE%\Desktop\RedPaper.lnk"
)

REM Remove registry entries
echo Removing registry entries...
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" /f >nul 2>&1

echo.
echo Uninstallation completed successfully!
echo.
echo RedPaper has been removed from your system.
echo.
pause
