package memory

import (
	"sync"
	"time"

	"auto_upload_tiktok/internal/domain"
)

// VideoRepository is an in-memory implementation of VideoRepository
type VideoRepository struct {
	mu     sync.RWMutex
	videos map[string]*domain.Video
}

// NewVideoRepository creates a new in-memory video repository
func NewVideoRepository() *VideoRepository {
	return &VideoRepository{
		videos: make(map[string]*domain.Video),
	}
}

// GetByYouTubeID returns a video by its YouTube ID
func (r *VideoRepository) GetByYouTubeID(youtubeID string) (*domain.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, video := range r.videos {
		if video.YouTubeVideoID == youtubeID {
			return video, nil
		}
	}

	return nil, nil
}

// GetPendingVideos returns all videos with pending status
func (r *VideoRepository) GetPendingVideos(limit int) ([]*domain.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pendingVideos []*domain.Video
	for _, video := range r.videos {
		if video.Status == domain.VideoStatusPending {
			pendingVideos = append(pendingVideos, video)
			if len(pendingVideos) >= limit {
				break
			}
		}
	}

	return pendingVideos, nil
}

// CountPending returns number of pending videos
func (r *VideoRepository) CountPending() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, video := range r.videos {
		if video.Status == domain.VideoStatusPending {
			count++
		}
	}
	return count, nil
}

// Save creates or updates a video
func (r *VideoRepository) Save(video *domain.Video) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if video.ID == "" {
		video.ID = video.YouTubeVideoID
		video.CreatedAt = time.Now()
	}
	video.UpdatedAt = time.Now()

	r.videos[video.ID] = video
	return nil
}

// UpdateStatus updates the video status
func (r *VideoRepository) UpdateStatus(id string, status domain.VideoStatus, errorMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, exists := r.videos[id]
	if !exists {
		return nil
	}

	video.Status = status
	video.ErrorMessage = errorMsg
	video.UpdatedAt = time.Now()

	return nil
}

// UpdateFilePath updates the local file path
func (r *VideoRepository) UpdateFilePath(id string, filePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, exists := r.videos[id]
	if !exists {
		return nil
	}

	video.LocalFilePath = filePath
	video.UpdatedAt = time.Now()

	return nil
}

// UpdateTikTokID updates the TikTok video ID
func (r *VideoRepository) UpdateTikTokID(id string, tiktokID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, exists := r.videos[id]
	if !exists {
		return nil
	}

	video.TikTokVideoID = tiktokID
	video.UpdatedAt = time.Now()

	return nil
}

