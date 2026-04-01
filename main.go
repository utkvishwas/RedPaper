package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
	"golang.org/x/sys/windows/registry"
)

// Windows API constants
const (
	SPI_SETDESKWALLPAPER = 0x0014
	SPIF_UPDATEINIFILE   = 0x01
	SPIF_SENDCHANGE      = 0x02
	MAX_WALLPAPER_SIZE   = 16 * 1024 * 1024
	COMPRESSION_QUALITY  = 85
	APP_NAME             = "RedPaper"
	STARTUP_REG_KEY      = `Software\Microsoft\Windows\CurrentVersion\Run`
)

//go:embed installer.ico
var iconBytes []byte

var version = "1.0.0"

// Config holds persistent user preferences
type Config struct {
	Subreddit     string    `json:"subreddit"`
	IntervalHours int       `json:"interval_hours"`
	TimePeriod    string    `json:"time_period"`
	LastRun       time.Time `json:"last_run"`
}

// Reddit API structures
type RedditResponse struct {
	Data struct {
		Children []struct {
			Data PostData `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type PostData struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
}

type WallpaperData struct {
	URL   string
	Title string
	Score int
}

// RedPaper is the core application struct
type RedPaper struct {
	Config         Config
	DownloadFolder string
	DataFolder     string
	configFile     string
	Client         *http.Client
}

var (
	app        *RedPaper
	timerReset = make(chan struct{}, 1)
)

// NewRedPaper initialises folders, logging, and loads config
func NewRedPaper() *RedPaper {
	homeDir, _ := os.UserHomeDir()
	downloadFolder := filepath.Join(homeDir, "Pictures", "redpaper_wallpapers")
	dataFolder := filepath.Join(homeDir, "AppData", "Local", "RedPaper")
	configFile := filepath.Join(dataFolder, "config.json")

	os.MkdirAll(downloadFolder, 0755)
	os.MkdirAll(dataFolder, 0755)

	logPath := filepath.Join(dataFolder, "redpaper.log")
	if lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(lf)
		log.SetFlags(log.LstdFlags)
	}

	rp := &RedPaper{
		DownloadFolder: downloadFolder,
		DataFolder:     dataFolder,
		configFile:     configFile,
		Client:         &http.Client{Timeout: 30 * time.Second},
	}
	rp.loadConfig()
	return rp
}

func (rp *RedPaper) loadConfig() {
	rp.Config = Config{Subreddit: "wallpaper", IntervalHours: 24, TimePeriod: "day"}
	data, err := os.ReadFile(rp.configFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &rp.Config)
	if rp.Config.Subreddit == "" {
		rp.Config.Subreddit = "wallpaper"
	}
	if rp.Config.IntervalHours <= 0 {
		rp.Config.IntervalHours = 24
	}
	if rp.Config.TimePeriod == "" {
		rp.Config.TimePeriod = "day"
	}
}

func (rp *RedPaper) saveConfig() {
	data, _ := json.MarshalIndent(rp.Config, "", "  ")
	os.WriteFile(rp.configFile, data, 0644)
}

// GetTopWallpaper fetches the top image post from the configured subreddit
func (rp *RedPaper) GetTopWallpaper() (*WallpaperData, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/top/.json?t=%s&limit=25",
		rp.Config.Subreddit, rp.Config.TimePeriod)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := rp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from Reddit API", resp.StatusCode)
	}

	var result RedditResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	for _, post := range result.Data.Children {
		if isImageURL(post.Data.URL) {
			return &WallpaperData{
				URL:   post.Data.URL,
				Title: post.Data.Title,
				Score: post.Data.Score,
			}, nil
		}
	}
	return nil, fmt.Errorf("no suitable image found in r/%s", rp.Config.Subreddit)
}

func isImageURL(url string) bool {
	lower := strings.ToLower(url)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".bmp"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// DownloadImage saves a wallpaper image to the download folder
func (rp *RedPaper) DownloadImage(w *WallpaperData) (string, error) {
	resp, err := rp.Client.Get(w.URL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d downloading image", resp.StatusCode)
	}

	ext := "jpg"
	parts := strings.Split(w.URL, ".")
	if len(parts) > 1 {
		e := strings.ToLower(parts[len(parts)-1])
		if e == "jpg" || e == "jpeg" || e == "png" || e == "bmp" {
			ext = e
		}
	}

	safe := sanitizeFilename(w.Title)
	if len(safe) > 50 {
		safe = safe[:50]
	}
	filename := fmt.Sprintf("%s_%s.%s", time.Now().Format("20060102"), safe, ext)
	path := filepath.Join(rp.DownloadFolder, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer f.Close()

	if _, err = io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}
	log.Printf("Downloaded: %s", filename)
	return path, nil
}

func sanitizeFilename(s string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*\s]+`)
	clean := strings.Trim(reg.ReplaceAllString(s, "_"), "_")
	if clean == "" {
		return "redpaper"
	}
	return clean
}

// SetWallpaper sets the desktop wallpaper via the Windows API
func (rp *RedPaper) SetWallpaper(imagePath string) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	spi := user32.NewProc("SystemParametersInfoW")
	ptr, err := syscall.UTF16PtrFromString(imagePath)
	if err != nil {
		return fmt.Errorf("UTF16 conversion failed: %w", err)
	}
	ret, _, callErr := spi.Call(
		uintptr(SPI_SETDESKWALLPAPER), 0,
		uintptr(unsafe.Pointer(ptr)),
		uintptr(SPIF_UPDATEINIFILE|SPIF_SENDCHANGE),
	)
	if ret == 0 {
		return fmt.Errorf("SystemParametersInfoW failed: %v", callErr)
	}
	return nil
}

