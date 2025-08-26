@echo off
REM RedPaper Build Script
REM This script builds the Go application and creates the Windows installer

echo RedPaper Build Script
echo =====================

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

REM Set version and output paths
set VERSION=1.0.0
set OUTPUT_DIR=build
set EXE_NAME=redpaper.exe
set INSTALLER_NAME=RedPaper_Installer.exe

REM Create output directory
if not exist "%OUTPUT_DIR%" (
    mkdir "%OUTPUT_DIR%"
)

echo Building Go application...
echo.

REM Build the Go application for Windows
go build -ldflags "-s -w -X main.version=%VERSION%" -o "%OUTPUT_DIR%\%EXE_NAME%" .
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Go application built successfully: %OUTPUT_DIR%\%EXE_NAME%
echo.

REM Check if NSIS is available for installer creation
makensis /VERSION >nul 2>&1
if %errorlevel% neq 0 (
    echo NSIS not found. Skipping installer creation.
    echo You can install NSIS from https://nsis.sourceforge.io/
    echo.
    echo Build completed without installer.
    goto :createZip
)

REM Copy files to build directory for installer
echo Preparing files for installer...
copy "license.txt" "%OUTPUT_DIR%\" >nul
copy "installer.nsi" "%OUTPUT_DIR%\" >nul
if exist "installer.ico" copy "installer.ico" "%OUTPUT_DIR%\" >nul
if exist "header.bmp" copy "header.bmp" "%OUTPUT_DIR%\" >nul
if exist "wizard.bmp" copy "wizard.bmp" "%OUTPUT_DIR%\" >nul

REM Create installer
echo Creating Windows installer...
cd "%OUTPUT_DIR%"
makensis installer.nsi
if %errorlevel% neq 0 (
    echo Installer creation failed!
    cd ..
    pause
    exit /b 1
)

cd ..
echo Windows installer created: %OUTPUT_DIR%\%INSTALLER_NAME%
echo.

:createZip
REM Create a ZIP file with all files
echo Creating ZIP package...
powershell.exe -Command "Compress-Archive -Path '%OUTPUT_DIR%\*' -DestinationPath '%OUTPUT_DIR%\RedPaper_v%VERSION%.zip' -Force"

echo.
echo Build completed successfully!
echo.
echo Generated files:
echo - %OUTPUT_DIR%\%EXE_NAME% (Standalone executable)
echo - %OUTPUT_DIR%\%INSTALLER_NAME% (Windows installer)
echo - %OUTPUT_DIR%\RedPaper_v%VERSION%.zip (ZIP package)
echo.
echo Installation options:
echo 1. Use the Windows installer (%INSTALLER_NAME%) for full installation with scheduled task
echo 2. Use the standalone executable (%EXE_NAME%) for manual installation
echo 3. Use the batch files (install.bat, uninstall.bat) for manual installation/uninstallation
echo.
pause
