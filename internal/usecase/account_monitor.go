package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/domain"
	"auto_upload_tiktok/internal/infrastructure/youtube"
	"auto_upload_tiktok/internal/logger"
)

// AccountMonitor monitors YouTube accounts for new videos
type AccountMonitor struct {
	config            *config.Config
	accountRepo       domain.AccountRepository
	videoRepo         domain.VideoRepository
	youtubeService    *youtube.Service
	videoProcessor    *VideoProcessor // Optional: for immediate processing
	processingLimiter chan struct{}   // Controls concurrent immediate processing to avoid resource spikes
	baseCtx           context.Context // Root context for background processing
}

// NewAccountMonitor creates a new account monitor
func NewAccountMonitor(
	cfg *config.Config,
	accountRepo domain.AccountRepository,
	videoRepo domain.VideoRepository,
	youtubeService *youtube.Service,
) *AccountMonitor {
	limiterSize := cfg.WorkerPoolSize
	if limiterSize <= 0 {
		limiterSize = 1
	}

	return &AccountMonitor{
		config:            cfg,
		accountRepo:       accountRepo,
		videoRepo:         videoRepo,
		youtubeService:    youtubeService,
		processingLimiter: make(chan struct{}, limiterSize),
		baseCtx:           context.Background(),
	}
}

// SetVideoProcessor sets the video processor for immediate processing of new videos
func (m *AccountMonitor) SetVideoProcessor(processor *VideoProcessor) {
	m.videoProcessor = processor
}

// SetBaseContext configures the root context used for long-running background processing.
func (m *AccountMonitor) SetBaseContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.baseCtx = ctx
}

