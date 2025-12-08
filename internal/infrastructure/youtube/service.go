package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/domain"
	httpclient "auto_upload_tiktok/internal/infrastructure/http"
)

// Service handles YouTube API interactions
type Service struct {
	apiKey  string
	client  *httpclient.HTTPClient
	baseURL string
}

// NewService creates a new YouTube service
func NewService(cfg *config.Config, httpClient *httpclient.HTTPClient) *Service {
	return &Service{
		apiKey:  cfg.YouTubeAPIKey,
		client:  httpClient,
		baseURL: "https://www.googleapis.com/youtube/v3",
	}
}

// VideoItem represents a video item from YouTube API
type VideoItem struct {
	ID      string `json:"id"`
	Snippet struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		PublishedAt time.Time `json:"publishedAt"`
		Thumbnails  struct {
			Default struct {
				URL string `json:"url"`
			} `json:"default"`
		} `json:"thumbnails"`
	} `json:"snippet"`
}

// SearchResponse represents the YouTube API search response
type SearchResponse struct {
	Items []VideoItem `json:"items"`
}

// GetLatestVideos fetches the latest videos from a YouTube channel
func (s *Service) GetLatestVideos(channelID string, maxResults int, publishedAfter time.Time) ([]*domain.Video, error) {
	// First, get the uploads playlist ID
	playlistID, err := s.getUploadsPlaylistID(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get uploads playlist: %w", err)
	}

	// Get videos from the uploads playlist
	videos, err := s.getPlaylistVideos(playlistID, maxResults, publishedAfter)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist videos: %w", err)
	}

	return videos, nil
}

// getUploadsPlaylistID retrieves the uploads playlist ID for a channel
func (s *Service) getUploadsPlaylistID(channelID string) (string, error) {
	apiURL := fmt.Sprintf("%s/channels", s.baseURL)
	params := url.Values{}
	params.Set("part", "contentDetails")
	params.Set("id", channelID)
	params.Set("key", s.apiKey)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?%s", apiURL, params.Encode()), nil)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("channel not found")
	}

	return result.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

// getPlaylistVideos retrieves videos from a playlist
func (s *Service) getPlaylistVideos(playlistID string, maxResults int, publishedAfter time.Time) ([]*domain.Video, error) {
	apiURL := fmt.Sprintf("%s/playlistItems", s.baseURL)
	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("playlistId", playlistID)
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	params.Set("key", s.apiKey)
	params.Set("order", "date")

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?%s", apiURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			Snippet struct {
				PublishedAt time.Time `json:"publishedAt"`
				Title       string    `json:"title"`
				Description string    `json:"description"`
				Thumbnails  struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	videos := make([]*domain.Video, 0, len(result.Items))
	for _, item := range result.Items {
		// Filter by published date
		if item.Snippet.PublishedAt.Before(publishedAfter) {
			continue
		}

		video := &domain.Video{
			ID:           item.ContentDetails.VideoID,
			YouTubeVideoID: item.ContentDetails.VideoID,
			Title:        item.Snippet.Title,
			Description:  item.Snippet.Description,
			ThumbnailURL: item.Snippet.Thumbnails.Default.URL,
			Status:       domain.VideoStatusPending,
			PublishedAt:  item.Snippet.PublishedAt,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		videos = append(videos, video)
	}

	return videos, nil
}

// DownloadVideo downloads a video from YouTube
func (s *Service) DownloadVideo(videoID string, outputPath string) error {
	// In a real implementation, you would use youtube-dl or yt-dlp
	// For this example, we'll use a simplified approach
	// Note: This requires youtube-dl or yt-dlp to be installed on the system
	
	// For production, you should use yt-dlp library or exec command
	// This is a placeholder that shows the structure
	return fmt.Errorf("download implementation required - use yt-dlp or similar tool")
}

// GetVideoDownloadURL retrieves the direct download URL for a video
func (s *Service) GetVideoDownloadURL(videoID string) (string, error) {
	// This would typically use youtube-dl or yt-dlp to get the download URL
	// For now, return an error indicating this needs implementation
	return "", fmt.Errorf("video download URL retrieval requires yt-dlp integration")
}

// DownloadVideoStream downloads a video using a streaming approach for better performance
func (s *Service) DownloadVideoStream(videoID string, outputPath string, progressChan chan<- int64) error {
	// This would implement streaming download for better memory efficiency
	// Placeholder for actual implementation
	return fmt.Errorf("streaming download implementation required")
}

