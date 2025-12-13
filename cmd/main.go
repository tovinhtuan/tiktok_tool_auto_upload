package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/delivery/cron"
	"auto_upload_tiktok/internal/delivery/httpapi"
	"auto_upload_tiktok/internal/domain"
	"auto_upload_tiktok/internal/infrastructure/downloader"
	httpclient "auto_upload_tiktok/internal/infrastructure/http"
	tiktok "auto_upload_tiktok/internal/infrastructure/tiktok"
	"auto_upload_tiktok/internal/infrastructure/youtube"
	"auto_upload_tiktok/internal/logger"
	sqliterepo "auto_upload_tiktok/internal/repository/sqlite"
	"auto_upload_tiktok/internal/usecase"
)

func main() {
	// Parse command line flags
	loginMode := flag.Bool("login", false, "Run in interactive login mode to save TikTok cookies")
	flag.Parse()

	// Load configuration from YAML file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if _, err := logger.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			log.Printf("Failed to close log files: %v", err)
		}
	}()

	// Handle login mode
	if *loginMode {
		handleLoginMode(cfg)
		return
	}

	// Validate required configuration
	if cfg.YouTubeAPIKey == "" {
		logger.Error().Fatal("YOUTUBE_API_KEY is required")
	}
	if cfg.TikTokAPIKey == "" {
		logger.Error().Fatal("TIKTOK_API_KEY is required")
	}
	if cfg.TikTokAPISecret == "" {
		logger.Error().Fatal("TIKTOK_API_SECRET is required")
	}

	// Initialize HTTP client
	httpClient := httpclient.NewHTTPClient(cfg)

	// Initialize persistent repositories
	db, err := sqliterepo.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error().Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	accountRepo := sqliterepo.NewAccountRepository(db)
	videoRepo := sqliterepo.NewVideoRepository(db)

	// Initialize services
	youtubeService := youtube.NewService(cfg, httpClient)
	downloadService, err := downloader.NewService(cfg, httpClient)
	if err != nil {
		logger.Error().Fatalf("Failed to create download service: %v", err)
	}
	tiktokService := tiktok.NewService(cfg, httpClient)

	// Initialize use cases
	accountManager := usecase.NewAccountManager(accountRepo)

	bootstrapAccounts(cfg, accountManager, accountRepo)
	accountMonitor := usecase.NewAccountMonitor(cfg, accountRepo, videoRepo, youtubeService)
	videoProcessor := usecase.NewVideoProcessor(
		cfg,
		videoRepo,
		accountRepo,
		youtubeService,
		downloadService,
		tiktokService,
	)

	// Set video processor in account monitor for immediate processing
	accountMonitor.SetVideoProcessor(videoProcessor)

	// Initialize and start cron scheduler
	scheduler := cron.NewScheduler(cfg, accountMonitor, videoProcessor)
	if err := scheduler.Start(); err != nil {
		logger.Error().Fatalf("Failed to start scheduler: %v", err)
	}

	// Start HTTP API server for runtime management
	apiServer := httpapi.NewServer(cfg, accountManager, videoRepo, tiktokService)
	if err := apiServer.Start(); err != nil {
		logger.Error().Fatalf("Failed to start HTTP API server: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info().Println("Application started. Press Ctrl+C to stop.")
	<-sigChan

	// Graceful shutdown
	logger.Info().Println("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	scheduler.Stop()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Printf("HTTP API shutdown error: %v", err)
	}
	logger.Info().Println("Application stopped.")
}

func bootstrapAccounts(cfg *config.Config, accountManager *usecase.AccountManager, repo domain.AccountRepository) {
	if len(cfg.BootstrapAccounts) == 0 {
		return
	}

	for _, acc := range cfg.BootstrapAccounts {
		// Validate required fields (token is optional - can be set via exchange-code API)
		if acc.YouTubeChannelID == "" || acc.TikTokAccountID == "" {
			logger.Error().Printf("Skipping invalid bootstrap mapping (missing YouTubeChannelID or TikTokAccountID): %+v", acc)
			continue
		}
		// Token is optional - if missing, we'll use placeholder and user must update via exchange-code API

		var existing *domain.Account
		var err error

		if acc.TikTokAccountID != "" {
			existing, err = repo.GetByTikTokAccountID(acc.TikTokAccountID)
			if err != nil {
				logger.Error().Printf("Failed to lookup TikTok account %s: %v", acc.TikTokAccountID, err)
				continue
			}
		}
		if existing == nil && acc.YouTubeChannelID != "" {
			existing, err = repo.GetByYouTubeChannelID(acc.YouTubeChannelID)
			if err != nil {
				logger.Error().Printf("Failed to lookup YouTube channel %s: %v", acc.YouTubeChannelID, err)
				continue
			}
		}

		if existing == nil {
			// Create account even without token - token can be set later via exchange-code API
			// But CreateAccountMapping requires a token, so we'll use a placeholder
			token := acc.TikTokAccessToken
			if token == "" {
				// Use placeholder token - user must update via exchange-code API
				token = "PLACEHOLDER_TOKEN_UPDATE_VIA_EXCHANGE_CODE_API"
				logger.Info().Printf("Creating account for channel %s without token. Token must be set via exchange-code API.", acc.YouTubeChannelID)
			}
			account, err := accountManager.CreateAccountMapping(acc.YouTubeChannelID, acc.TikTokAccountID, token)
			if err != nil {
				logger.Error().Printf("Failed to bootstrap mapping for channel %s: %v", acc.YouTubeChannelID, err)
				continue
			}
			logger.Info().Printf("Bootstrapped mapping %s -> %s (Note: Token from config has no refresh token. Use exchange-code API to get refresh token.)", acc.YouTubeChannelID, acc.TikTokAccountID)
			if acc.IsActive != nil && !*acc.IsActive {
				if err := accountManager.DeactivateAccountMapping(account.ID); err != nil {
					logger.Error().Printf("Failed to deactivate mapping for channel %s: %v", acc.YouTubeChannelID, err)
				}
			}
			continue
		}

		youtubeID := ""
		tiktokID := ""
		token := ""
		var activePtr *bool
		needsUpdate := false

		if acc.YouTubeChannelID != "" && acc.YouTubeChannelID != existing.YouTubeChannelID {
			youtubeID = acc.YouTubeChannelID
			needsUpdate = true
		}
		if acc.TikTokAccountID != "" && acc.TikTokAccountID != existing.TikTokAccountID {
			tiktokID = acc.TikTokAccountID
			needsUpdate = true
		}
		// Only update token from config if:
		// 1. Account has no token in database, OR
		// 2. Account has no refresh token (old token that can't be refreshed)
		// This prevents overwriting tokens that were updated via API exchange code
		if acc.TikTokAccessToken != "" && acc.TikTokAccessToken != existing.TikTokAccessToken {
			if existing.TikTokAccessToken == "" {
				// No token in database, use config token
				token = acc.TikTokAccessToken
				needsUpdate = true
			} else if existing.TikTokRefreshToken == "" {
				// Has token but no refresh token, update with config token
				// (but this is still not ideal - better to use exchange code API)
				logger.Info().Printf("Account %s has token but no refresh token. Consider using exchange-code API to get a refresh token instead of config token.", existing.ID)
				// Don't update from config - let user use exchange-code API instead
				// token = acc.TikTokAccessToken
				// needsUpdate = true
			} else {
				// Account has token with refresh token - don't overwrite with config
				// Token from database (obtained via API) takes precedence
				logger.Info().Printf("Account %s already has token with refresh token. Skipping token update from config. Use exchange-code API if token needs updating.", existing.ID)
			}
		}
		if acc.IsActive != nil && existing.IsActive != *acc.IsActive {
			activePtr = acc.IsActive
			needsUpdate = true
		}

		if needsUpdate {
			if _, err := accountManager.UpdateAccountMapping(existing.ID, youtubeID, tiktokID, token, activePtr); err != nil {
				logger.Error().Printf("Failed to update bootstrap mapping for channel %s: %v", existing.YouTubeChannelID, err)
			} else {
				logger.Info().Printf("Updated bootstrap mapping %s -> %s", existing.YouTubeChannelID, existing.TikTokAccountID)
			}
		}
	}
}

func handleLoginMode(cfg *config.Config) {
	logger.Info().Println("Starting interactive login mode...")

	if cfg.TikTokCookiesPath == "" {
		logger.Error().Fatal("tiktok.cookies_path is not set in config.yaml")
	}

	// Create web uploader in non-headless mode
	uploader := tiktok.NewWebUploader(cfg.TikTokCookiesPath, false)

	ctx := context.Background()
	if err := uploader.LoginAndSaveCookies(ctx); err != nil {
		logger.Error().Fatalf("Login failed: %v", err)
	}

	logger.Info().Println("Login successful! Cookies saved. You can now run the tool normally.")
}
