package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// InvidiousInstance represents a public Invidious instance
type InvidiousInstance struct {
	URL string
}

// PublicInstances lists working Invidious instances
var PublicInstances = []string{
	"https://invidious.fdn.fr",
	"https://invidious.privacydev.net",
	"https://inv.tux.pizza",
	"https://invidious.io.lol",
	"https://yt.artemislena.eu",
}

// GetVideoDownloadURL gets direct download URL from Invidious (no bot detection)
func GetVideoDownloadURL(videoID string) (string, error) {
	for _, instance := range PublicInstances {
		url := fmt.Sprintf("%s/api/v1/videos/%s", instance, videoID)

		resp, err := http.Get(url)
		if err != nil {
			continue // Try next instance
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			continue
		}

		// Get adaptive formats
		if formats, ok := data["adaptiveFormats"].([]interface{}); ok && len(formats) > 0 {
			// Find best video format
			for _, f := range formats {
				format := f.(map[string]interface{})
				if url, ok := format["url"].(string); ok {
					// Check if it's video format
					if typ, ok := format["type"].(string); ok && strings.Contains(typ, "video/mp4") {
						return url, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("failed to get download URL from all Invidious instances")
}
