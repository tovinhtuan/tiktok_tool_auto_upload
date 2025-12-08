package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"auto_upload_tiktok/config"
	"auto_upload_tiktok/internal/domain"
	"auto_upload_tiktok/internal/logger"
	tiktok "auto_upload_tiktok/internal/infrastructure/tiktok"
	"auto_upload_tiktok/internal/usecase"
)

// Server exposes a lightweight REST API for account management and queue visibility.
type Server struct {
	cfg            *config.Config
	accountManager *usecase.AccountManager
	videoRepo      domain.VideoRepository
	tiktokService  *tiktok.Service
	server         *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, accountManager *usecase.AccountManager, videoRepo domain.VideoRepository, tiktokService *tiktok.Service) *Server {
	mux := http.NewServeMux()
	s := &Server{
		cfg:            cfg,
		accountManager: accountManager,
		videoRepo:      videoRepo,
		tiktokService:  tiktokService,
	}

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/accounts", s.handleAccounts)
	mux.HandleFunc("/api/accounts/", s.handleAccountActions)
	mux.HandleFunc("/api/tiktok/exchange-code", s.handleExchangeCode)
	mux.HandleFunc("/api/tiktok/authorize/", s.handleAuthorize)
	mux.HandleFunc("/api/tiktok/callback", s.handleCallback)
	mux.HandleFunc("/api/videos/pending", s.handlePendingVideos)
	mux.HandleFunc("/api/videos/metrics", s.handleVideoMetrics)
	mux.HandleFunc("/", s.handleWebUI)

	s.server = &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: loggingMiddleware(mux),
	}
	return s
}

// Start begins serving HTTP requests in a separate goroutine.
func (s *Server) Start() error {
	if s.cfg.ServerPort == "" {
		return fmt.Errorf("server port is not configured")
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Printf("http api server stopped with error: %v", err)
		}
	}()
	logger.Info().Printf("HTTP API server listening on %s", s.server.Addr)
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAccounts(w, r)
	case http.MethodPost:
		s.createAccount(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleAccountActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(path, "/")
	id := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPatch:
			s.updateAccount(w, r, id)
		case http.MethodDelete:
			s.deleteAccount(w, r, id)
		default:
			methodNotAllowed(w)
		}
		return
	}

	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "activate":
			if err := s.accountManager.ActivateAccountMapping(id); err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			respondJSON(w, http.StatusOK, map[string]string{"status": "activated"})
			return
		case "deactivate":
			if err := s.accountManager.DeactivateAccountMapping(id); err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			respondJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Server) handlePendingVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			limit = parsed
		}
	}

	videos, err := s.videoRepo.GetPendingVideos(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]*videoResponse, 0, len(videos))
	for _, video := range videos {
		resp = append(resp, toVideoResponse(video))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"pending_videos": resp,
		"count":          len(resp),
	})
}

