package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Windows API constants
const (
	SPI_SETDESKWALLPAPER = 0x0014
	SPIF_UPDATEINIFILE   = 0x01
	SPIF_SENDCHANGE      = 0x02
)

// Image processing constants
const (
	MAX_WALLPAPER_SIZE  = 16 * 1024 * 1024 // 16MB limit for Windows wallpaper
	COMPRESSION_QUALITY = 85               // JPEG quality (0-100) - reduced to 50%
)

// Reddit API response structures
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

type RedPaper struct {
	Subreddit      string
	DownloadFolder string
	Client         *http.Client
	DataFolder     string
}

// NewRedPaper creates a new instance
func NewRedPaper(subreddit string) *RedPaper {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v", err)
		homeDir = "."
	}

	downloadFolder := filepath.Join(homeDir, "Pictures", "redpaper_wallpapers")
	dataFolder := filepath.Join(homeDir, "AppData", "Local", "RedPaper")

	// Create folders if they don't exist
	if err := os.MkdirAll(downloadFolder, 0755); err != nil {
		log.Printf("Error creating download folder: %v", err)
	}
	if err := os.MkdirAll(dataFolder, 0755); err != nil {
		log.Printf("Error creating data folder: %v", err)
	}

	return &RedPaper{
		Subreddit:      subreddit,
		DownloadFolder: downloadFolder,
		DataFolder:     dataFolder,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTopWallpaper fetches top wallpaper from Reddit subreddit
func (rwc *RedPaper) GetTopWallpaper(timePeriod string, limit int) (*WallpaperData, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/top/.json?t=%s&limit=%d",
		rwc.Subreddit, timePeriod, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := rwc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var redditResp RedditResponse
	if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	// Find first suitable image
	for _, post := range redditResp.Data.Children {
		postData := post.Data
		if rwc.IsImageURL(postData.URL) {
			return &WallpaperData{
				URL:   postData.URL,
				Title: postData.Title,
				Score: postData.Score,
			}, nil
		}
	}

	return nil, fmt.Errorf("no suitable wallpaper found")
}

// IsImageURL checks if URL is a direct image link
func (rwc *RedPaper) IsImageURL(url string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".bmp", ".gif"}
	lowerURL := strings.ToLower(url)

	for _, ext := range imageExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

// DownloadImage downloads image from URL
func (rwc *RedPaper) DownloadImage(wallpaper *WallpaperData) (string, error) {
	resp, err := rwc.Client.Get(wallpaper.URL)
	if err != nil {
		return "", fmt.Errorf("error downloading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error downloading image: %d", resp.StatusCode)
	}

	// Get file extension
	fileExt := "jpg"
	parts := strings.Split(wallpaper.URL, ".")
	if len(parts) > 1 {
		ext := strings.ToLower(parts[len(parts)-1])
		if contains([]string{"jpg", "jpeg", "png", "bmp"}, ext) {
			fileExt = ext
		}
	}

	// Create safe filename
	safeTitle := rwc.sanitizeFilename(wallpaper.Title)
	if len(safeTitle) > 50 {
		safeTitle = safeTitle[:50]
	}

	filename := fmt.Sprintf("%s_%s.%s",
		time.Now().Format("20060102"), safeTitle, fileExt)
	filepath := filepath.Join(rwc.DownloadFolder, filename)

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Copy image data to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving image: %w", err)
	}

	log.Printf("Downloaded wallpaper: %s", filename)
	return filepath, nil
}

// sanitizeFilename removes invalid characters from filename
func (rwc *RedPaper) sanitizeFilename(filename string) string {
	// Remove invalid characters for Windows filenames
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	clean := reg.ReplaceAllString(filename, "")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		clean = "redpaper"
	}
	return clean
}

// SetWallpaper sets wallpaper on Windows using syscalls
func (rwc *RedPaper) SetWallpaper(imagePath string) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	systemParametersInfo := user32.NewProc("SystemParametersInfoW")

	// Convert string to UTF-16
	imagePathPtr, err := syscall.UTF16PtrFromString(imagePath)
	if err != nil {
		return fmt.Errorf("error converting path to UTF-16: %w", err)
	}

	// Call Windows API
	ret, _, err := systemParametersInfo.Call(
		uintptr(SPI_SETDESKWALLPAPER),
		uintptr(0),
		uintptr(unsafe.Pointer(imagePathPtr)),
		uintptr(SPIF_UPDATEINIFILE|SPIF_SENDCHANGE),
	)

	if ret == 0 {
		return fmt.Errorf("failed to set wallpaper: %v", err)
	}

	log.Printf("Wallpaper set successfully: %s", imagePath)
	return nil
}

