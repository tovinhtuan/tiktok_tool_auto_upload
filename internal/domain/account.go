package domain

import "time"

// Account represents a YouTube account to monitor
type Account struct {
	// ID is the unique identifier for the account
	ID string

	// YouTubeChannelID is the YouTube channel ID to monitor
	YouTubeChannelID string

	// TikTokAccountID is the TikTok account ID where videos will be uploaded
	TikTokAccountID string

	// TikTokAccessToken is the access token for TikTok API
	TikTokAccessToken string

	// TikTokRefreshToken is the refresh token for TikTok API (optional, for auto-refresh)
	TikTokRefreshToken string

	// TikTokTokenExpiresAt is when the access token expires (optional)
	TikTokTokenExpiresAt *time.Time

	// LastCheckedAt is the timestamp of the last check for new videos
	LastCheckedAt time.Time

	// LastVideoID is the ID of the last processed video
	LastVideoID string

	// IsActive indicates if the account monitoring is active
	IsActive bool

	// CreatedAt is the timestamp when the account was created
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the account was last updated
	UpdatedAt time.Time
}

// AccountRepository defines the interface for account data operations
type AccountRepository interface {
	// GetAll returns all accounts
	GetAll() ([]*Account, error)

	// GetAllActive returns all active accounts
	GetAllActive() ([]*Account, error)

	// GetByID returns an account by its ID
	GetByID(id string) (*Account, error)

	// GetByYouTubeChannelID returns an account by YouTube channel ID
	GetByYouTubeChannelID(channelID string) (*Account, error)

	// GetByTikTokAccountID returns an account by TikTok account ID
	GetByTikTokAccountID(tiktokID string) (*Account, error)

	// GetByYouTubeAndTikTok returns an account by both YouTube channel ID and TikTok account ID
	GetByYouTubeAndTikTok(youtubeChannelID, tiktokAccountID string) (*Account, error)

	// UpdateLastChecked updates the last checked timestamp and last video ID
	UpdateLastChecked(id string, lastVideoID string, checkedAt time.Time) error

	// Save creates or updates an account
	Save(account *Account) error

	// Delete removes an account
	Delete(id string) error
}

