package main

import (
	"fmt"
)

// calculateJobTime calculates the estimated time to complete a job
func calculateJobTime(videoSizeMB float64, networkSpeedMbps float64) {
	// Convert Mbps to MB/s (divide by 8)
	networkSpeedMBps := networkSpeedMbps / 8.0

	fmt.Printf("=== Job Performance Calculator ===\n\n")
	fmt.Printf("Video Size: %.2f MB\n", videoSizeMB)
	fmt.Printf("Network Speed: %.2f Mbps (%.2f MB/s)\n\n", networkSpeedMbps, networkSpeedMBps)

	// Step 1: Download from YouTube
	downloadTime := videoSizeMB / networkSpeedMBps
	fmt.Printf("1. Download from YouTube:\n")
	fmt.Printf("   Time: %.3f seconds (%.2f ms)\n", downloadTime, downloadTime*1000)

	// YouTube API overhead (get video info, get download URL)
	fmt.Printf("   API Overhead: ~150ms\n")
	fmt.Printf("   Total: %.3f seconds\n\n", downloadTime+0.15)

	// Step 2: Upload to TikTok
	uploadTime := videoSizeMB / networkSpeedMBps
	fmt.Printf("2. Upload to TikTok:\n")
	fmt.Printf("   Time: %.3f seconds (%.2f ms)\n", uploadTime, uploadTime*1000)

	// TikTok API overhead (initialize upload, publish)
	fmt.Printf("   API Overhead: ~500ms (initialize + publish)\n")
	fmt.Printf("   Total: %.3f seconds\n\n", uploadTime+0.5)

	// Step 3: Processing overhead
	fmt.Printf("3. Processing Overhead:\n")
	fmt.Printf("   File operations, status updates: ~100ms\n\n")

	// Total time
	totalNetworkTime := downloadTime + uploadTime
	totalAPITime := 0.15 + 0.5 // YouTube + TikTok API overhead
	totalProcessingTime := 0.1
	totalTime := totalNetworkTime + totalAPITime + totalProcessingTime

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("Network I/O (Download + Upload): %.3f seconds\n", totalNetworkTime)
	fmt.Printf("API Calls Overhead: %.3f seconds\n", totalAPITime)
	fmt.Printf("Processing Overhead: %.3f seconds\n", totalProcessingTime)
	fmt.Printf("─────────────────────────────────\n")
	fmt.Printf("TOTAL JOB TIME: %.3f seconds (%.2f ms)\n\n", totalTime, totalTime*1000)

	// Real-world considerations
	fmt.Printf("=== Real-World Considerations ===\n")
	fmt.Printf("• Network latency: +50-200ms\n")
	fmt.Printf("• Connection setup: +50-100ms\n")
	fmt.Printf("• Server processing: +100-300ms\n")
	fmt.Printf("• Buffer overhead: +10-50ms\n")
	fmt.Printf("─────────────────────────────────\n")
	estimatedRealWorld := totalTime + 0.3 // Add 300ms for real-world overhead
	fmt.Printf("ESTIMATED REAL-WORLD TIME: %.3f seconds (%.2f ms)\n", estimatedRealWorld, estimatedRealWorld*1000)
	fmt.Printf("                                ≈ %.1f seconds\n", estimatedRealWorld)
}

func main() {
	// Example: 5MB video with 170Mbps network
	calculateJobTime(5.0, 170.0)

	fmt.Println()
	fmt.Println("=== Additional Scenarios ===")
	fmt.Println()

	// Different video sizes
	sizes := []float64{1, 5, 10, 50, 100}
	for _, size := range sizes {
		networkSpeedMBps := 170.0 / 8.0
		downloadTime := size / networkSpeedMBps
		uploadTime := size / networkSpeedMBps
		totalNetworkTime := downloadTime + uploadTime
		estimatedTotal := totalNetworkTime + 0.65 + 0.3 // API + overhead
		fmt.Printf("Video %5.0f MB: ~%.2f seconds\n", size, estimatedTotal)
	}
}
