package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"auto_upload_tiktok/internal/domain"
)

// AccountRepository is a SQLite implementation of domain.AccountRepository.
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new AccountRepository backed by SQLite.
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// GetAll returns all accounts regardless of status.
func (r *AccountRepository) GetAll() ([]*domain.Account, error) {
	rows, err := r.db.Query(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// GetAllActive returns all active accounts.
func (r *AccountRepository) GetAllActive() ([]*domain.Account, error) {
	rows, err := r.db.Query(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts WHERE is_active = 1 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// GetByID returns an account by ID.
func (r *AccountRepository) GetByID(id string) (*domain.Account, error) {
	row := r.db.QueryRow(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts WHERE id = ?`, id)
	return scanAccount(row)
}

// GetByYouTubeChannelID returns an account by YouTube channel ID.
func (r *AccountRepository) GetByYouTubeChannelID(channelID string) (*domain.Account, error) {
	row := r.db.QueryRow(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts WHERE youtube_channel_id = ?`, channelID)
	return scanAccount(row)
}

// GetByTikTokAccountID returns an account by TikTok account ID.
func (r *AccountRepository) GetByTikTokAccountID(tiktokID string) (*domain.Account, error) {
	row := r.db.QueryRow(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts WHERE tiktok_account_id = ?`, tiktokID)
	return scanAccount(row)
}

// GetByYouTubeAndTikTok returns an account by both IDs.
func (r *AccountRepository) GetByYouTubeAndTikTok(youtubeChannelID, tiktokAccountID string) (*domain.Account, error) {
	row := r.db.QueryRow(`SELECT id, youtube_channel_id, tiktok_account_id, tiktok_access_token,
		tiktok_refresh_token, tiktok_token_expires_at, last_checked_at, last_video_id, is_active, created_at, updated_at
		FROM accounts WHERE youtube_channel_id = ? AND tiktok_account_id = ?`,
		youtubeChannelID, tiktokAccountID)
	return scanAccount(row)
}

// UpdateLastChecked updates metadata about last processed video.
func (r *AccountRepository) UpdateLastChecked(id string, lastVideoID string, checkedAt time.Time) error {
	_, err := r.db.Exec(`UPDATE accounts SET last_video_id = ?, last_checked_at = ?, updated_at = ?
		WHERE id = ?`, lastVideoID, checkedAt.UTC(), time.Now().UTC(), id)
	return err
}

// Save inserts or updates an account.
func (r *AccountRepository) Save(account *domain.Account) error {
	now := time.Now().UTC()
	if account.ID == "" {
		account.ID = uuid.NewString()
		account.CreatedAt = now
	}
	account.UpdatedAt = now

	_, err := r.db.Exec(`INSERT INTO accounts
		(id, youtube_channel_id, tiktok_account_id, tiktok_access_token, tiktok_refresh_token, tiktok_token_expires_at,
		last_checked_at, last_video_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			youtube_channel_id = excluded.youtube_channel_id,
			tiktok_account_id = excluded.tiktok_account_id,
			tiktok_access_token = excluded.tiktok_access_token,
			tiktok_refresh_token = excluded.tiktok_refresh_token,
			tiktok_token_expires_at = excluded.tiktok_token_expires_at,
			last_checked_at = excluded.last_checked_at,
			last_video_id = excluded.last_video_id,
			is_active = excluded.is_active,
			updated_at = excluded.updated_at`, account.ID, account.YouTubeChannelID, account.TikTokAccountID,
		account.TikTokAccessToken, account.TikTokRefreshToken, nullableTimePtr(account.TikTokTokenExpiresAt),
		nullableTime(account.LastCheckedAt), account.LastVideoID,
		boolToInt(account.IsActive), account.CreatedAt.UTC(), account.UpdatedAt.UTC())
	return err
}

// Delete removes an account.
func (r *AccountRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func scanAccount(scanner interface {
	Scan(dest ...any) error
}) (*domain.Account, error) {
	var (
		refreshToken    sql.NullString
		tokenExpiresAt  sql.NullTime
		lastChecked     sql.NullTime
		lastVideoID     sql.NullString
		isActive        int
		account         domain.Account
	)

	if err := scanner.Scan(
		&account.ID,
		&account.YouTubeChannelID,
		&account.TikTokAccountID,
		&account.TikTokAccessToken,
		&refreshToken,
		&tokenExpiresAt,
		&lastChecked,
		&lastVideoID,
		&isActive,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if refreshToken.Valid {
		account.TikTokRefreshToken = refreshToken.String
	}
	if tokenExpiresAt.Valid {
		account.TikTokTokenExpiresAt = &tokenExpiresAt.Time
	}
	if lastChecked.Valid {
		account.LastCheckedAt = lastChecked.Time
	}
	if lastVideoID.Valid {
		account.LastVideoID = lastVideoID.String
	}
	account.IsActive = isActive == 1
	return &account, nil
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func nullableTimePtr(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
