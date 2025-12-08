package tiktok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"auto_upload_tiktok/config"
	httpclient "auto_upload_tiktok/internal/infrastructure/http"
)

// Service handles TikTok API interactions
type Service struct {
	apiKey         string
	apiSecret      string
	region         string
	client         *httpclient.HTTPClient
	baseURL        string
	uploadInitPath string
	publishPath    string
}

// NewService creates a new TikTok service
func NewService(cfg *config.Config, httpClient *httpclient.HTTPClient) *Service {
	return &Service{
		apiKey:         cfg.TikTokAPIKey,
		apiSecret:      cfg.TikTokAPISecret,
		region:         cfg.TikTokRegion,
		client:         httpClient,
		baseURL:        cfg.TikTokBaseURL,
		uploadInitPath: cfg.TikTokUploadInitPath,
		publishPath:    cfg.TikTokPublishPath,
	}
}

// UploadRequest represents a video upload request
type UploadRequest struct {
	// AccessToken is the TikTok access token
	AccessToken string

	// OpenID is the TikTok user identifier associated with the access token
	OpenID string

	// VideoPath is the local path to the video file
	VideoPath string

	// Title is the video title
	Title string

	// Description is the video description
	Description string

	// PrivacyLevel sets the video privacy (PUBLIC_TO_EVERYONE, MUTUAL_FOLLOW_FRIEND, SELF_ONLY)
	PrivacyLevel string
}

// UploadResponse represents the TikTok API upload response
type UploadResponse struct {
	Data struct {
		VideoID string `json:"video_id"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// UploadVideo uploads a video to TikTok
func (s *Service) UploadVideo(req *UploadRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("upload request is nil")
	}
	if req.AccessToken == "" {
		return "", fmt.Errorf("access token is required")
	}
	if req.OpenID == "" {
		return "", fmt.Errorf("open_id is required for upload")
	}
	if req.VideoPath == "" {
		return "", fmt.Errorf("video path is required for upload")
	}

	fileInfo, err := os.Stat(req.VideoPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat video file: %w", err)
	}

	// Step 1: Initialize upload
	uploadURL, uploadID, err := s.initializeUpload(req.AccessToken, req.OpenID, fileInfo.Size())
	if err != nil {
		return "", fmt.Errorf("failed to initialize upload: %w", err)
	}

	// Step 2: Upload video file
	if err := s.uploadVideoFile(uploadURL, req.VideoPath); err != nil {
		return "", fmt.Errorf("failed to upload video file: %w", err)
	}

	// Step 3: Publish video
	videoID, err := s.publishVideo(req.AccessToken, req.OpenID, uploadID, req.Title, req.Description, req.PrivacyLevel)
	if err != nil {
		return "", fmt.Errorf("failed to publish video: %w", err)
	}

	return videoID, nil
}

// initializeUpload initializes a video upload session
func (s *Service) initializeUpload(accessToken string, openID string, videoSize int64) (string, string, error) {
	apiURL := s.combinePath(s.uploadInitPath)

	payload := map[string]any{
		"open_id":     openID,
		"upload_type": "video",
	}
	if videoSize > 0 {
		payload["video_size"] = videoSize
	}

	// TikTok API requires access_token as query parameter for POST requests
	// Add access_token to URL as query parameter
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse API URL: %w", err)
	}
	params := parsedURL.Query()
	params.Set("access_token", accessToken)
	parsedURL.RawQuery = params.Encode()
	apiURL = parsedURL.String()

	httpReq, err := s.newJSONRequest(http.MethodPost, apiURL, payload, "")
	if err != nil {
		return "", "", err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("upload init failed with status %d: %s", resp.StatusCode, previewBody(bodyBytes))
	}

	var result struct {
		Data struct {
			UploadURL string `json:"upload_url"`
			UploadID  string `json:"upload_id"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", "", fmt.Errorf("failed to decode upload init response: %w; body=%s", err, previewBody(bodyBytes))
	}

	if result.Error.Code != "" {
		return "", "", fmt.Errorf("TikTok API error: %s - %s", result.Error.Code, result.Error.Message)
	}

	return result.Data.UploadURL, result.Data.UploadID, nil
}

