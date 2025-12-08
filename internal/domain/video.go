package domain

import "time"

// VideoStatus represents the processing status of a video
type VideoStatus string

const (
	// VideoStatusPending indicates the video is waiting to be processed
	VideoStatusPending VideoStatus = "pending"

	// VideoStatusDownloading indicates the video is currently being downloaded
	VideoStatusDownloading VideoStatus = "downloading"

	// VideoStatusDownloaded indicates the video has been downloaded
	VideoStatusDownloaded VideoStatus = "downloaded"

	// VideoStatusUploading indicates the video is currently being uploaded
	VideoStatusUploading VideoStatus = "uploading"

	// VideoStatusCompleted indicates the video has been successfully uploaded
	VideoStatusCompleted VideoStatus = "completed"

	// VideoStatusFailed indicates the video processing failed
	VideoStatusFailed VideoStatus = "failed"
)

// Video represents a video that needs to be processed
type Video struct {
	// ID is the unique identifier for the video
	ID string

	// YouTubeVideoID is the YouTube video ID
	YouTubeVideoID string

	// AccountID is the associated account ID
	AccountID string

	// Title is the video title
	Title string

	// Description is the video description
	Description string

	// ThumbnailURL is the URL of the video thumbnail
	ThumbnailURL string

	// VideoURL is the URL of the video file
	VideoURL string

	// LocalFilePath is the local path where the video is downloaded
	LocalFilePath string

	// Status is the current processing status
	Status VideoStatus

	// ErrorMessage contains error details if processing failed
	ErrorMessage string

	// TikTokVideoID is the TikTok video ID after upload
	TikTokVideoID string

	// CreatedAt is the timestamp when the video was created
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the video was last updated
	UpdatedAt time.Time

	// PublishedAt is the timestamp when the video was published on YouTube
	PublishedAt time.Time
}

// VideoRepository defines the interface for video data operations
type VideoRepository interface {
	// GetByYouTubeID returns a video by its YouTube ID
	GetByYouTubeID(youtubeID string) (*Video, error)

	// GetPendingVideos returns all videos with pending status
	GetPendingVideos(limit int) ([]*Video, error)

	// CountPending returns the total number of pending videos
	CountPending() (int, error)

	// Save creates or updates a video
	Save(video *Video) error

	// UpdateStatus updates the video status
	UpdateStatus(id string, status VideoStatus, errorMsg string) error

	// UpdateFilePath updates the local file path
	UpdateFilePath(id string, filePath string) error

	// UpdateTikTokID updates the TikTok video ID
	UpdateTikTokID(id string, tiktokID string) error
}

