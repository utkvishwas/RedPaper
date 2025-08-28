;NSIS Installer Script for RedPaper
;This script creates a Windows installer that installs the Go wallpaper changer
;and sets up a scheduled task to run it automatically

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "WinVer.nsh"

;General Configuration
Name "RedPaper"
OutFile "RedPaper_Installer.exe"
Unicode True
InstallDir "$PROGRAMFILES\RedPaper"
InstallDirRegKey HKCU "Software\RedPaper" ""
RequestExecutionLevel admin

;Modern UI Configuration
!define MUI_ABORTWARNING
!define MUI_ICON "installer.ico"
!define MUI_UNICON "installer.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "wizard.bmp"

;Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

;Languages
!insertmacro MUI_LANGUAGE "English"

;Component descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} "Installs the main RedPaper application files."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecTask} "Creates a scheduled task to automatically run RedPaper every hour."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcut} "Creates a desktop shortcut for easy access to RedPaper."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Creates Start Menu shortcuts for RedPaper and its uninstaller."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

;Version Information
VIProductVersion "1.0.0.0"
VIAddVersionKey "ProductName" "RedPaper"
VIAddVersionKey "CompanyName" "RedPaper"
VIAddVersionKey "LegalCopyright" "Copyright (C) 2024 RedPaper"
VIAddVersionKey "FileVersion" "1.0.0.0"
VIAddVersionKey "ProductVersion" "1.0.0.0"
VIAddVersionKey "FileDescription" "Automatically changes desktop wallpaper from Reddit"

;Installer Sections
Section "Main Application" SecApp
    SectionIn RO

    SetOutPath "$INSTDIR"

    ;Create the application directory
    CreateDirectory "$INSTDIR"

    ;Copy the main executable
    DetailPrint "Installing main application..."
    File "redpaper.exe"
    
    ;Copy the icon file for shortcuts
    File "installer.ico"

    ;Copy additional files if any
    ;File "config.ini"

    ;Create data directory for user data
    CreateDirectory "$APPDATA\RedPaper"

    ;Store installation folder
    WriteRegStr HKCU "Software\RedPaper" "" $INSTDIR
    
    ;Store application icon path for Windows shell
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayIcon" "$INSTDIR\installer.ico"

    ;Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayName" "RedPaper"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "UninstallString" "$INSTDIR\Uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayVersion" "1.0.0"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "Publisher" "RedPaper"
    WriteRegDWord HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "NoModify" 1
    WriteRegDWord HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "NoRepair" 1

SectionEnd

Section "Create Scheduled Task" SecTask
    DetailPrint "Setting up scheduled task..."

    ;Create a batch file to run the program
    FileOpen $0 "$INSTDIR\run_redpaper.bat" w
    FileWrite $0 '@echo off$\r$\n'
    FileWrite $0 'if not exist "%LOCALAPPDATA%\RedPaper\logs" mkdir "%LOCALAPPDATA%\RedPaper\logs"$\r$\n'
    FileWrite $0 '"$INSTDIR\redpaper.exe" >> "%LOCALAPPDATA%\RedPaper\logs\redpaper.log" 2>&1$\r$\n'
    FileWrite $0 'exit /b %errorlevel%$\r$\n'
    FileClose $0

    ;Create the scheduled task using PowerShell
    ;This task will run every hour and check if 24 hours have passed
    FileOpen $0 "$INSTDIR\create_task.ps1" w
    FileWrite $0 '$$taskName = "RedPaper"$\r$\n'
    FileWrite $0 '$$taskPath = "$$env:ProgramFiles\RedPaper\run_redpaper.bat"$\r$\n'
    FileWrite $0 '$\r$\n'
    FileWrite $0 '# Check if task already exists and remove it$\r$\n'
    FileWrite $0 'if (Get-ScheduledTask -TaskName $$taskName -ErrorAction SilentlyContinue) {$\r$\n'
    FileWrite $0 '    Unregister-ScheduledTask -TaskName $$taskName -Confirm:$$false$\r$\n'
    FileWrite $0 '    Write-Host "Removed existing task: $$taskName"$\r$\n'
    FileWrite $0 '}$\r$\n'
    FileWrite $0 '$\r$\n'
    FileWrite $0 '# Create new scheduled task$\r$\n'
    FileWrite $0 '$$action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument "/c `"$$taskPath`""$\r$\n'
    FileWrite $0 '$\r$\n'
    FileWrite $0 '# Create triggers separately (cannot combine with +=)$\r$\n'
    FileWrite $0 '$$triggerStartup = New-ScheduledTaskTrigger -AtStartup$\r$\n'
    FileWrite $0 '$$triggerHourly = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Hours 1)$\r$\n'
    FileWrite $0 '$\r$\n'
    FileWrite $0 '# Use correct LogonType$\r$\n'
    FileWrite $0 '$$principal = New-ScheduledTaskPrincipal -UserId $$env:USERNAME -LogonType Interactive$\r$\n'
    FileWrite $0 '$$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RunOnlyIfNetworkAvailable$\r$\n'
    FileWrite $0 '$\r$\n'
    FileWrite $0 '# Create the task with multiple triggers$\r$\n'
    FileWrite $0 '$$task = New-ScheduledTask -Action $$action -Trigger @($$triggerStartup, $$triggerHourly) -Principal $$principal -Settings $$settings$\r$\n'
    FileWrite $0 'Register-ScheduledTask -TaskName $$taskName -InputObject $$task$\r$\n'
    FileWrite $0 'Write-Host "Successfully created scheduled task: $$taskName"$\r$\n'
    FileClose $0

    ;Execute the PowerShell script to create the task
    nsExec::ExecToLog 'powershell.exe -ExecutionPolicy Bypass -File "$INSTDIR\create_task.ps1"'
    
    ;If PowerShell fails, try schtasks as fallback
    Pop $0
    IntCmp $0 0 taskSuccess taskFailed taskFailed
    taskFailed:
        DetailPrint "PowerShell method failed, trying schtasks fallback..."
        nsExec::ExecToLog 'schtasks /create /tn "RedPaper" /tr "$INSTDIR\run_redpaper.bat" /sc hourly /f'
        Pop $0
        IntCmp $0 0 taskSuccess taskStillFailed taskStillFailed
        taskStillFailed:
            DetailPrint "Warning: Could not create scheduled task automatically"
            MessageBox MB_OK "Scheduled task creation failed. You may need to create it manually in Task Scheduler."
            Goto taskDone
    taskSuccess:
        DetailPrint "Scheduled task created successfully"
    taskDone:

SectionEnd

Section "Desktop Shortcut" SecShortcut
    SectionIn 1 2 3
    CreateShortCut "$DESKTOP\RedPaper.lnk" "$INSTDIR\redpaper.exe" "" "$INSTDIR\installer.ico" 0
SectionEnd

Section "Start Menu Shortcuts" SecStartMenu
    SectionIn 1 2 3
    ;Create Start Menu directory
    CreateDirectory "$SMPROGRAMS\RedPaper"
    
    ;Create Start Menu shortcut
    CreateShortCut "$SMPROGRAMS\RedPaper\RedPaper.lnk" "$INSTDIR\redpaper.exe" "" "$INSTDIR\installer.ico" 0
    
    ;Create uninstall shortcut in Start Menu
    CreateShortCut "$SMPROGRAMS\RedPaper\Uninstall RedPaper.lnk" "$INSTDIR\Uninstall.exe" "" "$INSTDIR\installer.ico" 0
    
    ;Write Start Menu registry key for proper uninstall
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "StartMenuFolder" "RedPaper"
SectionEnd

;Uninstaller Section
Section "Uninstall"

    ;Remove scheduled task
    DetailPrint "Removing scheduled task..."
    nsExec::ExecToLog 'schtasks /delete /tn "RedPaper" /f'

    ;Remove files
    DetailPrint "Removing application files..."
    Delete "$INSTDIR\redpaper.exe"
    Delete "$INSTDIR\Uninstall.exe"
    Delete "$INSTDIR\installer.ico"
    Delete "$INSTDIR\run_redpaper.bat"
    Delete "$INSTDIR\create_task.ps1"
    RMDir "$INSTDIR"

    ;Remove user data (ask user first)
    MessageBox MB_YESNO "Do you want to remove all downloaded wallpapers and settings?" IDYES removeUserData IDNO skipUserData
    removeUserData:
        RMDir /r "$APPDATA\RedPaper"
        RMDir /r "$PICTURES\redpaper_wallpapers"
    skipUserData:

    ;Remove desktop shortcut
    Delete "$DESKTOP\RedPaper.lnk"
    
    ;Remove Start Menu shortcuts (if they exist)
    Delete "$SMPROGRAMS\RedPaper\RedPaper.lnk"
    Delete "$SMPROGRAMS\RedPaper\Uninstall RedPaper.lnk"
    RMDir "$SMPROGRAMS\RedPaper"

    ;Remove registry entries
    DeleteRegKey HKCU "Software\RedPaper"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper"

    ;Remove from PATH if added (future enhancement)
    ;Call un.RemoveFromPath

SectionEnd

;Functions
Function .onInit
    ;Check if already installed
    ReadRegStr $R0 HKCU "Software\RedPaper" ""
    ${If} $R0 != ""
        MessageBox MB_YESNO "RedPaper is already installed. Do you want to reinstall?" IDYES continueInstall
        Abort
        continueInstall:
    ${EndIf}

    ;Check Windows version (requires Windows 7 or later)
    ${IfNot} ${AtLeastWin7}
        MessageBox MB_OK "This application requires Windows 7 or later."
        Abort
    ${EndIf}
FunctionEnd

Function .onInstSuccess
    ;Run the application once after installation
    MessageBox MB_YESNO "Installation complete! Would you like to run RedPaper now?" IDYES runNow
    Goto done
    runNow:
        Exec '"$INSTDIR\redpaper.exe"'
    done:
FunctionEnd
