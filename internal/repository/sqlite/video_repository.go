package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"auto_upload_tiktok/internal/domain"
)

// VideoRepository is a SQLite implementation of domain.VideoRepository.
type VideoRepository struct {
	db *sql.DB
}

// NewVideoRepository creates a new VideoRepository backed by SQLite.
func NewVideoRepository(db *sql.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// GetByYouTubeID returns a video by YouTube ID.
func (r *VideoRepository) GetByYouTubeID(youtubeID string) (*domain.Video, error) {
	row := r.db.QueryRow(`SELECT id, youtube_video_id, account_id, title, description, thumbnail_url,
		video_url, local_file_path, status, error_message, tiktok_video_id,
		created_at, updated_at, published_at
		FROM videos WHERE youtube_video_id = ?`, youtubeID)
	return scanVideo(row)
}

// GetPendingVideos returns pending videos up to limit ordered by oldest first.
func (r *VideoRepository) GetPendingVideos(limit int) ([]*domain.Video, error) {
	rows, err := r.db.Query(`SELECT id, youtube_video_id, account_id, title, description, thumbnail_url,
		video_url, local_file_path, status, error_message, tiktok_video_id,
		created_at, updated_at, published_at
		FROM videos WHERE status = ? ORDER BY created_at ASC LIMIT ?`, domain.VideoStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		video, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, rows.Err()
}

// CountPending returns the number of pending videos.
func (r *VideoRepository) CountPending() (int, error) {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE status = ?`, domain.VideoStatusPending)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Save inserts or updates a video.
func (r *VideoRepository) Save(video *domain.Video) error {
	now := time.Now().UTC()
	if video.ID == "" {
		video.ID = uuid.NewString()
		video.CreatedAt = now
	}
	if video.Status == "" {
		video.Status = domain.VideoStatusPending
	}
	video.UpdatedAt = now

	_, err := r.db.Exec(`INSERT INTO videos
		(id, youtube_video_id, account_id, title, description, thumbnail_url, video_url, local_file_path,
			status, error_message, tiktok_video_id, created_at, updated_at, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			youtube_video_id = excluded.youtube_video_id,
			account_id = excluded.account_id,
			title = excluded.title,
			description = excluded.description,
			thumbnail_url = excluded.thumbnail_url,
			video_url = excluded.video_url,
			local_file_path = excluded.local_file_path,
			status = excluded.status,
			error_message = excluded.error_message,
			tiktok_video_id = excluded.tiktok_video_id,
			updated_at = excluded.updated_at,
			published_at = excluded.published_at`, video.ID, video.YouTubeVideoID, video.AccountID, video.Title,
		video.Description, video.ThumbnailURL, video.VideoURL, video.LocalFilePath, string(video.Status),
		video.ErrorMessage, video.TikTokVideoID, video.CreatedAt.UTC(), video.UpdatedAt.UTC(), nullableTime(video.PublishedAt))
	return err
}

// UpdateStatus updates the status and optional error message.
func (r *VideoRepository) UpdateStatus(id string, status domain.VideoStatus, errorMsg string) error {
	_, err := r.db.Exec(`UPDATE videos SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		string(status), errorMsg, time.Now().UTC(), id)
	return err
}

// UpdateFilePath updates local file path.
func (r *VideoRepository) UpdateFilePath(id string, filePath string) error {
	_, err := r.db.Exec(`UPDATE videos SET local_file_path = ?, updated_at = ? WHERE id = ?`,
		filePath, time.Now().UTC(), id)
	return err
}

// UpdateTikTokID updates TikTok video ID.
func (r *VideoRepository) UpdateTikTokID(id string, tiktokID string) error {
	_, err := r.db.Exec(`UPDATE videos SET tiktok_video_id = ?, updated_at = ? WHERE id = ?`,
		tiktokID, time.Now().UTC(), id)
	return err
}

func scanVideo(scanner interface {
	Scan(dest ...any) error
}) (*domain.Video, error) {
	var video domain.Video
	var (
		thumbnail sql.NullString
		videoURL  sql.NullString
		localPath sql.NullString
		errorMsg  sql.NullString
		tiktokID  sql.NullString
		published sql.NullTime
	)

	if err := scanner.Scan(
		&video.ID,
		&video.YouTubeVideoID,
		&video.AccountID,
		&video.Title,
		&video.Description,
		&thumbnail,
		&videoURL,
		&localPath,
		&video.Status,
		&errorMsg,
		&tiktokID,
		&video.CreatedAt,
		&video.UpdatedAt,
		&published,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if thumbnail.Valid {
		video.ThumbnailURL = thumbnail.String
	}
	if videoURL.Valid {
		video.VideoURL = videoURL.String
	}
	if localPath.Valid {
		video.LocalFilePath = localPath.String
	}
	if errorMsg.Valid {
		video.ErrorMessage = errorMsg.String
	}
	if tiktokID.Valid {
		video.TikTokVideoID = tiktokID.String
	}
	if published.Valid {
		video.PublishedAt = published.Time
	}

	return &video, nil
}
