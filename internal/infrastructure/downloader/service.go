package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"auto_upload_tiktok/config"
	httpclient "auto_upload_tiktok/internal/infrastructure/http"
	"auto_upload_tiktok/internal/logger"
)

// Service handles video downloading with high performance
type Service struct {
	config      *config.Config
	httpClient  *httpclient.HTTPClient
	downloadDir string
	ytDlpPath   string
}

// NewService creates a new download service
func NewService(cfg *config.Config, httpClient *httpclient.HTTPClient) (*Service, error) {
	// Ensure download directory exists
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	ytDlpPath, err := resolveYtDlpPath(cfg)
	if err != nil {
		return nil, err
	}

	return &Service{
		config:      cfg,
		httpClient:  httpClient,
		downloadDir: cfg.DownloadDir,
		ytDlpPath:   ytDlpPath,
	}, nil
}

// DownloadOptions contains options for video download
type DownloadOptions struct {
	// VideoID is the YouTube video ID
	VideoID string

	// Format is the desired video format (mp4, webm, etc.)
	Format string

	// Quality is the desired video quality (best, worst, 720p, etc.)
	Quality string

	// ProgressCallback is called with download progress (0-100)
	ProgressCallback func(progress int)
}

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	// FilePath is the path to the downloaded file
	FilePath string

	// FileSize is the size of the downloaded file in bytes
	FileSize int64

	// Duration is the time taken to download
	Duration time.Duration
}

// DownloadVideo downloads a video using yt-dlp for high performance
func (s *Service) DownloadVideo(ctx context.Context, opts DownloadOptions) (*DownloadResult, error) {
	startTime := time.Now()
	outputPath := filepath.Join(s.downloadDir, fmt.Sprintf("%s.%%(ext)s", opts.VideoID))

	// Log yt-dlp path for debugging
	logger.Info().Printf("Using yt-dlp at: %s", s.ytDlpPath)

	// Build yt-dlp command with options to bypass YouTube bot detection
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--no-check-certificates",
		// Use tv_embedded client - least likely to be blocked
		"--extractor-args", "youtube:player_client=tv_embedded",
		// Skip problematic formats
		"--extractor-args", "youtube:skip=hls,dash",
		// Add user-agent
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		// Prefer IPv4 (some servers have IPv6 issues)
		"--force-ipv4",
		// Retry on failures
		"--retries", "3",
		// Add delay between retries to avoid rate limiting
		"--retry-sleep", "3",
	}

	// Add cookies if available (helps bypass bot detection significantly)
	cookiesPath := s.config.YoutubeCookiesPath
	if cookiesPath == "" {
		// Try default paths
		cookiesPath = "./youtube_cookies.txt"
	}
	if _, err := os.Stat(cookiesPath); err == nil {
		logger.Info().Printf("Using YouTube cookies from: %s", cookiesPath)
		args = append(args, "--cookies", cookiesPath)
	} else {
		logger.Warn().Printf("YouTube cookies not found at %s - may encounter bot detection", cookiesPath)
	}

	args = append(args, "-o", outputPath)

	// Add format options
	if opts.Format != "" {
		args = append(args, "-f", opts.Format)
	} else if opts.Quality != "" {
		args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", opts.Quality, opts.Quality))
	} else {
		// Default: best quality mp4
		args = append(args, "-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]")
	}

	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", opts.VideoID)
	args = append(args, videoURL)

	// Log command for debugging
	logger.Info().Printf("Executing: %s %s", s.ytDlpPath, strings.Join(args, " "))

	// Execute yt-dlp
	cmd := exec.CommandContext(ctx, s.ytDlpPath, args...)

	// Capture output for better error logging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Log stderr for debugging
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("yt-dlp download failed: %w\nStderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("yt-dlp download failed: %w", err)
	}

	// Find the downloaded file
	pattern := filepath.Join(s.downloadDir, fmt.Sprintf("%s.*", opts.VideoID))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("downloaded file not found")
	}

	filePath := matches[0]
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	// Rename to .mp4 if needed
	if filepath.Ext(filePath) != ".mp4" {
		newPath := filepath.Join(s.downloadDir, fmt.Sprintf("%s.mp4", opts.VideoID))
		if err := os.Rename(filePath, newPath); err != nil {
			return nil, fmt.Errorf("failed to rename file: %w", err)
		}
		filePath = newPath
	}

	duration := time.Since(startTime)

	return &DownloadResult{
		FilePath: filePath,
		FileSize: fileInfo.Size(),
		Duration: duration,
	}, nil
}