func (s *Server) handleVideoMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	count, err := s.videoRepo.CountPending()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{"pending": count})
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.accountManager.GetAllAccountMappings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]*accountResponse, 0, len(accounts))
	for _, account := range accounts {
		resp = append(resp, toAccountResponse(account))
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		YouTubeChannelID string `json:"youtube_channel_id"`
		TikTokAccountID  string `json:"tiktok_account_id"`
		TikTokToken      string `json:"tiktok_access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := s.accountManager.CreateAccountMapping(payload.YouTubeChannelID, payload.TikTokAccountID, payload.TikTokToken)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, toAccountResponse(account))
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request, id string) {
	var payload struct {
		YouTubeChannelID *string `json:"youtube_channel_id"`
		TikTokAccountID  *string `json:"tiktok_account_id"`
		TikTokToken      *string `json:"tiktok_access_token"`
		IsActive         *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := s.accountManager.GetAccountMapping(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if account == nil {
		http.NotFound(w, r)
		return
	}

	youtubeID := ""
	if payload.YouTubeChannelID != nil {
		youtubeID = *payload.YouTubeChannelID
	}
	tiktokID := ""
	if payload.TikTokAccountID != nil {
		tiktokID = *payload.TikTokAccountID
	}
	token := ""
	if payload.TikTokToken != nil {
		token = *payload.TikTokToken
	}

	updated, err := s.accountManager.UpdateAccountMapping(id, youtubeID, tiktokID, token, payload.IsActive)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, toAccountResponse(updated))
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.accountManager.DeleteAccountMapping(id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleExchangeCode exchanges TikTok authorization code for access token and automatically updates account
func (s *Server) handleExchangeCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var payload struct {
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
		AccountID    string `json:"account_id"`    // Optional: if provided, update this account
		TikTokUserID string `json:"tiktok_user_id"` // Optional: if provided, find account by TikTok user ID
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if payload.Code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	if payload.RedirectURI == "" {
		respondError(w, http.StatusBadRequest, "redirect_uri is required")
		return
	}

	// Exchange code for token
	tokenResp, err := s.tiktokService.ExchangeCodeForToken(payload.Code, payload.RedirectURI)
	if err != nil {
		logger.Error().Printf("Failed to exchange code for token: %v", err)
		respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to exchange code: %v", err))
		return
	}

	// Find account to update
	var account *domain.Account
	if payload.AccountID != "" {
		account, err = s.accountManager.GetAccountMapping(payload.AccountID)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to get account: %v", err))
			return
		}
		if account == nil {
			respondError(w, http.StatusNotFound, "account not found")
			return
		}
	} else if payload.TikTokUserID != "" {
		// Find by TikTok user ID (OpenID)
		accounts, err := s.accountManager.GetAllAccountMappings()
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get accounts: %v", err))
			return
		}
		for _, acc := range accounts {
			if acc.TikTokAccountID == payload.TikTokUserID {
				account = acc
				break
			}
		}
		if account == nil {
			respondError(w, http.StatusNotFound, "account not found with TikTok user ID")
			return
		}
	} else {
		respondError(w, http.StatusBadRequest, "either account_id or tiktok_user_id is required")
		return
	}

	// Update account with new tokens
	expiresIn := tokenResp.Data.ExpiresIn
	refreshToken := tokenResp.Data.RefreshToken
	if refreshToken == "" {
		logger.Info().Printf("WARNING: No refresh token received from TikTok API for account %s. Token will expire and need manual update.", account.ID)
	}
	
	updated, err := s.accountManager.UpdateAccountTokens(
		account.ID,
		tokenResp.Data.AccessToken,
		refreshToken,
		&expiresIn,
	)
	if err != nil {
		logger.Error().Printf("Failed to update account tokens: %v", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update tokens: %v", err))
		return
	}

	logger.Info().Printf("Successfully updated tokens for account %s via code exchange", account.ID)
	if refreshToken != "" {
		logger.Info().Printf("Refresh token saved for account %s - token will auto-refresh when expired", account.ID)
	} else {
		logger.Info().Printf("WARNING: No refresh token for account %s - token will need manual update when expired", account.ID)
	}
	
	response := map[string]interface{}{
		"status":      "success",
		"account":     toAccountResponse(updated),
		"expires_in":  tokenResp.Data.ExpiresIn,
		"token_type":  tokenResp.Data.TokenType,
		"scope":       tokenResp.Data.Scope,
		"has_refresh_token": refreshToken != "",
	}
	if refreshToken == "" {
		response["warning"] = "No refresh token received. Token will need manual update when expired."
	}
	respondJSON(w, http.StatusOK, response)
}

// handleAuthorize starts the OAuth flow by redirecting to TikTok authorization page
func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// Extract account ID from path: /api/tiktok/authorize/{account_id}
	path := strings.TrimPrefix(r.URL.Path, "/api/tiktok/authorize/")
	if path == "" {
		respondError(w, http.StatusBadRequest, "account_id is required in path")
		return
	}

	accountID := path
	account, err := s.accountManager.GetAccountMapping(accountID)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to get account: %v", err))
		return
	}
	if account == nil {
		respondError(w, http.StatusNotFound, "account not found")
		return
	}

	// Build authorization URL
	redirectURI := fmt.Sprintf("http://localhost:%s/api/tiktok/callback?account_id=%s", s.cfg.ServerPort, accountID)
	authURL := fmt.Sprintf(
		"https://www.tiktok.com/v2/auth/authorize/?client_key=%s&scope=user.info.basic,video.upload&response_type=code&redirect_uri=%s&state=%s",
		s.cfg.TikTokAPIKey,
		url.QueryEscape(redirectURI),
		accountID,
	)

	// Redirect to TikTok authorization page
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleCallback receives the OAuth callback from TikTok and automatically exchanges code for token
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// Get code and account_id from query parameters
	code := r.URL.Query().Get("code")
	accountID := r.URL.Query().Get("account_id")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		logger.Error().Printf("TikTok authorization error: %s - %s", errorParam, errorDesc)
		s.renderCallbackPage(w, false, fmt.Sprintf("Authorization failed: %s", errorDesc), accountID)
		return
	}

	if code == "" {
		respondError(w, http.StatusBadRequest, "authorization code is missing")
		return
	}

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "account_id is missing")
		return
	}

	// Get account
	account, err := s.accountManager.GetAccountMapping(accountID)
	if err != nil {
		logger.Error().Printf("Failed to get account: %v", err)
		s.renderCallbackPage(w, false, fmt.Sprintf("Failed to get account: %v", err), accountID)
		return
	}
	if account == nil {
		s.renderCallbackPage(w, false, "Account not found", accountID)
		return
	}

	// Build redirect URI (must match the one used in authorization)
	redirectURI := fmt.Sprintf("http://localhost:%s/api/tiktok/callback?account_id=%s", s.cfg.ServerPort, accountID)

	// Exchange code for token
	logger.Info().Printf("Exchanging code for token for account %s", accountID)
	tokenResp, err := s.tiktokService.ExchangeCodeForToken(code, redirectURI)
	if err != nil {
		logger.Error().Printf("Failed to exchange code for token: %v", err)
		s.renderCallbackPage(w, false, fmt.Sprintf("Failed to exchange code: %v", err), accountID)
		return
	}

	// Update account with new tokens
	expiresIn := tokenResp.Data.ExpiresIn
	refreshToken := tokenResp.Data.RefreshToken
	_, err = s.accountManager.UpdateAccountTokens(
		accountID,
		tokenResp.Data.AccessToken,
		refreshToken,
		&expiresIn,
	)
	if err != nil {
		logger.Error().Printf("Failed to update account tokens: %v", err)
		s.renderCallbackPage(w, false, fmt.Sprintf("Failed to update tokens: %v", err), accountID)
		return
	}

	logger.Info().Printf("Successfully updated tokens for account %s via OAuth callback", accountID)
	if refreshToken != "" {
		logger.Info().Printf("Refresh token saved for account %s - token will auto-refresh when expired", accountID)
	} else {
		logger.Info().Printf("WARNING: No refresh token for account %s - token will need manual update when expired", accountID)
	}

	s.renderCallbackPage(w, true, "Token updated successfully!", accountID)
}

// renderCallbackPage renders a simple HTML page to show the result
func (s *Server) renderCallbackPage(w http.ResponseWriter, success bool, message string, accountID string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	statusColor := "red"
	statusIcon := "❌"
	if success {
		statusColor = "green"
		statusIcon = "✅"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>TikTok Token Update</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			background: #f5f5f5;
		}
		.container {
			background: white;
			padding: 30px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.status {
			font-size: 24px;
			margin-bottom: 20px;
		}
		.message {
			font-size: 16px;
			margin-bottom: 20px;
			color: %s;
		}
		.info {
			background: #f0f0f0;
			padding: 15px;
			border-radius: 4px;
			margin-top: 20px;
			font-size: 14px;
		}
		.close-btn {
			background: #007bff;
			color: white;
			border: none;
			padding: 10px 20px;
			border-radius: 4px;
			cursor: pointer;
			font-size: 14px;
			margin-top: 20px;
		}
		.close-btn:hover {
			background: #0056b3;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="status">%s</div>
		<div class="message">%s</div>
		<div class="info">
			<strong>Account ID:</strong> %s<br>
			<strong>Status:</strong> %s
		</div>
		<button class="close-btn" onclick="window.close()">Close Window</button>
	</div>
</body>
</html>`, statusColor, statusIcon, message, accountID, map[bool]string{true: "Token updated successfully", false: "Update failed"}[success])

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleWebUI renders a simple web interface for token management
func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Get all accounts
	accounts, err := s.accountManager.GetAllAccountMappings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>TikTok Token Manager</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 1000px;
			margin: 20px auto;
			padding: 20px;
			background: #f5f5f5;
		}
		.container {
			background: white;
			padding: 30px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
		}
		table {
			width: 100%%;
			border-collapse: collapse;
			margin-top: 20px;
		}
		th, td {
			padding: 12px;
			text-align: left;
			border-bottom: 1px solid #ddd;
		}
		th {
			background: #f8f9fa;
			font-weight: bold;
		}
		.btn {
			background: #007bff;
			color: white;
			border: none;
			padding: 8px 16px;
			border-radius: 4px;
			cursor: pointer;
			text-decoration: none;
			display: inline-block;
			font-size: 14px;
		}
		.btn:hover {
			background: #0056b3;
		}
		.btn-success {
			background: #28a745;
		}
		.btn-success:hover {
			background: #218838;
		}
		.status-badge {
			padding: 4px 8px;
			border-radius: 4px;
			font-size: 12px;
			font-weight: bold;
		}
		.status-active {
			background: #d4edda;
			color: #155724;
		}
		.status-inactive {
			background: #f8d7da;
			color: #721c24;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🔐 TikTok Token Manager</h1>
		<p>Click "Authorize" to update token for an account. The system will automatically handle the rest.</p>
		<table>
			<thead>
				<tr>
					<th>Account ID</th>
					<th>YouTube Channel</th>
					<th>TikTok Account</th>
					<th>Status</th>
					<th>Action</th>
				</tr>
			</thead>
			<tbody>`

	for _, account := range accounts {
		statusClass := "status-active"
		statusText := "Active"
		if !account.IsActive {
			statusClass = "status-inactive"
			statusText = "Inactive"
		}

		html += fmt.Sprintf(`
				<tr>
					<td><code>%s</code></td>
					<td>%s</td>
					<td>%s</td>
					<td><span class="status-badge %s">%s</span></td>
					<td><a href="/api/tiktok/authorize/%s" class="btn btn-success">🔑 Authorize & Update Token</a></td>
				</tr>`,
			account.ID,
			account.YouTubeChannelID,
			account.TikTokAccountID,
			statusClass,
			statusText,
			account.ID,
		)
	}

	html += `
			</tbody>
		</table>
		<p style="margin-top: 30px; color: #666; font-size: 14px;">
			<strong>How it works:</strong><br>
			1. Click "Authorize & Update Token" for an account<br>
			2. You will be redirected to TikTok to authorize<br>
			3. After authorization, you'll be redirected back<br>
			4. Token will be automatically updated with refresh token<br>
			5. System will auto-refresh token when it expires
		</p>
	</div>
</body>
</html>`

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info().Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

type accountResponse struct {
	ID               string     `json:"id"`
	YouTubeChannelID string     `json:"youtube_channel_id"`
	TikTokAccountID  string     `json:"tiktok_account_id"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	LastVideoID      string     `json:"last_video_id,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toAccountResponse(account *domain.Account) *accountResponse {
	resp := &accountResponse{
		ID:               account.ID,
		YouTubeChannelID: account.YouTubeChannelID,
		TikTokAccountID:  account.TikTokAccountID,
		LastVideoID:      account.LastVideoID,
		IsActive:         account.IsActive,
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
	}
	if !account.LastCheckedAt.IsZero() {
		t := account.LastCheckedAt
		resp.LastCheckedAt = &t
	}
	return resp
}

type videoResponse struct {
	ID             string     `json:"id"`
	YouTubeVideoID string     `json:"youtube_video_id"`
	AccountID      string     `json:"account_id"`
	Status         string     `json:"status"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
}

func toVideoResponse(video *domain.Video) *videoResponse {
	resp := &videoResponse{
		ID:             video.ID,
		YouTubeVideoID: video.YouTubeVideoID,
		AccountID:      video.AccountID,
		Status:         string(video.Status),
		ErrorMessage:   video.ErrorMessage,
		CreatedAt:      video.CreatedAt,
		UpdatedAt:      video.UpdatedAt,
	}
	if !video.PublishedAt.IsZero() {
		t := video.PublishedAt
		resp.PublishedAt = &t
	}
	return resp
}
