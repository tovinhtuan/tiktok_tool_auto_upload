package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// WebUploader handles video upload via browser automation
type WebUploader struct {
	cookiesPath string
	headless    bool
}

// NewWebUploader creates a new WebUploader
func NewWebUploader(cookiesPath string, headless bool) *WebUploader {
	return &WebUploader{
		cookiesPath: cookiesPath,
		headless:    headless,
	}
}

// UploadVideo uploads a video using browser automation
func (u *WebUploader) UploadVideo(ctx context.Context, req *UploadRequest) (string, error) {
	// Create allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", u.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create browser context
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout for the entire operation
	ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// 1. Load cookies
	if err := u.loadCookies(ctx); err != nil {
		return "", fmt.Errorf("failed to load cookies: %w", err)
	}

	// 2. Navigate to upload page and upload file
	fmt.Println("[WEB UPLOAD] Navigating to upload page...")

	// Selectors
	const (
		uploadURL      = "https://www.tiktok.com/creator-center/upload?from=upload"
		iframeSelector = "iframe" // Upload often happens in an iframe
		fileInputSel   = "input[type='file']"
		captionSel     = ".notranslate.public-DraftEditor-content" // Common DraftJS editor class
		postBtnSel     = "button[data-e2e='post_video_button']"    // Common data-e2e attribute
		successModal   = ".tiktok-modal__modal-title"              // "Your video is being uploaded"
	)

	var videoID string

	// Note: TikTok's upload UI is complex and changes often.
	// This is a best-effort implementation based on common structures.
	// Real implementation might need adjustment based on actual DOM.

	err := chromedp.Run(ctx,
		chromedp.Navigate(uploadURL),
		chromedp.Sleep(5*time.Second), // Wait for page load

		// Handle file upload
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("[WEB UPLOAD] Uploading file...")
			// Ensure absolute path
			absPath, err := filepath.Abs(req.VideoPath)
			if err != nil {
				return err
			}

			// Find file input and set files
			// Note: We might need to wait for the input to be present
			return chromedp.SetUploadFiles(fileInputSel, []string{absPath}, chromedp.NodeVisible).Do(ctx)
		}),

		// Wait for upload to complete (simplified check)
		// In reality, we need to watch for progress bar or "Uploaded" status
		chromedp.Sleep(10*time.Second),

		// Set caption
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("[WEB UPLOAD] Setting caption...")
			return chromedp.SendKeys(captionSel, req.Title+" #fyp #tiktok", chromedp.NodeVisible).Do(ctx)
		}),

		chromedp.Sleep(2*time.Second),

		// Click post
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("[WEB UPLOAD] Clicking post...")
			return chromedp.Click(postBtnSel, chromedp.NodeVisible).Do(ctx)
		}),

		// Wait for success
		chromedp.Sleep(5*time.Second),
	)

	if err != nil {
		return "", fmt.Errorf("browser automation failed: %w", err)
	}

	// For now, return a placeholder ID as we can't easily extract the real ID from web
	// In a real scenario, we might parse the success URL or response
	videoID = fmt.Sprintf("web_upload_%d", time.Now().Unix())

	return videoID, nil
}

// loadCookies loads cookies from file and sets them in the browser
func (u *WebUploader) loadCookies(ctx context.Context) error {
	if u.cookiesPath == "" {
		return fmt.Errorf("cookies path is empty")
	}

	data, err := os.ReadFile(u.cookiesPath)
	if err != nil {
		return err
	}

	// Try parsing as JSON first (EditThisCookie format)
	var cookies []struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expirationDate"`
		HttpOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite"`
	}

	if err := json.Unmarshal(data, &cookies); err == nil {
		return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			for _, c := range cookies {
				// Convert SameSite string to network.CookieSameSite
				sameSite := network.CookieSameSiteLax
				switch c.SameSite {
				case "Strict":
					sameSite = network.CookieSameSiteStrict
				case "None":
					sameSite = network.CookieSameSiteNone
				}

				err := network.SetCookie(c.Name, c.Value).
					WithDomain(c.Domain).
					WithPath(c.Path).
					WithHTTPOnly(c.HttpOnly).
					WithSecure(c.Secure).
					WithSameSite(sameSite).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}))
	}

	// If JSON fails, assume Netscape format (simple parsing)
	// TODO: Implement Netscape format parsing if needed

	return nil
}

// LoginAndSaveCookies opens a browser for the user to login and saves cookies
func (u *WebUploader) LoginAndSaveCookies(ctx context.Context) error {
	// Force headless to false for interactive login
	u.headless = false

	// Create allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // Must be visible
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create browser context
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a long timeout for user to login (e.g., 5 minutes)
	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	fmt.Println("[LOGIN MODE] Opening browser...")
	fmt.Println("[LOGIN MODE] Please login to TikTok manually in the browser window.")
	fmt.Println("[LOGIN MODE] Once logged in, navigate to the upload page (https://www.tiktok.com/creator-center/upload).")
	fmt.Println("[LOGIN MODE] The tool will wait for you to reach the upload page.")

	const (
		loginURL     = "https://www.tiktok.com/login"
		uploadURL    = "https://www.tiktok.com/creator-center/upload"
		uploadURLAlt = "https://www.tiktok.com/upload"
	)

	// Navigate to login page
	if err := chromedp.Run(ctx, chromedp.Navigate(loginURL)); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// Wait for user to navigate to upload page (indicating successful login)
	// We check for the presence of the upload page URL
	fmt.Println("[LOGIN MODE] Waiting for successful login (navigation to upload page)...")

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					var url string
					if err := chromedp.Evaluate(`window.location.href`, &url).Do(ctx); err != nil {
						continue
					}

					if (len(url) >= len(uploadURL) && url[:len(uploadURL)] == uploadURL) ||
						(len(url) >= len(uploadURLAlt) && url[:len(uploadURLAlt)] == uploadURLAlt) {
						fmt.Println("[LOGIN MODE] Login detected! Saving cookies...")
						return nil
					}
				}
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("login timeout or error: %w", err)
	}

	// Get cookies
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Save cookies to file
	if err := u.saveCookiesToFile(cookies); err != nil {
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	fmt.Printf("[LOGIN MODE] Successfully saved %d cookies to %s\n", len(cookies), u.cookiesPath)
	return nil
}

// saveCookiesToFile saves cookies to the configured path in JSON format
func (u *WebUploader) saveCookiesToFile(cookies []*network.Cookie) error {
	if u.cookiesPath == "" {
		return fmt.Errorf("cookies path is empty")
	}

	// Convert to JSON-friendly format (similar to EditThisCookie)
	type CookieJSON struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expirationDate"`
		HttpOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite"`
	}

	var cookiesJSON []CookieJSON
	for _, c := range cookies {
		sameSite := "Unspecified"
		switch c.SameSite {
		case network.CookieSameSiteStrict:
			sameSite = "Strict"
		case network.CookieSameSiteLax:
			sameSite = "Lax"
		case network.CookieSameSiteNone:
			sameSite = "None"
		}

		cookiesJSON = append(cookiesJSON, CookieJSON{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			HttpOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: sameSite,
		})
	}

	data, err := json.MarshalIndent(cookiesJSON, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(u.cookiesPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(u.cookiesPath, data, 0644)
}