// monitorProgress monitors download progress from yt-dlp output
func (s *Service) monitorProgress(stdout, stderr io.ReadCloser, callback func(int)) {
	if callback == nil {
		return
	}

	// Read from stderr (yt-dlp outputs progress to stderr)
	buf := make([]byte, 1024)
	for {
		n, err := stderr.Read(buf)
		if err != nil {
			break
		}

		// Parse progress from output
		// yt-dlp format: [download]  45.2% of 123.45MiB at 5.67MiB/s ETA 00:12
		output := string(buf[:n])
		// Simple progress extraction (in production, use regex)
		_ = output // Placeholder for progress parsing
	}
}

// DownloadVideoStream downloads a video using streaming for better memory efficiency
func (s *Service) DownloadVideoStream(ctx context.Context, videoURL string, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Stream download with optimized buffer size for I/O bound operations
	// Larger buffer reduces system calls and improves throughput
	bufferSize := 1024 * 1024 // 1MB default, configurable via config
	if s.config != nil && s.config.DownloadBufferSize > 0 {
		bufferSize = s.config.DownloadBufferSize
	}
	buffer := make([]byte, bufferSize)
	_, err = io.CopyBuffer(file, resp.Body, buffer)
	return err
}

// CleanupOldDownloads removes old downloaded files
func (s *Service) CleanupOldDownloads(maxAge time.Duration) error {
	entries, err := os.ReadDir(s.downloadDir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(s.downloadDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}

// resolveYtDlpPath determines the path to the yt-dlp executable.
func resolveYtDlpPath(cfg *config.Config) (string, error) {
	// Helper that validates a candidate path.
	checkCandidate := func(candidate string) (string, bool) {
		if candidate == "" {
			return "", false
		}

		// If candidate contains a path separator, treat it as a direct path.
		if strings.ContainsAny(candidate, `/\`) {
			full := candidate
			if !filepath.IsAbs(full) {
				full = filepath.Clean(full)
			}
			if info, err := os.Stat(full); err == nil && !info.IsDir() {
				return full, true
			}
			return "", false
		}

		// Otherwise, ask the OS to resolve it inside PATH.
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, true
		}
		return "", false
	}

	if cfg != nil && cfg.YtDlpPath != "" {
		if resolved, ok := checkCandidate(cfg.YtDlpPath); ok {
			return resolved, nil
		}
		return "", fmt.Errorf("configured download.yt_dlp_path %q does not point to a valid yt-dlp binary", cfg.YtDlpPath)
	}

	// Default candidates: PATH plus common local locations.
	if resolved, ok := checkCandidate("yt-dlp"); ok {
		return resolved, nil
	}
	if runtime.GOOS == "windows" {
		if resolved, ok := checkCandidate("yt-dlp.exe"); ok {
			return resolved, nil
		}
	}

	wd, _ := os.Getwd()
	potentialDirs := []string{
		wd,
		filepath.Join(wd, "bin"),
		filepath.Join(wd, "cmd"),
		filepath.Join(wd, "cmd", "bin"),
		filepath.Join("bin"),
		filepath.Join("cmd", "bin"),
	}

	binaryName := "yt-dlp"
	if runtime.GOOS == "windows" {
		binaryName = "yt-dlp.exe"
	}

	for _, dir := range potentialDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, binaryName)
		if resolved, ok := checkCandidate(candidate); ok {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("yt-dlp executable not found. Please install yt-dlp (https://github.com/yt-dlp/yt-dlp), add it to PATH, or set download.yt_dlp_path in config.yaml")
}
