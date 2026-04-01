<h1 align="center">RedPaper</h1>
<h3 align="center">A lightweight Windows system tray application that automatically changes your desktop wallpaper by fetching top images from any Reddit subreddit.</h3>

---

<p align="center">

<img width="100%" alt="Banner image" src="https://github.com/user-attachments/assets/fbe2c57c-85c4-49a3-a6d8-d2ac4de9bcc8" />

<br/>

---

## Features

- 🖼️ Fetches top wallpapers from any Reddit subreddit (default: r/wallpaper)
- 🔔 Runs silently as a **system tray icon** — no terminal windows, ever
- ⚙️ Right-click tray menu to change all settings on the fly
- ⏱️ Configurable wallpaper change interval (6h / 12h / 24h / 48h)
- � Configurable Reddit time period (Day / Week / Month / All Time)
- 🚀 Optional **Start with Windows** toggle built into the tray menu
- 📥 Downloads wallpapers to `Pictures\redpaper_wallpapers\`
- 🗜️ Auto-compresses oversized images for Windows compatibility
- 🛠️ Complete Windows installer with uninstaller

## How It Works

RedPaper runs as a persistent background process in the system tray. It uses an internal Go timer — no Windows Task Scheduler required. On startup it:

1. Loads your settings from `%LOCALAPPDATA%\RedPaper\config.json`
2. Calculates the time remaining until the next wallpaper change
3. Waits silently, then fetches a top image post from your chosen subreddit
4. Downloads, optionally compresses, and sets it as your desktop wallpaper
5. Saves the timestamp and sleeps until the next interval

All activity is logged to `%LOCALAPPDATA%\RedPaper\redpaper.log`.

## Tray Menu

Right-click the RedPaper icon in the system tray to access:

| Option                    | Description                                            |
| ------------------------- | ------------------------------------------------------ |
| **Change Wallpaper Now**  | Immediately fetch and apply a new wallpaper            |
| **Subreddit: r/...**      | Click to type a new subreddit name                     |
| **Interval**              | Choose how often to change: 6h / 12h / 24h / 48h       |
| **Fetch Period**          | Reddit top posts period: Day / Week / Month / All Time |
| **Start with Windows**    | Toggle automatic launch at login                       |
| **Open Wallpaper Folder** | Browse your saved wallpapers in Explorer               |
| **Quit**                  | Exit RedPaper                                          |

## Building and Installation

### Prerequisites

- Go 1.21 or later
- Windows 7 or later
- NSIS (optional, for creating the installer)

### Build Instructions

1. **Clone or download** this repository

2. **Build the application**:

   ```cmd
   .\build.bat
   ```

   This will:
   - Compile the Go application (GUI mode, no console window)
   - Create a Windows installer (if NSIS is installed)
   - Generate a ZIP package

   Output files:
   - `build\redpaper.exe` — Standalone executable
   - `build\RedPaper_Installer.exe` — Windows installer (requires NSIS)
   - `build\RedPaper_v1.0.1.zip` — ZIP package

### Installation Options

#### Option 1: Windows Installer (Recommended)

- Run `RedPaper_Installer.exe`
- Follow the installation wizard
- The installer will:
  - Copy the application to `Program Files\RedPaper\`
  - Add an entry to Add/Remove Programs
  - Create a desktop and Start Menu shortcut
  - Offer to launch RedPaper immediately
- After launch, right-click the tray icon and enable **Start with Windows** if desired

#### Option 2: Manual Installation

- Run `install.bat` as administrator
- RedPaper will be copied to `Program Files\RedPaper\` and launched automatically

#### Option 3: Portable

- Run `redpaper.exe` directly from any folder
- A tray icon will appear — no installation needed

## Configuration

All settings are changed via the **system tray menu** — no file editing required.

Settings are saved automatically to `%LOCALAPPDATA%\RedPaper\config.json`:

```json
{
  "subreddit": "wallpaper",
  "interval_hours": 24,
  "time_period": "day",
  "last_run": "2024-01-01T12:00:00Z"
}
```

## File Locations

| Path                                          | Contents              |
| --------------------------------------------- | --------------------- |
| `%PROGRAMFILES%\RedPaper\`                    | Application files     |
| `%LOCALAPPDATA%\RedPaper\config.json`         | Settings              |
| `%LOCALAPPDATA%\RedPaper\redpaper.log`        | Activity log          |
| `%USERPROFILE%\Pictures\redpaper_wallpapers\` | Downloaded wallpapers |

## Uninstallation

### Via Windows Add/Remove Programs

- Search for "RedPaper" and click Uninstall

### Manual Uninstallation

- Run `uninstall.bat` as administrator
- Choose whether to keep your downloaded wallpapers and settings

The uninstaller will stop the running process, remove the startup registry entry, and clean up all files.

## Troubleshooting

### Tray icon doesn't appear

- Check if `redpaper.exe` is already running in Task Manager
- Only one instance is allowed; a second launch exits silently

### No wallpaper changes

- Check the log: `%LOCALAPPDATA%\RedPaper\redpaper.log`
- Ensure internet access is available
- Try **Change Wallpaper Now** from the tray menu to test immediately
- Verify the subreddit has direct image posts (`.jpg`, `.png`, etc.)

### Build Issues

- Ensure Go 1.21+ is installed and in your PATH
- For installer creation, install NSIS from https://nsis.sourceforge.io/

## License

This project is provided as-is for personal use. See `license.txt` for details.

## Technical Details

- Written in Go — single self-contained binary, no runtime dependencies
- System tray via [`getlantern/systray`](https://github.com/getlantern/systray)
- Native input dialogs via [`ncruces/zenity`](https://github.com/ncruces/zenity)
- Wallpaper set via `SystemParametersInfoW` (Windows API, no external tools)
- Startup toggle writes directly to `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- Single-instance enforcement via a named Win32 mutex
- No Windows Task Scheduler dependency
