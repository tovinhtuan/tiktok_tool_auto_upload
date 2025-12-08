package config

// Example usage of Config Manager to update YAML configuration at runtime
//
// Example 1: Load configuration
//
//	cfg, err := config.Load()
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Example 2: Update configuration and save to YAML file
//
//	manager := config.GetManager()
//
//	// Update specific fields
//	err := manager.Update(map[string]interface{}{
//		"youtube.api_key": "new_youtube_key",
//		"download.max_concurrent": 10,
//		"performance.worker_pool_size": 20,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Reload to get updated config
//	cfg, err := manager.Reload()
//
// Example 3: Update and save full configuration
//
//	manager := config.GetManager()
//	cfg := manager.Get()
//
//	// Modify config
//	cfg.MaxConcurrentDownloads = 15
//	cfg.CronSchedule = "*/3 * * * *"
//
//	// Save to YAML file
//	err := manager.Save(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Example 4: Create new manager with custom config path
//
//	manager := config.NewManager("custom_config.yaml")
//	cfg, err := manager.Load()
//	if err != nil {
//		log.Fatal(err)
//	}

