package usecase

import (
	"fmt"
	"time"

	"auto_upload_tiktok/internal/domain"
)

// AccountManager manages YouTube-TikTok account mappings
type AccountManager struct {
	accountRepo domain.AccountRepository
}

// NewAccountManager creates a new account manager
func NewAccountManager(accountRepo domain.AccountRepository) *AccountManager {
	return &AccountManager{
		accountRepo: accountRepo,
	}
}

// CreateAccountMapping creates a new mapping between YouTube channel and TikTok account
func (m *AccountManager) CreateAccountMapping(
	youtubeChannelID string,
	tiktokAccountID string,
	tiktokAccessToken string,
) (*domain.Account, error) {
	// Validate inputs
	if youtubeChannelID == "" {
		return nil, fmt.Errorf("youtube channel ID is required")
	}
	if tiktokAccountID == "" {
		return nil, fmt.Errorf("tiktok account ID is required")
	}
	if tiktokAccessToken == "" {
		return nil, fmt.Errorf("tiktok access token is required")
	}

	// Check if mapping already exists
	existing, err := m.accountRepo.GetByYouTubeAndTikTok(youtubeChannelID, tiktokAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing mapping: %w", err)
	}

	if existing != nil {
		return nil, fmt.Errorf("mapping already exists for YouTube channel %s and TikTok account %s", youtubeChannelID, tiktokAccountID)
	}

	// Check if YouTube channel is already mapped to another TikTok account
	existingByYouTube, err := m.accountRepo.GetByYouTubeChannelID(youtubeChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to check YouTube channel mapping: %w", err)
	}

	if existingByYouTube != nil {
		return nil, fmt.Errorf("YouTube channel %s is already mapped to TikTok account %s", youtubeChannelID, existingByYouTube.TikTokAccountID)
	}

	// Check if TikTok account is already mapped to another YouTube channel
	existingByTikTok, err := m.accountRepo.GetByTikTokAccountID(tiktokAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check TikTok account mapping: %w", err)
	}

	if existingByTikTok != nil {
		return nil, fmt.Errorf("TikTok account %s is already mapped to YouTube channel %s", tiktokAccountID, existingByTikTok.YouTubeChannelID)
	}

	// Create new account mapping
	account := &domain.Account{
		YouTubeChannelID:  youtubeChannelID,
		TikTokAccountID:   tiktokAccountID,
		TikTokAccessToken: tiktokAccessToken,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := m.accountRepo.Save(account); err != nil {
		return nil, fmt.Errorf("failed to save account mapping: %w", err)
	}

	return account, nil
}

// UpdateAccountMapping updates an existing account mapping
func (m *AccountManager) UpdateAccountMapping(
	accountID string,
	youtubeChannelID string,
	tiktokAccountID string,
	tiktokAccessToken string,
	isActive *bool,
) (*domain.Account, error) {
	// Get existing account
	account, err := m.accountRepo.GetByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// Update fields
	if youtubeChannelID != "" {
		account.YouTubeChannelID = youtubeChannelID
	}
	if tiktokAccountID != "" {
		account.TikTokAccountID = tiktokAccountID
	}
	if tiktokAccessToken != "" {
		account.TikTokAccessToken = tiktokAccessToken
	}
	if isActive != nil {
		account.IsActive = *isActive
	}
	account.UpdatedAt = time.Now()

	if err := m.accountRepo.Save(account); err != nil {
		return nil, fmt.Errorf("failed to update account mapping: %w", err)
	}

	return account, nil
}

// GetAccountMapping retrieves an account mapping by ID
func (m *AccountManager) GetAccountMapping(accountID string) (*domain.Account, error) {
	return m.accountRepo.GetByID(accountID)
}

// GetAllAccountMappings retrieves all account mappings
func (m *AccountManager) GetAllAccountMappings() ([]*domain.Account, error) {
	return m.accountRepo.GetAll()
}

// GetActiveAccountMappings retrieves only active account mappings
func (m *AccountManager) GetActiveAccountMappings() ([]*domain.Account, error) {
	return m.accountRepo.GetAllActive()
}

// DeleteAccountMapping removes an account mapping
func (m *AccountManager) DeleteAccountMapping(accountID string) error {
	account, err := m.accountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	return m.accountRepo.Delete(accountID)
}

// ActivateAccountMapping activates an account mapping
func (m *AccountManager) ActivateAccountMapping(accountID string) error {
	account, err := m.accountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	account.IsActive = true
	account.UpdatedAt = time.Now()

	return m.accountRepo.Save(account)
}

// DeactivateAccountMapping deactivates an account mapping
func (m *AccountManager) DeactivateAccountMapping(accountID string) error {
	account, err := m.accountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	account.IsActive = false
	account.UpdatedAt = time.Now()

	return m.accountRepo.Save(account)
}

// UpdateAccountTokens updates access token and optionally refresh token for an account
func (m *AccountManager) UpdateAccountTokens(
	accountID string,
	accessToken string,
	refreshToken string,
	expiresIn *int,
) (*domain.Account, error) {
	account, err := m.accountRepo.GetByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	if accessToken != "" {
		account.TikTokAccessToken = accessToken
	}
	if refreshToken != "" {
		account.TikTokRefreshToken = refreshToken
	}
	if expiresIn != nil && *expiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		account.TikTokTokenExpiresAt = &expiresAt
	}
	account.UpdatedAt = time.Now()

	if err := m.accountRepo.Save(account); err != nil {
		return nil, fmt.Errorf("failed to update account tokens: %w", err)
	}

	return account, nil
}