// uploadVideoFile uploads the video file to TikTok
func (s *Service) uploadVideoFile(uploadURL string, videoPath string) error {
	file, err := os.Open(videoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info for Content-Length
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// Create multipart form streamed through an io.Pipe to avoid loading entire file in memory
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		bufferSize := 1024 * 1024 // 1MB default buffer for throughput
		buffer := make([]byte, bufferSize)

		part, err := writer.CreateFormFile("video", fileInfo.Name())
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		if _, err := io.CopyBuffer(part, file, buffer); err != nil {
			pw.CloseWithError(err)
			return
		}

		if err := writer.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}

		pw.Close()
	}()

	// Create request with streaming body (chunked transfer)
	httpReq, err := http.NewRequest(http.MethodPost, uploadURL, pr)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// Perform upload with streaming for better performance
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// publishVideo publishes the uploaded video
func (s *Service) publishVideo(accessToken, openID, uploadID, title, description, privacyLevel string) (string, error) {
	apiURL := s.combinePath(s.publishPath)

	postInfo := map[string]any{}
	if title != "" {
		postInfo["title"] = title
	}
	if description != "" {
		postInfo["description"] = description
	}
	if privacyLevel == "" {
		privacyLevel = "PUBLIC_TO_EVERYONE"
	}
	postInfo["privacy_level"] = privacyLevel

	payload := map[string]any{
		"open_id":   openID,
		"upload_id": uploadID,
		"post_info": postInfo,
	}

	// TikTok API requires access_token as query parameter for POST requests
	// Add access_token to URL as query parameter
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse API URL: %w", err)
	}
	params := parsedURL.Query()
	params.Set("access_token", accessToken)
	parsedURL.RawQuery = params.Encode()
	apiURL = parsedURL.String()

	httpReq, err := s.newJSONRequest(http.MethodPost, apiURL, payload, "")
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("publish failed with status %d: %s", resp.StatusCode, previewBody(bodyBytes))
	}

	var result struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("failed to decode publish response: %w; body=%s", err, previewBody(bodyBytes))
	}

	if result.Error.Code != "" {
		return "", fmt.Errorf("TikTok API error: %s - %s", result.Error.Code, result.Error.Message)
	}

	return result.Data.VideoID, nil
}

// VerifyAccessToken verifies if an access token is valid
func (s *Service) VerifyAccessToken(accessToken string) (bool, error) {
	apiURL := fmt.Sprintf("%s/user/info/", s.baseURL)

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "open_id,union_id,avatar_url,display_name")

	httpReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?%s", apiURL, params.Encode()), nil)
	if err != nil {
		return false, err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// TokenResponse represents the response from TikTok token exchange
type TokenResponse struct {
	Data struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		OpenID       string `json:"open_id"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (s *Service) ExchangeCodeForToken(authCode, redirectURI string) (*TokenResponse, error) {
	apiURL := fmt.Sprintf("%s/v2/oauth/token/", s.baseURL)

	payload := map[string]string{
		"client_key":    s.apiKey,
		"client_secret": s.apiSecret,
		"code":          authCode,
		"grant_type":    "authorization_code",
		"redirect_uri":  redirectURI,
	}

	httpReq, err := s.newJSONRequest(http.MethodPost, apiURL, payload, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, previewBody(bodyBytes))
	}

	var result TokenResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w; body=%s", err, previewBody(bodyBytes))
	}

	if result.Error.Code != "" {
		return nil, fmt.Errorf("TikTok API error: %s - %s", result.Error.Code, result.Error.Message)
	}

	return &result, nil
}

// RefreshAccessToken refreshes an access token using refresh token
func (s *Service) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	apiURL := fmt.Sprintf("%s/v2/oauth/token/", s.baseURL)

	payload := map[string]string{
		"client_key":     s.apiKey,
		"client_secret":  s.apiSecret,
		"grant_type":    "refresh_token",
		"refresh_token":  refreshToken,
	}

	httpReq, err := s.newJSONRequest(http.MethodPost, apiURL, payload, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, previewBody(bodyBytes))
	}

	var result TokenResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w; body=%s", err, previewBody(bodyBytes))
	}

	if result.Error.Code != "" {
		return nil, fmt.Errorf("TikTok API error: %s - %s", result.Error.Code, result.Error.Message)
	}

	return &result, nil
}

func previewBody(body []byte) string {
	bodyStr := strings.TrimSpace(string(body))
	const limit = 512
	if len(bodyStr) > limit {
		bodyStr = bodyStr[:limit] + "..."
	}
	return bodyStr
}

func (s *Service) combinePath(path string) string {
	if path == "" {
		return s.baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(s.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (s *Service) newJSONRequest(method, url string, payload interface{}, accessToken string) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}
	return req, nil
}
