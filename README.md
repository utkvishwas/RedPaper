# RedPaper

A Windows application that automatically changes your desktop wallpaper by fetching images from Reddit subreddits. The application includes a 24-hour timer to prevent excessive downloads and comes with a complete Windows installer.

## Features

- 🖼️ Automatically fetches wallpapers from Reddit (default: r/wallpaper)
- ⏰ 24-hour cooldown period between wallpaper changes
- 🔄 Handles PC shutdowns/restarts properly (tracks time since last change)
- 📥 Downloads images to `Pictures/redpaper_wallpapers/`
- 🪟 Native Windows desktop wallpaper integration
- 📅 Scheduled task for automatic operation
- 🛠️ Complete Windows installer with uninstaller

## How It Works

The application stores a timestamp of the last successful wallpaper change in `%APPDATA%\RedPaper\last_run.txt`. On each run, it:

1. Checks if 24 hours have passed since the last change
2. If yes: Downloads a new wallpaper from Reddit and sets it
3. If no: Logs the remaining time and exits gracefully

This ensures you get fresh wallpapers regularly without overwhelming the Reddit servers or your bandwidth.

## Building and Installation

### Prerequisites

- Go 1.19 or later
- Windows 7 or later (for the compiled application)
- NSIS (optional, for creating the installer)

### Build Instructions

1. **Clone or download** this repository

2. **Build the application**:

   ```cmd
   .\build.bat
   ```

   This will:

   - Compile the Go application for Windows
   - Create a Windows installer (if NSIS is installed)
   - Generate a ZIP package with all files

   The generated files will be:

   - `build\redpaper.exe` (Standalone executable)
   - `build\RedPaper_Installer.exe` (Windows installer)
   - `build\RedPaper_v1.0.0.zip` (ZIP package)

### Installation Options

#### Option 1: Windows Installer (Recommended)

- Run `RedPaper_Installer.exe`
- Follow the installation wizard
- The installer will:
  - Install the application to Program Files
  - Create a scheduled task to run every hour
  - Add an entry to Add/Remove Programs
  - Create a desktop shortcut

#### Option 2: Manual Installation

- Run `install.bat` as administrator
- This will install the application and set up the scheduled task

#### Option 3: Portable Installation

- Just run `redpaper.exe` directly
- No installation required, but you'll need to run it manually

## Usage

### Automatic (Scheduled Task)

Once installed with the installer or batch script, the application will:

- Run every hour via Windows Task Scheduler
- Check if 24 hours have passed since last wallpaper change
- Download and set new wallpaper only when needed

### Manual Operation

```cmd
redpaper.exe
```

The application will log its actions to the console and exit with appropriate status codes.

## Configuration

### Changing the Subreddit

Edit the main.go file and change the subreddit parameter in the `main()` function:

```go
changer := NewRedPaper("your_subreddit")
```

### Adjusting the Time Period

The application fetches "top" posts from the past day by default. You can modify this in the `Run()` method:

```go
wallpaperData, err := rwc.GetTopWallpaper("week", 10) // Change "day" to "week", "month", etc.
```

## File Locations

- **Application**: `%PROGRAMFILES%\RedPaper\`
- **User Data**: `%APPDATA%\RedPaper\`
- **Wallpapers**: `%USERPROFILE%\Pictures\redpaper_wallpapers\`
- **Logs**: `%PROGRAMFILES%\RedPaper\logs\`

## Uninstallation

### Via Windows Add/Remove Programs

- Search for "RedPaper"
- Click Uninstall

### Manual Uninstallation

- Run `uninstall.bat` as administrator
- Choose whether to keep your downloaded wallpapers

## Troubleshooting

### Application Won't Start

- Ensure you're running as administrator (for installation)
- Check Windows Event Viewer for error details

### Scheduled Task Not Working

- Open Task Scheduler (`taskschd.msc`)
- Look for "RedPaper" task
- Check the task's history for errors

### No Wallpaper Changes

- Check the log file: `%PROGRAMFILES%\RedPaper\logs\redpaper.log`
- Ensure internet connection is available
- Verify the subreddit exists and has images

### Build Issues

- Ensure Go is installed and in your PATH
- For installer creation, install NSIS from https://nsis.sourceforge.io/

## License

This project is provided as-is for personal use. See `license.txt` for details.

## Technical Details

- Written in Go for cross-platform compatibility
- Uses Windows API for native wallpaper setting
- Implements proper error handling and logging
- Scheduled tasks run with user privileges
- Stores timestamps in Unix format for reliability