// MonitorAllAccounts monitors all active accounts for new videos
func (m *AccountMonitor) MonitorAllAccounts(ctx context.Context) error {
	accounts, err := m.accountRepo.GetAllActive()
	if err != nil {
		return fmt.Errorf("failed to get active accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil
	}

	// Monitor accounts concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(accounts))

	for _, account := range accounts {
		wg.Add(1)
		go func(acc *domain.Account) {
			defer wg.Done()
			if err := m.monitorAccount(ctx, acc); err != nil {
				errChan <- fmt.Errorf("failed to monitor account %s: %w", acc.ID, err)
			}
		}(account)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("monitoring errors: %v", errors)
	}

	return nil
}

// monitorAccount monitors a single account for new videos
// Each account represents a job that links one YouTube channel to one TikTok account
func (m *AccountMonitor) monitorAccount(ctx context.Context, account *domain.Account) error {
	// Log which job is running (YouTube channel -> TikTok account mapping)
	// This helps track which job is processing which pair

	// Determine the time window for logging and bootstrap filtering.
	scanSince := account.LastCheckedAt
	var bootstrapCutoff time.Time
	if scanSince.IsZero() {
		// If never checked, only consider the last 24 hours to avoid importing the entire backlog.
		bootstrapCutoff = time.Now().Add(-24 * time.Hour)
		scanSince = bootstrapCutoff
	}

	// Fetch latest videos from YouTube channel
	videos, err := m.youtubeService.GetLatestVideos(
		account.YouTubeChannelID,
		50, // Max results
	)
	if err != nil {
		return fmt.Errorf("failed to get latest videos for YouTube channel %s (TikTok account %s): %w",
			account.YouTubeChannelID, account.TikTokAccountID, err)
	}

	// Filter out videos we've already processed
	newVideos := make([]*domain.Video, 0)
	var persistedVideos []*domain.Video
	var storageErrors []error
	for _, video := range videos {
		existing, err := m.videoRepo.GetByYouTubeID(video.YouTubeVideoID)
		if err != nil {
			logger.Error().Printf("video repository lookup failed for channel %s video %s: %v",
				account.YouTubeChannelID, video.YouTubeVideoID, err)
			storageErrors = append(storageErrors, err)
			continue
		}

		if existing == nil {
			if !bootstrapCutoff.IsZero() && video.PublishedAt.Before(bootstrapCutoff) {
				// Skip older content during the initial bootstrap window.
				continue
			}

			// New video found
			video.AccountID = account.ID
			newVideos = append(newVideos, video)
		}
	}

	if len(newVideos) == 0 {
		logger.Info().Printf("No new videos detected for YouTube channel %s (TikTok account %s) since %s",
			account.YouTubeChannelID, account.TikTokAccountID, scanSince.Format(time.RFC3339))
	} else {
		logger.Info().Printf("Discovered %d new videos for YouTube channel %s (TikTok account %s); newest video ID: %s",
			len(newVideos), account.YouTubeChannelID, account.TikTokAccountID, newVideos[0].YouTubeVideoID)
	}

	// Save new videos
	if len(newVideos) > 1 {
		sort.Slice(newVideos, func(i, j int) bool {
			return newVideos[i].PublishedAt.After(newVideos[j].PublishedAt)
		})
	}
	for _, video := range newVideos {
		if err := m.videoRepo.Save(video); err != nil {
			logger.Error().Printf("failed to persist video %s for channel %s: %v", video.YouTubeVideoID, account.YouTubeChannelID, err)
			storageErrors = append(storageErrors, err)
			continue
		}
		persistedVideos = append(persistedVideos, video)
	}

	// Update account's last checked time
	lastVideoID := account.LastVideoID
	if len(persistedVideos) > 0 {
		lastVideoID = persistedVideos[0].YouTubeVideoID
	}

	if len(storageErrors) > 0 {
		return fmt.Errorf("storage errors occurred while processing account %s", account.ID)
	}

	now := time.Now()
	if err := m.accountRepo.UpdateLastChecked(account.ID, lastVideoID, now); err != nil {
		return fmt.Errorf("failed to update last checked: %w", err)
	}

	if len(persistedVideos) > 0 {
		logger.Info().Printf("Persisted %d new videos for YouTube channel %s (TikTok account %s)",
			len(persistedVideos), account.YouTubeChannelID, account.TikTokAccountID)

		// Process new videos immediately instead of waiting for schedule
		if m.videoProcessor != nil {
			logger.Info().Printf("Starting immediate processing for %d new videos from channel %s",
				len(persistedVideos), account.YouTubeChannelID)

			// Process videos in background goroutines to avoid blocking monitoring
			for _, video := range persistedVideos {
				m.launchImmediateProcessing(video)
			}
		}
	}

	return nil
}

// launchImmediateProcessing starts asynchronous processing with concurrency safeguards to avoid leaks/spikes.
func (m *AccountMonitor) launchImmediateProcessing(video *domain.Video) {
	if m.videoProcessor == nil {
		return
	}

	baseCtx := m.baseCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	go func(v *domain.Video) {
		if !m.acquireProcessingSlot(baseCtx) {
			logger.Error().Printf("Skipping immediate processing for video %s: context cancelled before slot available", v.YouTubeVideoID)
			return
		}
		defer m.releaseProcessingSlot()

		processCtx, cancel := context.WithTimeout(baseCtx, 30*time.Minute)
		defer cancel()

		if err := m.videoProcessor.ProcessVideo(processCtx, v); err != nil {
			logger.Error().Printf("Failed to process video %s immediately: %v", v.YouTubeVideoID, err)
		} else {
			logger.Info().Printf("Successfully processed video %s immediately after discovery", v.YouTubeVideoID)
		}
	}(video)
}

func (m *AccountMonitor) acquireProcessingSlot(ctx context.Context) bool {
	limiter := m.processingLimiter
	if limiter == nil {
		return true
	}

	select {
	case limiter <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (m *AccountMonitor) releaseProcessingSlot() {
	limiter := m.processingLimiter
	if limiter == nil {
		return
	}

	select {
	case <-limiter:
	default:
	}
}
