; NSIS Installer Script for RedPaper
; This script creates a Windows installer for the RedPaper system tray application

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "WinVer.nsh"

; General Configuration
Name "RedPaper"
OutFile "RedPaper_Installer.exe"
Unicode true
InstallDir "$PROGRAMFILES\RedPaper"
InstallDirRegKey HKCU "Software\RedPaper" ""
RequestExecutionLevel admin

; Modern UI Configuration
!define MUI_ABORTWARNING
!define MUI_ICON "installer.ico"
!define MUI_UNICON "installer.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "wizard.bmp"

; Pages
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

; Languages
!insertmacro MUI_LANGUAGE "English"

; Component descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
!insertmacro MUI_DESCRIPTION_TEXT ${SecApp} "Installs the main RedPaper application files."
!insertmacro MUI_DESCRIPTION_TEXT ${SecShortcut} "Creates a desktop shortcut for easy access to RedPaper."
!insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Creates Start Menu shortcuts for RedPaper and its uninstaller."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Version Information
VIProductVersion "1.0.0.0"
VIAddVersionKey "ProductName" "RedPaper"
VIAddVersionKey "CompanyName" "RedPaper"
VIAddVersionKey "LegalCopyright" "Copyright (C) 2024 RedPaper"
VIAddVersionKey "FileVersion" "1.0.0.0"
VIAddVersionKey "ProductVersion" "1.0.0.0"
VIAddVersionKey "FileDescription" "Automatically changes desktop wallpaper from Reddit"

; Installer Sections
Section "Main Application" SecApp
	SectionIn RO

	SetOutPath "$INSTDIR"

	; Create the application directory
	CreateDirectory "$INSTDIR"

	; Copy the main executable
	DetailPrint "Installing main application..."
	File "redpaper.exe"

	; Copy the icon file for shortcuts
	File "installer.ico"

	; Copy additional files if any
	; File "config.ini"

	; Create data directory for user data
	CreateDirectory "$APPDATA\RedPaper"

	; Store installation folder
	WriteRegStr HKCU "Software\RedPaper" "" $INSTDIR

	; Store application icon path for Windows shell
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayIcon" "$INSTDIR\installer.ico"

	; Create uninstaller
	WriteUninstaller "$INSTDIR\Uninstall.exe"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayName" "RedPaper"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "UninstallString" "$INSTDIR\Uninstall.exe"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "DisplayVersion" "1.0.0"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "Publisher" "RedPaper"
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "NoModify" 1
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "NoRepair" 1

SectionEnd

Section "Desktop Shortcut" SecShortcut
	SectionIn 1 2 3
	CreateShortcut "$DESKTOP\RedPaper.lnk" "$INSTDIR\redpaper.exe" "" "$INSTDIR\installer.ico" 0
SectionEnd

Section "Start Menu Shortcuts" SecStartMenu
	SectionIn 1 2 3
	; Create Start Menu directory
	CreateDirectory "$SMPROGRAMS\RedPaper"

	; Create Start Menu shortcut
	CreateShortcut "$SMPROGRAMS\RedPaper\RedPaper.lnk" "$INSTDIR\redpaper.exe" "" "$INSTDIR\installer.ico" 0

	; Create uninstall shortcut in Start Menu
	CreateShortcut "$SMPROGRAMS\RedPaper\Uninstall RedPaper.lnk" "$INSTDIR\Uninstall.exe" "" "$INSTDIR\installer.ico" 0

	; Write Start Menu registry key for proper uninstall
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper" "StartMenuFolder" "RedPaper"
SectionEnd

; Uninstaller Section
Section "Uninstall"

	; Stop the running tray process before removing files
	DetailPrint "Stopping RedPaper..."
	nsExec::ExecToLog 'taskkill /f /im redpaper.exe'

	; Remove old scheduled task if it exists from a previous install
	nsExec::ExecToLog 'schtasks /delete /tn "RedPaper" /f'

	; Remove startup registry key set by the tray app
	DetailPrint "Removing startup entry..."
	DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "RedPaper"

	; Remove files
	DetailPrint "Removing application files..."
	Delete "$INSTDIR\redpaper.exe"
	Delete "$INSTDIR\Uninstall.exe"
	Delete "$INSTDIR\installer.ico"
	Delete "$INSTDIR\run_redpaper.bat"
	Delete "$INSTDIR\create_task.ps1"
	RMDir "$INSTDIR"

	; Remove user data (ask user first)
	MessageBox MB_YESNO "Do you want to remove all downloaded wallpapers and settings?" IDYES removeUserData IDNO skipUserData
	removeUserData:
	RMDir /r "$LOCALAPPDATA\RedPaper"
	RMDir /r "$PICTURES\redpaper_wallpapers"
	skipUserData:

	; Remove desktop shortcut
	Delete "$DESKTOP\RedPaper.lnk"

	; Remove Start Menu shortcuts
	Delete "$SMPROGRAMS\RedPaper\RedPaper.lnk"
	Delete "$SMPROGRAMS\RedPaper\Uninstall RedPaper.lnk"
	RMDir "$SMPROGRAMS\RedPaper"

	; Remove registry entries
	DeleteRegKey HKCU "Software\RedPaper"
	DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\RedPaper"

SectionEnd

; Functions
Function .onInit
	; Check Windows version (requires Windows 7 or later)
	${IfNot} ${AtLeastWin7}
		MessageBox MB_OK "This application requires Windows 7 or later."
		Abort
	${EndIf}

	; Check if already installed and stop running instance before upgrading
	ReadRegStr $R0 HKCU "Software\RedPaper" ""

	${If} $R0 != ""
		MessageBox MB_YESNO "RedPaper is already installed. Do you want to upgrade/reinstall?" IDYES continueInstall
		Abort
		continueInstall:
		; Kill the running tray process so the exe is not locked
		nsExec::ExecToLog 'taskkill /f /im redpaper.exe'
		; Remove old scheduled task from previous installs
		nsExec::ExecToLog 'schtasks /delete /tn "RedPaper" /f'
	${EndIf}
FunctionEnd

Function .onInstSuccess
	; Launch the tray app after installation
	MessageBox MB_YESNO "Installation complete! Would you like to start RedPaper now?$\n$\nIt will appear as an icon in your system tray." IDYES runNow
	Goto done
	runNow:
	Exec '"$INSTDIR\redpaper.exe"'
	done:
FunctionEnd
