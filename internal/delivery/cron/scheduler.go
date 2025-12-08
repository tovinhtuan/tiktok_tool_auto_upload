package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	cron "github.com/robfig/cron/v3"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/logger"
	"auto_upload_tiktok/internal/usecase"
)

// Scheduler manages cron jobs for the application
type Scheduler struct {
	cron           *cron.Cron
	config         *config.Config
	accountMonitor *usecase.AccountMonitor
	videoProcessor *usecase.VideoProcessor
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewScheduler creates a new cron scheduler
func NewScheduler(
	cfg *config.Config,
	accountMonitor *usecase.AccountMonitor,
	videoProcessor *usecase.VideoProcessor,
) *Scheduler {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create cron with seconds support
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:           c,
		config:         cfg,
		accountMonitor: accountMonitor,
		videoProcessor: videoProcessor,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the cron scheduler
func (s *Scheduler) Start() error {
	// Schedule account monitoring job
	monitorSchedule := normalizeSchedule(s.config.CronSchedule)
	monitorJobID, err := s.cron.AddFunc(monitorSchedule, s.monitorAccountsJob)
	if err != nil {
		return fmt.Errorf("failed to schedule monitor job: %w", err)
	}
	logger.Info().Printf("Scheduled account monitoring job with ID: %d, schedule: %s", monitorJobID, monitorSchedule)

	// Schedule video processing job (runs more frequently)
	processSchedule := normalizeSchedule("*/2 * * * *") // Every 2 minutes
	processJobID, err := s.cron.AddFunc(processSchedule, s.processVideosJob)
	if err != nil {
		return fmt.Errorf("failed to schedule process job: %w", err)
	}
	logger.Info().Printf("Scheduled video processing job with ID: %d, schedule: %s", processJobID, processSchedule)

	// Start cron
	s.cron.Start()
	logger.Info().Println("Cron scheduler started")

	// Run initial jobs immediately
	go s.monitorAccountsJob()
	go s.processVideosJob()

	return nil
}

// Stop stops the cron scheduler gracefully
func (s *Scheduler) Stop() {
	logger.Info().Println("Stopping cron scheduler...")
	s.cancel()
	s.cron.Stop()
	logger.Info().Println("Cron scheduler stopped")
}

// monitorAccountsJob is the job function for monitoring accounts
// This job scans all YouTube channels and creates video tasks for each YouTube->TikTok mapping
func (s *Scheduler) monitorAccountsJob() {
	logger.Info().Println("Starting account monitoring job...")
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	if err := s.accountMonitor.MonitorAllAccounts(ctx); err != nil {
		logger.Error().Printf("Account monitoring job failed: %v", err)
		return
	}

	duration := time.Since(startTime)
	logger.Info().Printf("Account monitoring job completed in %v (scanned all YouTube->TikTok mappings)", duration)
}

// processVideosJob is the job function for processing videos
// Each video is processed according to its account mapping (YouTube channel -> TikTok account)
func (s *Scheduler) processVideosJob() {
	logger.Info().Println("Starting video processing job...")
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()

	if err := s.videoProcessor.ProcessPendingVideos(ctx); err != nil {
		logger.Error().Printf("Video processing job failed: %v", err)
		return
	}

	duration := time.Since(startTime)
	logger.Info().Printf("Video processing job completed in %v (processed videos for all active YouTube->TikTok mappings)", duration)
}

// normalizeSchedule ensures cron expressions are compatible with cron.WithSeconds
func normalizeSchedule(expr string) string {
	fields := strings.Fields(expr)
	if len(fields) == 5 {
		return "0 " + expr
	}
	return expr
}