// compressImage compresses the image only if it exceeds the Windows wallpaper size limit
func (rp *RedPaper) compressImage(imagePath string) (string, error) {
	info, err := os.Stat(imagePath)
	if err != nil {
		return "", err
	}
	if info.Size() <= MAX_WALLPAPER_SIZE {
		return imagePath, nil
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}

	compressed := strings.TrimSuffix(imagePath, filepath.Ext(imagePath)) + "_c.jpg"
	cf, err := os.Create(compressed)
	if err != nil {
		return "", err
	}
	defer cf.Close()

	if err := jpeg.Encode(cf, img, &jpeg.Options{Quality: COMPRESSION_QUALITY}); err != nil {
		return "", err
	}
	log.Printf("Compressed image saved: %s", filepath.Base(compressed))
	return compressed, nil
}

// changeWallpaper fetches, downloads, and applies a new wallpaper
func (rp *RedPaper) changeWallpaper() error {
	log.Printf("Fetching wallpaper from r/%s (period: %s)", rp.Config.Subreddit, rp.Config.TimePeriod)

	wallpaper, err := rp.GetTopWallpaper()
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	log.Printf("Found: %s (score: %d)", wallpaper.Title, wallpaper.Score)

	imagePath, err := rp.DownloadImage(wallpaper)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	finalPath, err := rp.compressImage(imagePath)
	if err != nil {
		log.Printf("Compression skipped: %v", err)
		finalPath = imagePath
	}

	if err := rp.SetWallpaper(finalPath); err != nil {
		return fmt.Errorf("set wallpaper: %w", err)
	}

	rp.Config.LastRun = time.Now()
	rp.saveConfig()
	log.Println("Wallpaper changed successfully")
	return nil
}

// isStartupEnabled checks if the app is registered in the Windows startup registry
func (rp *RedPaper) isStartupEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, STARTUP_REG_KEY, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(APP_NAME)
	return err == nil
}

// setStartupEnabled adds or removes the app from the Windows startup registry
func (rp *RedPaper) setStartupEnabled(enable bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, STARTUP_REG_KEY, registry.SET_VALUE)
	if err != nil {
		log.Printf("Registry open error: %v", err)
		return
	}
	defer k.Close()
	if enable {
		exePath, _ := os.Executable()
		if err := k.SetStringValue(APP_NAME, exePath); err != nil {
			log.Printf("Registry write error: %v", err)
		}
	} else {
		k.DeleteValue(APP_NAME)
	}
}

// wallpaperLoop is the background goroutine that changes the wallpaper on schedule
func wallpaperLoop() {
	for {
		interval := time.Duration(app.Config.IntervalHours) * time.Hour

		var wait time.Duration
		if !app.Config.LastRun.IsZero() {
			elapsed := time.Since(app.Config.LastRun)
			if elapsed < interval {
				wait = interval - elapsed
			}
		}

		if wait > 0 {
			select {
			case <-time.After(wait):
			case <-timerReset:
				continue
			}
		}

		if err := app.changeWallpaper(); err != nil {
			log.Printf("Wallpaper loop error: %v", err)
			select {
			case <-time.After(10 * time.Minute):
			case <-timerReset:
			}
			continue
		}

		select {
		case <-time.After(time.Duration(app.Config.IntervalHours) * time.Hour):
		case <-timerReset:
		}
	}
}

func sendTimerReset() {
	select {
	case timerReset <- struct{}{}:
	default:
	}
}

