package config

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	ServerPort string `yaml:"server.port"`

	// YouTube API configuration
	YouTubeAPIKey string `yaml:"youtube.api_key"`

	// TikTok API configuration
	TikTokAPIKey         string `yaml:"tiktok.api_key"`
	TikTokAPISecret      string `yaml:"tiktok.api_secret"`
	TikTokRegion         string `yaml:"tiktok.region"`
	TikTokBaseURL        string `yaml:"tiktok.base_url"`
	TikTokUploadInitPath string `yaml:"tiktok.upload_init_path"`
	TikTokPublishPath    string `yaml:"tiktok.publish_path"`
	TikTokRedirectURI    string `yaml:"tiktok.redirect_uri"` // OAuth redirect URI
	TikTokEnableWeb      bool   `yaml:"tiktok.enable_web"`   // Enable web upload via browser automation
	TikTokCookiesPath    string `yaml:"tiktok.cookies_path"` // Path to cookies file for web upload

	// Cron schedule configuration
	CronSchedule string `yaml:"cron.schedule"`

	// Download configuration
	DownloadDir            string        `yaml:"download.dir"`
	MaxConcurrentDownloads int           `yaml:"download.max_concurrent"`
	DownloadTimeout        time.Duration `yaml:"-"`
	DownloadTimeoutStr     string        `yaml:"download.timeout"`
	YtDlpPath              string        `yaml:"download.yt_dlp_path"`
	YoutubeCookiesPath     string        `yaml:"download.youtube_cookies_path"`

	// Upload configuration
	MaxConcurrentUploads int           `yaml:"upload.max_concurrent"`
	UploadTimeout        time.Duration `yaml:"-"`
	UploadTimeoutStr     string        `yaml:"upload.timeout"`

	// Database configuration
	DatabaseURL string `yaml:"database.url"`

	// Performance tuning
	WorkerPoolSize       int           `yaml:"performance.worker_pool_size"`
	HTTPClientTimeout    time.Duration `yaml:"-"`
	HTTPClientTimeoutStr string        `yaml:"performance.http_client_timeout"`
	MaxIdleConns         int           `yaml:"performance.max_idle_conns"`
	MaxConnsPerHost      int           `yaml:"performance.max_conns_per_host"`

	// I/O optimization
	DownloadBufferSize int `yaml:"download.buffer_size"`
	UploadBufferSize   int `yaml:"upload.buffer_size"`
	MaxConcurrentIO    int `yaml:"performance.max_concurrent_io"`

	// Logging configuration
	LogDirectory  string `yaml:"logging.dir"`
	LogOutputFile string `yaml:"logging.output_file"`
	LogErrorFile  string `yaml:"logging.error_file"`

	// Bootstrap account mappings
	BootstrapAccounts []AccountBootstrap `yaml:"accounts"`
}

// AccountBootstrap defines an account mapping loaded from config
type AccountBootstrap struct {
	YouTubeChannelID  string `yaml:"youtube_channel_id"`
	TikTokAccountID   string `yaml:"tiktok_account_id"`
	TikTokAccessToken string `yaml:"tiktok_access_token"`
	IsActive          *bool  `yaml:"is_active,omitempty"`
}

