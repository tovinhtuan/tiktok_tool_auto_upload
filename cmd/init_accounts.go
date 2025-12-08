package main

import (
	"log"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/logger"
	sqliterepo "auto_upload_tiktok/internal/repository/sqlite"
	"auto_upload_tiktok/internal/usecase"
)

// initAccountsExample demonstrates how to create YouTube-TikTok account mappings
// Each mapping represents a job that downloads from one YouTube channel and uploads to one TikTok account
func initAccountsExample() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if _, err := logger.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logger.Close()

	db, err := sqliterepo.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error().Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	accountRepo := sqliterepo.NewAccountRepository(db)
	accountManager := usecase.NewAccountManager(accountRepo)

	// Example: Create multiple account mappings (one job per YouTube-TikTok pair)
	// Each mapping links one YouTube channel to one TikTok account
	mappings := []struct {
		name             string // Friendly name for logging
		youtubeChannelID string
		tiktokAccountID  string
		tiktokToken      string
	}{
		{
			name:             "Job 1: Tech Channel -> Tech TikTok",
			youtubeChannelID: "UCo9-zvcJyMhTEtgpSIk-L7g",                                                 // Replace with real YouTube Channel ID
			tiktokAccountID:  "7580560736729088017",                                                      // Replace with real TikTok Account ID
			tiktokToken:      "act.wFCvaQeArDwklJ9ZaluhF8GzCJQyQipenwSG628S2DmjzpWG5Yjf6c1iW3lN!6512.va", // Replace with real TikTok Access Token
		},
		// {
		// 	name:             "Job 2: Music Channel -> Music TikTok",
		// 	youtubeChannelID: "UCxxxxxxxxxxxxxxxxxxxxxxxxxx2", // Replace with real YouTube Channel ID
		// 	tiktokAccountID:  "tiktok_account_2",              // Replace with real TikTok Account ID
		// 	tiktokToken:      "tiktok_access_token_2",         // Replace with real TikTok Access Token
		// },
		// {
		// 	name:             "Job 3: Gaming Channel -> Gaming TikTok",
		// 	youtubeChannelID: "UCxxxxxxxxxxxxxxxxxxxxxxxxxx3", // Replace with real YouTube Channel ID
		// 	tiktokAccountID:  "tiktok_account_3",              // Replace with real TikTok Account ID
		// 	tiktokToken:      "tiktok_access_token_3",         // Replace with real TikTok Access Token
		// },
	}

	// Create all mappings
	for _, m := range mappings {
		account, err := accountManager.CreateAccountMapping(
			m.youtubeChannelID,
			m.tiktokAccountID,
			m.tiktokToken,
		)
		if err != nil {
			logger.Error().Printf("Failed to create %s: %v", m.name, err)
			continue
		}
		logger.Info().Printf("Created %s", m.name)
		logger.Info().Printf("   Account ID: %s", account.ID)
		logger.Info().Printf("   YouTube Channel: %s", account.YouTubeChannelID)
		logger.Info().Printf("   TikTok Account: %s", account.TikTokAccountID)
		logger.Info().Println("   Status: Active")
	}

	logger.Info().Println("All account mappings created successfully!")
	logger.Info().Println("Each mapping represents a job that will:")
	logger.Info().Println("  1. Monitor the YouTube channel for new videos")
	logger.Info().Println("  2. Download new videos automatically")
	logger.Info().Println("  3. Upload videos to the linked TikTok account")
}

// Uncomment the function call below to run this example
// func main() {
// 	initAccountsExample()
// }
