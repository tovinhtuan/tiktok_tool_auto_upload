package memory

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"auto_upload_tiktok/internal/domain"
)

// AccountRepository is an in-memory implementation of AccountRepository
type AccountRepository struct {
	mu       sync.RWMutex
	accounts map[string]*domain.Account
}

// NewAccountRepository creates a new in-memory account repository
func NewAccountRepository() *AccountRepository {
	return &AccountRepository{
		accounts: make(map[string]*domain.Account),
	}
}

// GetAllActive returns all active accounts
func (r *AccountRepository) GetAllActive() ([]*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeAccounts []*domain.Account
	for _, account := range r.accounts {
		if account.IsActive {
			activeAccounts = append(activeAccounts, account)
		}
	}

	return activeAccounts, nil
}

// GetAll returns all accounts regardless of status.
func (r *AccountRepository) GetAll() ([]*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var accounts []*domain.Account
	for _, account := range r.accounts {
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// GetByID returns an account by its ID
func (r *AccountRepository) GetByID(id string) (*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, exists := r.accounts[id]
	if !exists {
		return nil, nil
	}

	return account, nil
}

// GetByYouTubeChannelID returns an account by YouTube channel ID
func (r *AccountRepository) GetByYouTubeChannelID(channelID string) (*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, account := range r.accounts {
		if account.YouTubeChannelID == channelID {
			return account, nil
		}
	}

	return nil, nil
}

// GetByTikTokAccountID returns an account by TikTok account ID
func (r *AccountRepository) GetByTikTokAccountID(tiktokID string) (*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, account := range r.accounts {
		if account.TikTokAccountID == tiktokID {
			return account, nil
		}
	}

	return nil, nil
}

// GetByYouTubeAndTikTok returns an account by both YouTube channel ID and TikTok account ID
func (r *AccountRepository) GetByYouTubeAndTikTok(youtubeChannelID, tiktokAccountID string) (*domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, account := range r.accounts {
		if account.YouTubeChannelID == youtubeChannelID && account.TikTokAccountID == tiktokAccountID {
			return account, nil
		}
	}

	return nil, nil
}

// Delete removes an account
func (r *AccountRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.accounts, id)
	return nil
}

// UpdateLastChecked updates the last checked timestamp and last video ID
func (r *AccountRepository) UpdateLastChecked(id string, lastVideoID string, checkedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[id]
	if !exists {
		return nil
	}

	account.LastVideoID = lastVideoID
	account.LastCheckedAt = checkedAt
	account.UpdatedAt = time.Now()

	return nil
}

// Save creates or updates an account
func (r *AccountRepository) Save(account *domain.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if account.ID == "" {
		account.ID = generateID()
		account.CreatedAt = time.Now()
	}
	account.UpdatedAt = time.Now()

	r.accounts[account.ID] = account
	return nil
}

// generateID generates a simple ID (in production, use UUID)
func generateID() string {
	return uuid.NewString()
}
