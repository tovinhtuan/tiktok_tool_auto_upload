package usecase

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/domain"
	"auto_upload_tiktok/internal/infrastructure/downloader"
	tiktok "auto_upload_tiktok/internal/infrastructure/tiktok"
	"auto_upload_tiktok/internal/infrastructure/youtube"
	"auto_upload_tiktok/internal/logger"
)

// VideoProcessor handles video processing workflow with optimized I/O parallelism
type VideoProcessor struct {
	config          *config.Config
	videoRepo       domain.VideoRepository
	accountRepo     domain.AccountRepository
	youtubeService  *youtube.Service
	downloadService *downloader.Service
	tiktokService   *tiktok.Service
	workerPool      chan struct{} // General worker pool
	downloadSem     chan struct{} // Semaphore for download operations
	uploadSem       chan struct{} // Semaphore for upload operations
}

// NewVideoProcessor creates a new video processor with optimized I/O parallelism
func NewVideoProcessor(
	cfg *config.Config,
	videoRepo domain.VideoRepository,
	accountRepo domain.AccountRepository,
	youtubeService *youtube.Service,
	downloadService *downloader.Service,
	tiktokService *tiktok.Service,
) *VideoProcessor {
	// Create worker pools for concurrent I/O operations
	// For I/O bound operations, we can have more concurrent operations than CPU cores
	workerPool := make(chan struct{}, cfg.WorkerPoolSize)
	downloadSem := make(chan struct{}, cfg.MaxConcurrentDownloads)
	uploadSem := make(chan struct{}, cfg.MaxConcurrentUploads)

	return &VideoProcessor{
		config:          cfg,
		videoRepo:       videoRepo,
		accountRepo:     accountRepo,
		youtubeService:  youtubeService,
		downloadService: downloadService,
		tiktokService:   tiktokService,
		workerPool:      workerPool,
		downloadSem:     downloadSem,
		uploadSem:       uploadSem,
	}
}

// ProcessPendingVideos processes all pending videos concurrently with optimized I/O parallelism
// Uses separate semaphores for download and upload to maximize I/O throughput
func (p *VideoProcessor) ProcessPendingVideos(ctx context.Context) error {
	batchSize := p.config.MaxConcurrentDownloads + p.config.MaxConcurrentUploads
	if batchSize <= 0 {
		batchSize = p.config.WorkerPoolSize
		if batchSize <= 0 {
			batchSize = 1
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		videos, err := p.videoRepo.GetPendingVideos(batchSize)
		if err != nil {
			return fmt.Errorf("failed to get pending videos: %w", err)
		}

		if len(videos) == 0 {
			return nil
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(videos))

		for _, video := range videos {
			wg.Add(1)
			go func(v *domain.Video) {
				defer wg.Done()

				// Acquire general worker slot
				p.workerPool <- struct{}{}
				defer func() { <-p.workerPool }()

				if err := p.processVideo(ctx, v); err != nil {
					errChan <- fmt.Errorf("failed to process video %s: %w", v.ID, err)
				}
			}(video)
		}

		wg.Wait()
		close(errChan)

		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}

		if len(errors) > 0 {
			return fmt.Errorf("processing errors: %v", errors)
		}
	}
}

// ProcessVideo processes a single video through the complete workflow
// This is public so it can be called immediately after video discovery
func (p *VideoProcessor) ProcessVideo(ctx context.Context, video *domain.Video) error {
	return p.processVideo(ctx, video)
}

// processVideo processes a single video through the complete workflow
func (p *VideoProcessor) processVideo(ctx context.Context, video *domain.Video) error {
	logger.Info().Printf("Processing video %s (account %s)", video.YouTubeVideoID, video.AccountID)
	// Step 1: Download video
	if err := p.downloadVideo(ctx, video); err != nil {
		p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusFailed, err.Error())
		logger.Error().Printf("Download failed for video %s: %v", video.YouTubeVideoID, err)
		return err
	}

	// Step 2: Upload to TikTok
	if err := p.uploadVideo(ctx, video); err != nil {
		p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusFailed, err.Error())
		logger.Error().Printf("Upload failed for video %s: %v", video.YouTubeVideoID, err)
		return err
	}

	// Step 3: Mark as completed
	logger.Info().Printf("Completed processing video %s (TikTok video ID: %s)", video.YouTubeVideoID, video.TikTokVideoID)
	return p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusCompleted, "")
}

