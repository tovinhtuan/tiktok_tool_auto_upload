package usecase

// Example usage of AccountManager to create YouTube-TikTok mappings
//
// Example 1: Create a new mapping
//
//	accountManager := usecase.NewAccountManager(accountRepo)
//	account, err := accountManager.CreateAccountMapping(
//		"UCxxxxxxxxxxxxxxxxxxxxxxxxxx",  // YouTube Channel ID
//		"tiktok_account_123",            // TikTok Account ID
//		"tiktok_access_token_here",      // TikTok Access Token
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Printf("Created mapping: YouTube %s -> TikTok %s", account.YouTubeChannelID, account.TikTokAccountID)
//
// Example 2: Create multiple mappings (one job per YouTube-TikTok pair)
//
//	mappings := []struct {
//		youtubeChannelID string
//		tiktokAccountID  string
//		tiktokToken      string
//	}{
//		{"UCchannel1", "tiktok1", "token1"},
//		{"UCchannel2", "tiktok2", "token2"},
//		{"UCchannel3", "tiktok3", "token3"},
//	}
//
//	for _, m := range mappings {
//		account, err := accountManager.CreateAccountMapping(
//			m.youtubeChannelID,
//			m.tiktokAccountID,
//			m.tiktokToken,
//		)
//		if err != nil {
//			log.Printf("Failed to create mapping for %s: %v", m.youtubeChannelID, err)
//			continue
//		}
//		log.Printf("Job created: YouTube %s -> TikTok %s (Account ID: %s)",
//			account.YouTubeChannelID, account.TikTokAccountID, account.ID)
//	}
//
// Example 3: Deactivate a mapping (pause a job)
//
//	err := accountManager.DeactivateAccountMapping("account_id_here")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Example 4: Reactivate a mapping (resume a job)
//
//	err := accountManager.ActivateAccountMapping("account_id_here")
//	if err != nil {
//		log.Fatal(err)
//	}