// configFile represents the YAML structure
type configFile struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	YouTube struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"youtube"`
	TikTok struct {
		APIKey         string `yaml:"api_key"`
		APISecret      string `yaml:"api_secret"`
		Region         string `yaml:"region"`
		BaseURL        string `yaml:"base_url"`
		UploadInitPath string `yaml:"upload_init_path"`
		PublishPath    string `yaml:"publish_path"`
		RedirectURI    string `yaml:"redirect_uri"`
		EnableWeb      bool   `yaml:"enable_web"`
		CookiesPath    string `yaml:"cookies_path"`
	} `yaml:"tiktok"`
	Cron struct {
		Schedule string `yaml:"schedule"`
	} `yaml:"cron"`
	Download struct {
		Dir                string `yaml:"dir"`
		MaxConcurrent      int    `yaml:"max_concurrent"`
		Timeout            string `yaml:"timeout"`
		BufferSize         int    `yaml:"buffer_size"`
		YtDlpPath          string `yaml:"yt_dlp_path"`
		YoutubeCookiesPath string `yaml:"youtube_cookies_path"`
	} `yaml:"download"`
	Upload struct {
		MaxConcurrent int    `yaml:"max_concurrent"`
		Timeout       string `yaml:"timeout"`
		BufferSize    int    `yaml:"buffer_size"`
	} `yaml:"upload"`
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
	Performance struct {
		WorkerPoolSize    int    `yaml:"worker_pool_size"`
		HTTPClientTimeout string `yaml:"http_client_timeout"`
		MaxIdleConns      int    `yaml:"max_idle_conns"`
		MaxConnsPerHost   int    `yaml:"max_conns_per_host"`
		MaxConcurrentIO   int    `yaml:"max_concurrent_io"`
	} `yaml:"performance"`
	Logging struct {
		Directory  string `yaml:"dir"`
		OutputFile string `yaml:"output_file"`
		ErrorFile  string `yaml:"error_file"`
	} `yaml:"logging"`
	Accounts []struct {
		YouTubeChannelID  string `yaml:"youtube_channel_id"`
		TikTokAccountID   string `yaml:"tiktok_account_id"`
		TikTokAccessToken string `yaml:"tiktok_access_token"`
		IsActive          *bool  `yaml:"is_active"`
	} `yaml:"accounts"`
}

// Manager handles configuration loading and saving
type Manager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	if configPath == "" {
		configPath = "config.yaml"
	}
	return &Manager{
		configPath: configPath,
	}
}

// Load reads configuration from YAML file
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read YAML file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			return m.createDefaultConfig()
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfgFile configFile
	if err := yaml.Unmarshal(data, &cfgFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to Config struct
	cfg := &Config{
		ServerPort:             cfgFile.Server.Port,
		YouTubeAPIKey:          cfgFile.YouTube.APIKey,
		TikTokAPIKey:           cfgFile.TikTok.APIKey,
		TikTokAPISecret:        cfgFile.TikTok.APISecret,
		TikTokRegion:           cfgFile.TikTok.Region,
		TikTokBaseURL:          cfgFile.TikTok.BaseURL,
		TikTokUploadInitPath:   cfgFile.TikTok.UploadInitPath,
		TikTokPublishPath:      cfgFile.TikTok.PublishPath,
		TikTokRedirectURI:      cfgFile.TikTok.RedirectURI,
		TikTokEnableWeb:        cfgFile.TikTok.EnableWeb,
		TikTokCookiesPath:      cfgFile.TikTok.CookiesPath,
		CronSchedule:           cfgFile.Cron.Schedule,
		DownloadDir:            cfgFile.Download.Dir,
		MaxConcurrentDownloads: cfgFile.Download.MaxConcurrent,
		DownloadTimeoutStr:     cfgFile.Download.Timeout,
		YtDlpPath:              cfgFile.Download.YtDlpPath,
		YoutubeCookiesPath:     cfgFile.Download.YoutubeCookiesPath,
		MaxConcurrentUploads:   cfgFile.Upload.MaxConcurrent,
		UploadTimeoutStr:       cfgFile.Upload.Timeout,
		DatabaseURL:            cfgFile.Database.URL,
		WorkerPoolSize:         cfgFile.Performance.WorkerPoolSize,
		HTTPClientTimeoutStr:   cfgFile.Performance.HTTPClientTimeout,
		MaxIdleConns:           cfgFile.Performance.MaxIdleConns,
		MaxConnsPerHost:        cfgFile.Performance.MaxConnsPerHost,
		DownloadBufferSize:     cfgFile.Download.BufferSize,
		UploadBufferSize:       cfgFile.Upload.BufferSize,
		MaxConcurrentIO:        cfgFile.Performance.MaxConcurrentIO,
		LogDirectory:           cfgFile.Logging.Directory,
		LogOutputFile:          cfgFile.Logging.OutputFile,
		LogErrorFile:           cfgFile.Logging.ErrorFile,
	}

	if len(cfgFile.Accounts) > 0 {
		cfg.BootstrapAccounts = make([]AccountBootstrap, 0, len(cfgFile.Accounts))
		for _, acc := range cfgFile.Accounts {
			cfg.BootstrapAccounts = append(cfg.BootstrapAccounts, AccountBootstrap{
				YouTubeChannelID:  acc.YouTubeChannelID,
				TikTokAccountID:   acc.TikTokAccountID,
				TikTokAccessToken: acc.TikTokAccessToken,
				IsActive:          acc.IsActive,
			})
		}
	}

	// Set defaults if empty
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}
	if cfg.TikTokRegion == "" {
		cfg.TikTokRegion = "JP"
	}
	if cfg.TikTokBaseURL == "" {
		cfg.TikTokBaseURL = "https://open-api.tiktok.com"
	}
	if cfg.TikTokUploadInitPath == "" {
		cfg.TikTokUploadInitPath = "/video/upload/"
	}
	if cfg.TikTokPublishPath == "" {
		cfg.TikTokPublishPath = "/video/publish/"
	}
	if cfg.TikTokRedirectURI == "" {
		// Default to localhost callback, but can be overridden in config
		cfg.TikTokRedirectURI = fmt.Sprintf("http://localhost:%s/api/tiktok/callback", cfg.ServerPort)
		if cfg.ServerPort == "" {
			cfg.TikTokRedirectURI = "http://localhost:8080/api/tiktok/callback"
		}
	}
	if cfg.CronSchedule == "" {
		cfg.CronSchedule = "*/5 * * * *"
	}
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = "./downloads"
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "sqlite3:./data.db"
	}
	if cfg.LogDirectory == "" {
		cfg.LogDirectory = "./logs"
	}
	if cfg.LogOutputFile == "" {
		cfg.LogOutputFile = "app.log"
	}
	if cfg.LogErrorFile == "" {
		cfg.LogErrorFile = "app.error.log"
	}

	// Parse durations
	if cfg.DownloadTimeoutStr != "" {
		if d, err := time.ParseDuration(cfg.DownloadTimeoutStr); err == nil {
			cfg.DownloadTimeout = d
		} else {
			cfg.DownloadTimeout = 10 * time.Minute
		}
	} else {
		cfg.DownloadTimeout = 10 * time.Minute
	}

	if cfg.UploadTimeoutStr != "" {
		if d, err := time.ParseDuration(cfg.UploadTimeoutStr); err == nil {
			cfg.UploadTimeout = d
		} else {
			cfg.UploadTimeout = 15 * time.Minute
		}
	} else {
		cfg.UploadTimeout = 15 * time.Minute
	}

	if cfg.HTTPClientTimeoutStr != "" {
		if d, err := time.ParseDuration(cfg.HTTPClientTimeoutStr); err == nil {
			cfg.HTTPClientTimeout = d
		} else {
			cfg.HTTPClientTimeout = 30 * time.Second
		}
	} else {
		cfg.HTTPClientTimeout = 30 * time.Second
	}

	// Auto-calculate worker pool size if 0
	if cfg.WorkerPoolSize == 0 {
		cfg.WorkerPoolSize = runtime.NumCPU() * 4
		if cfg.WorkerPoolSize < 10 {
			cfg.WorkerPoolSize = 10
		}
		if cfg.WorkerPoolSize > 100 {
			cfg.WorkerPoolSize = 100
		}
	}

	// Set defaults for performance settings
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 300 // Increased from 200 for better connection pooling
	}
	if cfg.MaxConnsPerHost == 0 {
		cfg.MaxConnsPerHost = 100 // Increased from 50 for more concurrent connections
	}
	if cfg.DownloadBufferSize == 0 {
		cfg.DownloadBufferSize = 4 * 1024 * 1024 // 4MB (increased from 1MB)
	}
	if cfg.UploadBufferSize == 0 {
		cfg.UploadBufferSize = 1024 * 1024 // 1MB
	}
	if cfg.MaxConcurrentIO == 0 {
		cfg.MaxConcurrentIO = cfg.MaxConcurrentDownloads + cfg.MaxConcurrentUploads
	}

	m.config = cfg
	return cfg, nil
}

// Save writes configuration to YAML file
func (m *Manager) Save(cfg *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saveUnlocked(cfg)
}

// saveUnlocked persists config assuming caller already holds the write lock.
func (m *Manager) saveUnlocked(cfg *Config) error {
	// Convert Config to configFile
	cfgFile := configFile{
		Server: struct {
			Port string `yaml:"port"`
		}{
			Port: cfg.ServerPort,
		},
		YouTube: struct {
			APIKey string `yaml:"api_key"`
		}{
			APIKey: cfg.YouTubeAPIKey,
		},
		TikTok: struct {
			APIKey         string `yaml:"api_key"`
			APISecret      string `yaml:"api_secret"`
			Region         string `yaml:"region"`
			BaseURL        string `yaml:"base_url"`
			UploadInitPath string `yaml:"upload_init_path"`
			PublishPath    string `yaml:"publish_path"`
			RedirectURI    string `yaml:"redirect_uri"`
			EnableWeb      bool   `yaml:"enable_web"`
			CookiesPath    string `yaml:"cookies_path"`
		}{
			APIKey:         cfg.TikTokAPIKey,
			APISecret:      cfg.TikTokAPISecret,
			Region:         cfg.TikTokRegion,
			BaseURL:        cfg.TikTokBaseURL,
			UploadInitPath: cfg.TikTokUploadInitPath,
			PublishPath:    cfg.TikTokPublishPath,
			RedirectURI:    cfg.TikTokRedirectURI,
			EnableWeb:      cfg.TikTokEnableWeb,
			CookiesPath:    cfg.TikTokCookiesPath,
		},
		Cron: struct {
			Schedule string `yaml:"schedule"`
		}{
			Schedule: cfg.CronSchedule,
		},
		Download: struct {
			Dir                string `yaml:"dir"`
			MaxConcurrent      int    `yaml:"max_concurrent"`
			Timeout            string `yaml:"timeout"`
			BufferSize         int    `yaml:"buffer_size"`
			YtDlpPath          string `yaml:"yt_dlp_path"`
			YoutubeCookiesPath string `yaml:"youtube_cookies_path"`
		}{
			Dir:                cfg.DownloadDir,
			MaxConcurrent:      cfg.MaxConcurrentDownloads,
			Timeout:            cfg.DownloadTimeout.String(),
			BufferSize:         cfg.DownloadBufferSize,
			YtDlpPath:          cfg.YtDlpPath,
			YoutubeCookiesPath: cfg.YoutubeCookiesPath,
		},
		Upload: struct {
			MaxConcurrent int    `yaml:"max_concurrent"`
			Timeout       string `yaml:"timeout"`
			BufferSize    int    `yaml:"buffer_size"`
		}{
			MaxConcurrent: cfg.MaxConcurrentUploads,
			Timeout:       cfg.UploadTimeout.String(),
			BufferSize:    cfg.UploadBufferSize,
		},
		Database: struct {
			URL string `yaml:"url"`
		}{
			URL: cfg.DatabaseURL,
		},
		Performance: struct {
			WorkerPoolSize    int    `yaml:"worker_pool_size"`
			HTTPClientTimeout string `yaml:"http_client_timeout"`
			MaxIdleConns      int    `yaml:"max_idle_conns"`
			MaxConnsPerHost   int    `yaml:"max_conns_per_host"`
			MaxConcurrentIO   int    `yaml:"max_concurrent_io"`
		}{
			WorkerPoolSize:    cfg.WorkerPoolSize,
			HTTPClientTimeout: cfg.HTTPClientTimeout.String(),
			MaxIdleConns:      cfg.MaxIdleConns,
			MaxConnsPerHost:   cfg.MaxConnsPerHost,
			MaxConcurrentIO:   cfg.MaxConcurrentIO,
		},
		Logging: struct {
			Directory  string `yaml:"dir"`
			OutputFile string `yaml:"output_file"`
			ErrorFile  string `yaml:"error_file"`
		}{
			Directory:  cfg.LogDirectory,
			OutputFile: cfg.LogOutputFile,
			ErrorFile:  cfg.LogErrorFile,
		},
	}

	if len(cfg.BootstrapAccounts) > 0 {
		cfgFile.Accounts = make([]struct {
			YouTubeChannelID  string `yaml:"youtube_channel_id"`
			TikTokAccountID   string `yaml:"tiktok_account_id"`
			TikTokAccessToken string `yaml:"tiktok_access_token"`
			IsActive          *bool  `yaml:"is_active"`
		}, 0, len(cfg.BootstrapAccounts))
		for _, acc := range cfg.BootstrapAccounts {
			cfgFile.Accounts = append(cfgFile.Accounts, struct {
				YouTubeChannelID  string `yaml:"youtube_channel_id"`
				TikTokAccountID   string `yaml:"tiktok_account_id"`
				TikTokAccessToken string `yaml:"tiktok_access_token"`
				IsActive          *bool  `yaml:"is_active"`
			}{
				YouTubeChannelID:  acc.YouTubeChannelID,
				TikTokAccountID:   acc.TikTokAccountID,
				TikTokAccessToken: acc.TikTokAccessToken,
				IsActive:          acc.IsActive,
			})
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&cfgFile)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.config = cfg
	return nil
}

// Get returns the current configuration (thread-safe)
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update updates specific configuration fields and saves to file
func (m *Manager) Update(updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return fmt.Errorf("config not loaded, call Load() first")
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "server.port":
			m.config.ServerPort = value.(string)
		case "youtube.api_key":
			m.config.YouTubeAPIKey = value.(string)
		case "tiktok.api_key":
			m.config.TikTokAPIKey = value.(string)
		case "tiktok.api_secret":
			m.config.TikTokAPISecret = value.(string)
		case "tiktok.region":
			m.config.TikTokRegion = value.(string)
		case "tiktok.base_url":
			m.config.TikTokBaseURL = value.(string)
		case "tiktok.upload_init_path":
			m.config.TikTokUploadInitPath = value.(string)
		case "tiktok.publish_path":
			m.config.TikTokPublishPath = value.(string)
		case "tiktok.enable_web":
			if v, ok := value.(bool); ok {
				m.config.TikTokEnableWeb = v
			}
		case "tiktok.cookies_path":
			m.config.TikTokCookiesPath = value.(string)
		case "cron.schedule":
			m.config.CronSchedule = value.(string)
		case "download.dir":
			m.config.DownloadDir = value.(string)
		case "download.max_concurrent":
			m.config.MaxConcurrentDownloads = value.(int)
		case "download.timeout":
			if str, ok := value.(string); ok {
				m.config.DownloadTimeoutStr = str
				if d, err := time.ParseDuration(str); err == nil {
					m.config.DownloadTimeout = d
				}
			}
		case "download.buffer_size":
			m.config.DownloadBufferSize = value.(int)
		case "download.yt_dlp_path":
			if path, ok := value.(string); ok {
				m.config.YtDlpPath = path
			}
		case "upload.max_concurrent":
			m.config.MaxConcurrentUploads = value.(int)
		case "upload.timeout":
			if str, ok := value.(string); ok {
				m.config.UploadTimeoutStr = str
				if d, err := time.ParseDuration(str); err == nil {
					m.config.UploadTimeout = d
				}
			}
		case "upload.buffer_size":
			m.config.UploadBufferSize = value.(int)
		case "performance.worker_pool_size":
			m.config.WorkerPoolSize = value.(int)
		case "performance.http_client_timeout":
			if str, ok := value.(string); ok {
				m.config.HTTPClientTimeoutStr = str
				if d, err := time.ParseDuration(str); err == nil {
					m.config.HTTPClientTimeout = d
				}
			}
		case "performance.max_idle_conns":
			m.config.MaxIdleConns = value.(int)
		case "performance.max_conns_per_host":
			m.config.MaxConnsPerHost = value.(int)
		case "performance.max_concurrent_io":
			m.config.MaxConcurrentIO = value.(int)
		case "logging.dir":
			m.config.LogDirectory = value.(string)
		case "logging.output_file":
			m.config.LogOutputFile = value.(string)
		case "logging.error_file":
			m.config.LogErrorFile = value.(string)
		case "accounts":
			if accounts, ok := value.([]AccountBootstrap); ok {
				m.config.BootstrapAccounts = accounts
			}
		}
	}

	// Save to file
	return m.saveUnlocked(m.config)
}

// Reload reloads configuration from file
func (m *Manager) Reload() (*Config, error) {
	return m.Load()
}

// createDefaultConfig creates a default configuration file
func (m *Manager) createDefaultConfig() (*Config, error) {
	cfg := &Config{
		ServerPort:             "8080",
		TikTokRegion:           "JP",
		TikTokBaseURL:          "https://open-api.tiktok.com",
		TikTokUploadInitPath:   "/video/upload/",
		TikTokPublishPath:      "/video/publish/",
		CronSchedule:           "*/5 * * * *",
		DownloadDir:            "./downloads",
		DatabaseURL:            "sqlite3:./data.db",
		MaxConcurrentDownloads: 5,
		MaxConcurrentUploads:   3,
		DownloadTimeout:        10 * time.Minute,
		UploadTimeout:          15 * time.Minute,
		HTTPClientTimeout:      60 * time.Second, // Increased from 30s
		MaxIdleConns:           300,              // Increased from 200
		MaxConnsPerHost:        100,              // Increased from 50
		DownloadBufferSize:     4 * 1024 * 1024,  // 4MB instead of 1MB
		UploadBufferSize:       1024 * 1024,
		LogDirectory:           "./logs",
		LogOutputFile:          "app.log",
		LogErrorFile:           "app.error.log",
	}

	// Auto-calculate worker pool size
	cfg.WorkerPoolSize = runtime.NumCPU() * 4
	if cfg.WorkerPoolSize < 10 {
		cfg.WorkerPoolSize = 10
	}
	if cfg.WorkerPoolSize > 100 {
		cfg.WorkerPoolSize = 100
	}

	cfg.MaxConcurrentIO = cfg.MaxConcurrentDownloads + cfg.MaxConcurrentUploads

	// Save default config to file
	if err := m.saveUnlocked(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Global config manager instance
var globalManager *Manager

// Load loads configuration from YAML file (backward compatibility)
func Load() (*Config, error) {
	if globalManager == nil {
		configPath := "config.yaml"
		// Check if config/config.yaml exists, if so use it as default
		if _, err := os.Stat("config/config.yaml"); err == nil {
			configPath = "config/config.yaml"
		}
		globalManager = NewManager(configPath)
	}
	return globalManager.Load()
}

// GetManager returns the global config manager
func GetManager() *Manager {
	if globalManager == nil {
		configPath := "config.yaml"
		// Check if config/config.yaml exists, if so use it as default
		if _, err := os.Stat("config/config.yaml"); err == nil {
			configPath = "config/config.yaml"
		}
		globalManager = NewManager(configPath)
	}
	return globalManager
}