// downloadVideo downloads a video from YouTube with optimized I/O parallelism
func (p *VideoProcessor) downloadVideo(ctx context.Context, video *domain.Video) error {
	// Update status to downloading
	if err := p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusDownloading, ""); err != nil {
		return err
	}
	logger.Info().Printf("Starting download for video %s (account %s)", video.YouTubeVideoID, video.AccountID)

	// Acquire download semaphore to limit concurrent downloads
	p.downloadSem <- struct{}{}
	defer func() { <-p.downloadSem }()

	// Create context with timeout
	downloadCtx, cancel := context.WithTimeout(ctx, p.config.DownloadTimeout)
	defer cancel()

	// Download video with optimized settings for I/O bound operation
	opts := downloader.DownloadOptions{
		VideoID: video.YouTubeVideoID,
		Format:  "mp4",
		Quality: "720p", // Optimize for TikTok (balance quality vs download time)
		ProgressCallback: func(progress int) {
			// Progress tracking can be logged here
		},
	}

	result, err := p.downloadService.DownloadVideo(downloadCtx, opts)
	if err != nil {
		logger.Error().Printf("Download failed for video %s: %v", video.YouTubeVideoID, err)
		return fmt.Errorf("download failed: %w", err)
	}

	// Update video with file path
	if err := p.videoRepo.UpdateFilePath(video.ID, result.FilePath); err != nil {
		return err
	}
	video.LocalFilePath = result.FilePath

	// Update status to downloaded
	if err := p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusDownloaded, ""); err != nil {
		return err
	}
	logger.Info().Printf("Download completed for video %s -> %s", video.YouTubeVideoID, result.FilePath)
	return nil
}