// getLastRunTime reads the last run timestamp from file
func (rwc *RedPaper) getLastRunTime() (time.Time, error) {
	timestampFile := filepath.Join(rwc.DataFolder, "last_run.txt")

	// Check if file exists
	if _, err := os.Stat(timestampFile); os.IsNotExist(err) {
		// Return zero time if file doesn't exist (first run)
		return time.Time{}, nil
	}

	file, err := os.Open(timestampFile)
	if err != nil {
		return time.Time{}, fmt.Errorf("error opening timestamp file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		timestampStr := scanner.Text()
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("error parsing timestamp: %w", err)
		}
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, fmt.Errorf("empty timestamp file")
}

// saveLastRunTime saves the current timestamp to file
func (rwc *RedPaper) saveLastRunTime() error {
	timestampFile := filepath.Join(rwc.DataFolder, "last_run.txt")
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	file, err := os.Create(timestampFile)
	if err != nil {
		return fmt.Errorf("error creating timestamp file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(timestamp)
	if err != nil {
		return fmt.Errorf("error writing timestamp: %w", err)
	}

	return nil
}

// shouldRunWallpaperChange checks if 24 hours have passed since last run
func (rwc *RedPaper) shouldRunWallpaperChange() (bool, time.Duration, error) {
	lastRun, err := rwc.getLastRunTime()
	if err != nil {
		return false, 0, fmt.Errorf("error getting last run time: %w", err)
	}

	// If it's the first run (zero time), allow it to run
	if lastRun.IsZero() {
		log.Println("First run detected, proceeding with wallpaper change")
		return true, 0, nil
	}

	timeSinceLastRun := time.Since(lastRun)
	remainingTime := 24*time.Hour - timeSinceLastRun

	log.Printf("Last wallpaper change was %v ago", timeSinceLastRun)

	if timeSinceLastRun >= 24*time.Hour {
		log.Println("24 hours have passed, proceeding with wallpaper change")
		return true, 0, nil
	}

	log.Printf("Only %v has passed since last change. Next change in %v", timeSinceLastRun, remainingTime)
	return false, remainingTime, nil
}

// Run executes the main wallpaper changing logic
func (rwc *RedPaper) Run() error {
	// Check if 24 hours have passed since last run
	shouldRun, remainingTime, err := rwc.shouldRunWallpaperChange()
	if err != nil {
		log.Printf("Error checking run time: %v", err)
		// Continue with run despite error in time checking
	}

	if !shouldRun {
		log.Printf("Wallpaper change skipped. Next change available in: %v", remainingTime)
		return nil // Not an error, just skipping
	}

	log.Printf("Fetching wallpaper from r/%s", rwc.Subreddit)

	// Get wallpaper data
	wallpaperData, err := rwc.GetTopWallpaper("day", 10)
	if err != nil {
		return fmt.Errorf("error fetching wallpaper: %w", err)
	}

	log.Printf("Found wallpaper: %s (Score: %d)", wallpaperData.Title, wallpaperData.Score)

	// Download image
	imagePath, err := rwc.DownloadImage(wallpaperData)
	if err != nil {
		return fmt.Errorf("error downloading image: %w", err)
	}

	// Compress image if necessary
	finalImagePath, err := rwc.compressImage(imagePath)
	if err != nil {
		log.Printf("Warning: Failed to compress image, trying with original: %v", err)
		finalImagePath = imagePath // Fall back to original if compression fails
	}

	// Set wallpaper
	if err := rwc.SetWallpaper(finalImagePath); err != nil {
		return fmt.Errorf("error setting wallpaper: %w", err)
	}

	// Save the timestamp of successful run
	if err := rwc.saveLastRunTime(); err != nil {
		log.Printf("Warning: Could not save timestamp: %v", err)
		// Don't fail the whole operation for this
	}

	return nil
}

// forceRun bypasses the 24-hour check for testing purposes
func (rwc *RedPaper) forceRun() error {
	log.Printf("Fetching wallpaper from r/%s (forced run)", rwc.Subreddit)

	// Get wallpaper data
	wallpaperData, err := rwc.GetTopWallpaper("day", 10)
	if err != nil {
		return fmt.Errorf("error fetching wallpaper: %w", err)
	}

	log.Printf("Found wallpaper: %s (Score: %d)", wallpaperData.Title, wallpaperData.Score)

	// Download image
	imagePath, err := rwc.DownloadImage(wallpaperData)
	if err != nil {
		return fmt.Errorf("error downloading image: %w", err)
	}

	// Compress image if necessary
	finalImagePath, err := rwc.compressImage(imagePath)
	if err != nil {
		log.Printf("Warning: Failed to compress image, trying with original: %v", err)
		finalImagePath = imagePath // Fall back to original if compression fails
	}

	// Set wallpaper
	if err := rwc.SetWallpaper(finalImagePath); err != nil {
		return fmt.Errorf("error setting wallpaper: %w", err)
	}

	// Save the timestamp of successful run
	if err := rwc.saveLastRunTime(); err != nil {
		log.Printf("Warning: Could not save timestamp: %v", err)
		// Don't fail the whole operation for this
	}

	return nil
}

// compressImage compresses an image if it's too large for Windows wallpaper
func (rwc *RedPaper) compressImage(imagePath string) (string, error) {
	// Check file size
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return "", fmt.Errorf("error getting file info: %w", err)
	}

	// If file is within size limit, return original path
	if fileInfo.Size() <= MAX_WALLPAPER_SIZE {
		log.Printf("Image size (%d bytes) is within Windows limit (%d bytes), no compression needed",
			fileInfo.Size(), MAX_WALLPAPER_SIZE)
		return imagePath, nil
	}

	log.Printf("Image size (%d bytes) exceeds Windows limit (%d bytes), compressing with JPEG 80%%...",
		fileInfo.Size(), MAX_WALLPAPER_SIZE)

	// Open original image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("error opening image for compression: %w", err)
	}
	defer file.Close()

	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("error decoding image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	log.Printf("Compressing %dx%d image to JPEG 50%% quality (keeping original resolution)", originalWidth, originalHeight)

	// Create compressed filename
	compressedPath := strings.TrimSuffix(imagePath, filepath.Ext(imagePath)) + "_compressed.jpg"

	// Create compressed file
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("error creating compressed file: %w", err)
	}
	defer compressedFile.Close()

	// Encode with 50% JPEG compression (no resizing)
	jpegOptions := &jpeg.Options{Quality: COMPRESSION_QUALITY}
	if err := jpeg.Encode(compressedFile, img, jpegOptions); err != nil {
		return "", fmt.Errorf("error encoding compressed image: %w", err)
	}

	// Check compressed file size
	compressedInfo, err := os.Stat(compressedPath)
	if err != nil {
		return "", fmt.Errorf("error getting compressed file info: %w", err)
	}

	log.Printf("Compression successful: %d bytes -> %d bytes (%.1f%% reduction)",
		fileInfo.Size(), compressedInfo.Size(),
		float64(fileInfo.Size()-compressedInfo.Size())/float64(fileInfo.Size())*100)

	return compressedPath, nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// testCompressionFunction tests the compression functionality with existing files
func (rwc *RedPaper) testCompression() error {
	// Look for any image files in the download folder to test compression
	files, err := os.ReadDir(rwc.DownloadFolder)
	if err != nil {
		return fmt.Errorf("error reading download folder: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
				imagePath := filepath.Join(rwc.DownloadFolder, file.Name())
				log.Printf("Testing compression on: %s", imagePath)

				compressedPath, err := rwc.compressImage(imagePath)
				if err != nil {
					log.Printf("Compression test failed for %s: %v", file.Name(), err)
				} else {
					log.Printf("Compression test successful for %s -> %s", file.Name(), filepath.Base(compressedPath))
				}
			}
		}
	}

	return nil
}

func main() {
	changer := NewRedPaper("wallpaper")

	// Test compression functionality first
	log.Println("Testing compression functionality...")
	if err := changer.testCompression(); err != nil {
		log.Printf("Compression test error: %v", err)
	}

	// Check for test flag
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		log.Println("Test mode: bypassing 24-hour check")
		// Force run by temporarily modifying the logic
		if err := changer.forceRun(); err != nil {
			log.Fatalf("Error: %v", err)
		}
		log.Println("Test completed successfully!")
		return
	}

	if err := changer.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Check if we actually changed the wallpaper or skipped it
	shouldRun, _, err := changer.shouldRunWallpaperChange()
	if err == nil && !shouldRun {
		log.Println("Wallpaper change skipped (24-hour check)")
		os.Exit(0) // Exit successfully but indicate no change was made
	}

	log.Println("Wallpaper changed successfully!")
}