func setIntervalChecks(hours int, m6h, m12h, m24h, m48h *systray.MenuItem) {
	for h, item := range map[int]*systray.MenuItem{6: m6h, 12: m12h, 24: m24h, 48: m48h} {
		if h == hours {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func setPeriodChecks(period string, mDay, mWeek, mMonth, mAll *systray.MenuItem) {
	for p, item := range map[string]*systray.MenuItem{"day": mDay, "week": mWeek, "month": mMonth, "all": mAll} {
		if p == period {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTooltip("RedPaper - Reddit Wallpaper Changer")

	mTitle := systray.AddMenuItem("RedPaper v"+version, "Reddit Wallpaper Changer")
	mTitle.Disable()
	systray.AddSeparator()

	mChangeNow := systray.AddMenuItem("Change Wallpaper Now", "Fetch and apply a new wallpaper immediately")
	systray.AddSeparator()

	mSubreddit := systray.AddMenuItem(fmt.Sprintf("Subreddit: r/%s", app.Config.Subreddit), "Click to change subreddit")

	mInterval := systray.AddMenuItem("Interval", "How often to change wallpaper")
	m6h := mInterval.AddSubMenuItemCheckbox("Every 6 hours", "", app.Config.IntervalHours == 6)
	m12h := mInterval.AddSubMenuItemCheckbox("Every 12 hours", "", app.Config.IntervalHours == 12)
	m24h := mInterval.AddSubMenuItemCheckbox("Every 24 hours", "", app.Config.IntervalHours == 24)
	m48h := mInterval.AddSubMenuItemCheckbox("Every 48 hours", "", app.Config.IntervalHours == 48)

	mPeriod := systray.AddMenuItem("Fetch Period", "Reddit top posts time period")
	mDay := mPeriod.AddSubMenuItemCheckbox("Day", "Top posts of the day", app.Config.TimePeriod == "day")
	mWeek := mPeriod.AddSubMenuItemCheckbox("Week", "Top posts of the week", app.Config.TimePeriod == "week")
	mMonth := mPeriod.AddSubMenuItemCheckbox("Month", "Top posts of the month", app.Config.TimePeriod == "month")
	mAll := mPeriod.AddSubMenuItemCheckbox("All Time", "All-time top posts", app.Config.TimePeriod == "all")

	systray.AddSeparator()
	mStartup := systray.AddMenuItemCheckbox("Start with Windows", "Launch RedPaper at login", app.isStartupEnabled())
	mOpenFolder := systray.AddMenuItem("Open Wallpaper Folder", "Browse saved wallpapers")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit RedPaper")

	go wallpaperLoop()

	for {
		select {
		case <-mChangeNow.ClickedCh:
			go func() {
				if err := app.changeWallpaper(); err != nil {
					log.Printf("Manual change error: %v", err)
				} else {
					sendTimerReset()
				}
			}()

		case <-mSubreddit.ClickedCh:
			go func() {
				result, err := zenity.Entry(
					"Enter subreddit name (without r/):",
					zenity.Title("RedPaper - Change Subreddit"),
					zenity.EntryText(app.Config.Subreddit),
				)
				if err != nil || strings.TrimSpace(result) == "" {
					return
				}
				sub := strings.TrimPrefix(strings.TrimSpace(result), "r/")
				if sub == "" {
					return
				}
				app.Config.Subreddit = sub
				app.saveConfig()
				mSubreddit.SetTitle(fmt.Sprintf("Subreddit: r/%s", sub))
			}()

		case <-m6h.ClickedCh:
			app.Config.IntervalHours = 6
			app.saveConfig()
			setIntervalChecks(6, m6h, m12h, m24h, m48h)
			sendTimerReset()

		case <-m12h.ClickedCh:
			app.Config.IntervalHours = 12
			app.saveConfig()
			setIntervalChecks(12, m6h, m12h, m24h, m48h)
			sendTimerReset()

		case <-m24h.ClickedCh:
			app.Config.IntervalHours = 24
			app.saveConfig()
			setIntervalChecks(24, m6h, m12h, m24h, m48h)
			sendTimerReset()

		case <-m48h.ClickedCh:
			app.Config.IntervalHours = 48
			app.saveConfig()
			setIntervalChecks(48, m6h, m12h, m24h, m48h)
			sendTimerReset()

		case <-mDay.ClickedCh:
			app.Config.TimePeriod = "day"
			app.saveConfig()
			setPeriodChecks("day", mDay, mWeek, mMonth, mAll)

		case <-mWeek.ClickedCh:
			app.Config.TimePeriod = "week"
			app.saveConfig()
			setPeriodChecks("week", mDay, mWeek, mMonth, mAll)

		case <-mMonth.ClickedCh:
			app.Config.TimePeriod = "month"
			app.saveConfig()
			setPeriodChecks("month", mDay, mWeek, mMonth, mAll)

		case <-mAll.ClickedCh:
			app.Config.TimePeriod = "all"
			app.saveConfig()
			setPeriodChecks("all", mDay, mWeek, mMonth, mAll)

		case <-mStartup.ClickedCh:
			enabled := app.isStartupEnabled()
			app.setStartupEnabled(!enabled)
			if !enabled {
				mStartup.Check()
			} else {
				mStartup.Uncheck()
			}

		case <-mOpenFolder.ClickedCh:
			exec.Command("explorer", app.DownloadFolder).Start()

		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func onExit() {
	log.Println("RedPaper exiting")
}

func main() {
	// Single-instance check via named mutex
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")
	getLastError := kernel32.NewProc("GetLastError")
	mutexName, _ := syscall.UTF16PtrFromString("RedPaperSingleInstanceMutex")
	createMutex.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	if code, _, _ := getLastError.Call(); code == 183 { // ERROR_ALREADY_EXISTS
		return
	}

	app = NewRedPaper()
	log.Printf("RedPaper v%s starting", version)
	systray.Run(onReady, onExit)
}