// uploadVideo uploads a video to TikTok with optimized I/O parallelism
// Each video is linked to an account which maps YouTube channel -> TikTok account
func (p *VideoProcessor) uploadVideo(ctx context.Context, video *domain.Video) error {
	// Get account mapping (YouTube channel -> TikTok account) for this video
	account, err := p.accountRepo.GetByID(video.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account mapping: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account mapping not found for video %s (account ID: %s)", video.ID, video.AccountID)
	}

	// Validate that account has TikTok credentials
	if account.TikTokAccountID == "" {
		return fmt.Errorf("TikTok account ID not configured for account %s", account.ID)
	}

	// If Web Upload is enabled, we skip API token validation
	if !p.config.TikTokEnableWeb {
		if account.TikTokAccessToken == "" {
			authorizeURL := p.promptManualAuthorization(account.ID)
			return fmt.Errorf("TikTok access token not configured for account %s. Re-authorize via %s and exchange the returned code for a token", account.ID, authorizeURL)
		}

		// Validate and refresh access token if needed
		logger.Info().Printf("Validating TikTok access token for account %s", account.ID)
		isValid, err := p.tiktokService.VerifyAccessToken(account.TikTokAccessToken)
		if err != nil {
			logger.Error().Printf("Failed to verify access token for account %s: %v", account.ID, err)
			return fmt.Errorf("failed to verify access token: %w", err)
		}
		if !isValid {
			logger.Info().Printf("Access token is invalid or expired for account %s, attempting to refresh", account.ID)

			// Try to refresh token if refresh token is available
			if account.TikTokRefreshToken != "" {
				logger.Info().Printf("Attempting to refresh access token for account %s", account.ID)
				tokenResp, err := p.tiktokService.RefreshAccessToken(account.TikTokRefreshToken)
				if err != nil {
					logger.Error().Printf("Failed to refresh access token for account %s: %v", account.ID, err)
					return fmt.Errorf("TikTok access token is invalid and refresh failed for account %s: %w. Please update the token", account.ID, err)
				}

				// Update account with new tokens
				account.TikTokAccessToken = tokenResp.Data.AccessToken
				if tokenResp.Data.RefreshToken != "" {
					account.TikTokRefreshToken = tokenResp.Data.RefreshToken
				}
				if tokenResp.Data.ExpiresIn > 0 {
					expiresAt := time.Now().Add(time.Duration(tokenResp.Data.ExpiresIn) * time.Second)
					account.TikTokTokenExpiresAt = &expiresAt
				}

				// Save updated account
				if err := p.accountRepo.Save(account); err != nil {
					logger.Error().Printf("Failed to save refreshed token for account %s: %v", account.ID, err)
					return fmt.Errorf("failed to save refreshed token: %w", err)
				}

				logger.Info().Printf("Successfully refreshed access token for account %s", account.ID)
			} else {
				logger.Error().Printf("Access token is invalid or expired for account %s and no refresh token available", account.ID)
				authorizeURL := p.promptManualAuthorization(account.ID)
				return fmt.Errorf("TikTok access token is invalid or expired for account %s and no refresh token available. Re-authorize via %s and exchange the returned code for a new token", account.ID, authorizeURL)
			}
		}
		logger.Info().Printf("Access token validated successfully for account %s", account.ID)
	} else {
		logger.Info().Printf("Web upload enabled, skipping API token validation for account %s", account.ID)
	}

	// Update status to uploading
	if err := p.videoRepo.UpdateStatus(video.ID, domain.VideoStatusUploading, ""); err != nil {
		return err
	}
	logger.Info().Printf("Starting upload for video %s (account %s)", video.YouTubeVideoID, account.ID)

	// Acquire upload semaphore to limit concurrent uploads
	p.uploadSem <- struct{}{}
	defer func() { <-p.uploadSem }()

	// Create upload request for the specific TikTok account
	// Job context: Uploading video from YouTube channel %s to TikTok account %s
	uploadReq := &tiktok.UploadRequest{
		AccessToken:  account.TikTokAccessToken,
		OpenID:       account.TikTokAccountID,
		VideoPath:    video.LocalFilePath,
		Title:        video.Title,
		Description:  video.Description,
		PrivacyLevel: "PUBLIC_TO_EVERYONE",
	}

	// Perform upload to the linked TikTok account
	// Each job uploads to its specific TikTok account
	tiktokVideoID, err := p.tiktokService.UploadVideo(uploadReq)
	if err != nil {
		logger.Error().Printf("Upload failed for video %s: %v", video.YouTubeVideoID, err)
		return fmt.Errorf("upload failed: %w", err)
	}

	// Update video with TikTok ID
	if err := p.videoRepo.UpdateTikTokID(video.ID, tiktokVideoID); err != nil {
		return err
	}
	logger.Info().Printf("Upload completed for video %s -> TikTok video %s", video.YouTubeVideoID, tiktokVideoID)

	return nil
}

// promptManualAuthorization logs instructions for manually re-authorizing a TikTok account and returns the authorize URL.
func (p *VideoProcessor) promptManualAuthorization(accountID string) string {
	scopes := "user.info.basic,video.upload,video.publish"
	state := "12345"
	authorizeURL := fmt.Sprintf(
		"https://www.tiktok.com/v2/auth/authorize?client_key=%s&scope=%s&response_type=code&redirect_uri=%s&state=%s",
		url.QueryEscape(p.config.TikTokAPIKey),
		url.QueryEscape(scopes),
		url.QueryEscape(p.config.TikTokRedirectURI),
		url.QueryEscape(state),
	)

	logger.Error().Printf("To re-authorize TikTok account %s open: %s", accountID, authorizeURL)
	logger.Error().Printf("After login TikTok will redirect to %s with ?code=NEW_CODE", p.config.TikTokRedirectURI)
	logger.Error().Printf("Call https://open.tiktokapis.com/v2/oauth/token/ (or POST /api/tiktok/exchange-code) with client_key, client_secret, redirect_uri=%s and code=NEW_CODE to store the new access/refresh tokens", p.config.TikTokRedirectURI)

	return authorizeURL
}
