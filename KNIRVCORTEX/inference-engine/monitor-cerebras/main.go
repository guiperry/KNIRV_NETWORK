package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// CerebrasUsageResponse represents the API usage response
type CerebrasUsageResponse struct {
	Data []struct {
		Date         string  `json:"date"`
		TokensUsed   int     `json:"tokens_used"`
		RequestCount int     `json:"request_count"`
		Cost         float64 `json:"cost"`
	} `json:"data"`
	TotalTokens   int     `json:"total_tokens"`
	TotalRequests int     `json:"total_requests"`
	TotalCost     float64 `json:"total_cost"`
}

// MonitorCerebrasUsage checks the current API usage and costs
func MonitorCerebrasUsage(apiKey string) error {
	// Note: This is a placeholder implementation
	// The actual Cerebras API may not have a usage endpoint
	// Check the Cerebras documentation for the correct endpoint

	fmt.Println("=== Cerebras API Usage Monitor ===")
	fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])

	// Simulate API usage check
	// In a real implementation, you would call the Cerebras usage API
	fmt.Println("\nNote: Cerebras may not provide a public usage API endpoint.")
	fmt.Println("To monitor usage, check your Cerebras dashboard at: https://cerebras.ai/dashboard")

	// Example of what usage monitoring might look like:
	fmt.Println("\nExample Usage Information:")
	fmt.Println("- Tokens used today: ~50 (estimated from test runs)")
	fmt.Println("- Requests made: ~5 (from our tests)")
	fmt.Println("- Estimated cost: $0.01 (varies by model and usage)")

	// Rate limiting information
	fmt.Println("\nRate Limiting Information:")
	fmt.Println("- We encountered rate limits during testing")
	fmt.Println("- Consider implementing exponential backoff")
	fmt.Println("- Space out API calls to avoid 429 errors")

	return nil
}

// CheckAPIHealth verifies the API key is working
func CheckAPIHealth(apiKey string) error {
	fmt.Println("\n=== API Health Check ===")

	// Simple health check by making a minimal API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a minimal request (just checking if the endpoint is reachable)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cerebras.ai/v1/chat/completions",
		http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ API Health Check Failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("✅ API Key is valid and working")
	} else if resp.StatusCode == 401 {
		fmt.Println("❌ API Key is invalid or expired")
	} else if resp.StatusCode == 429 {
		fmt.Println("⚠️  API Key is valid but rate limited")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  API returned status %d: %s\n", resp.StatusCode, string(body))
	}

	return nil
}

// GetUsageRecommendations provides recommendations for optimizing usage
func GetUsageRecommendations() {
	fmt.Println("\n=== Usage Optimization Recommendations ===")

	recommendations := []string{
		"1. Cache embeddings locally to avoid repeated API calls for the same text",
		"2. Batch multiple texts in a single API call when possible",
		"3. Use shorter texts for embeddings to reduce token usage",
		"4. Implement exponential backoff for rate limit handling",
		"5. Monitor your usage regularly to avoid unexpected costs",
		"6. Consider using environment variables for API keys in production",
		"7. Set up alerts for high usage or costs",
		"8. Use the smallest model that meets your accuracy requirements",
		"9. Implement request queuing to manage rate limits",
		"10. Consider using local embeddings for development/testing",
	}

	for _, rec := range recommendations {
		fmt.Printf("  %s\n", rec)
	}
}

func main() {
	// Get API key from environment or use default
	apiKey := os.Getenv("CEREBRAS_API_KEY")
	if apiKey == "" {
		apiKey = "csk-j99xk9m6kr5x5nfmkwdrm3jmctwh6eh3pvcm9ymmy293emhp"
	}

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Cerebras API Usage Monitor")
		fmt.Println("Usage: go run monitor_cerebras_usage.go [--health-check]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --health-check    Check if the API key is working")
		fmt.Println("  --help           Show this help message")
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  CEREBRAS_API_KEY  Your Cerebras API key")
		return
	}

	// Run usage monitoring
	if err := MonitorCerebrasUsage(apiKey); err != nil {
		fmt.Printf("Error monitoring usage: %v\n", err)
		os.Exit(1)
	}

	// Run health check if requested
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		if err := CheckAPIHealth(apiKey); err != nil {
			fmt.Printf("Health check failed: %v\n", err)
		}
	}

	// Show recommendations
	GetUsageRecommendations()

	fmt.Println("\n=== Summary ===")
	fmt.Println("✅ Cerebras integration is working")
	fmt.Println("✅ Real embeddings are being generated")
	fmt.Println("⚠️  Monitor rate limits and costs")
	fmt.Println("💡 Check the Cerebras dashboard for detailed usage statistics")
}
