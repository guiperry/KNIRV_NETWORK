// Option package for KNIRV Gateway SDK configuration

package option

import (
	"net/http"
	"time"

	"github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway/internal/requestconfig"
)

// RequestOption is a function that modifies a request configuration
type RequestOption func(*requestconfig.RequestConfig)

// WithBaseURL sets the base URL for API requests
func WithBaseURL(baseURL string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.BaseURL = baseURL
	}
}

// WithEconomicsURL sets the economics service URL
func WithEconomicsURL(economicsURL string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.EconomicsURL = economicsURL
	}
}

// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.APIKey = apiKey
	}
}

// WithNRNContract sets the NRN contract address
func WithNRNContract(contract string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.NRNContract = contract
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Timeout = timeout
	}
}

// WithHeader adds a custom header to the request
func WithHeader(key, value string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		cfg.Headers[key] = value
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(userAgent string) RequestOption {
	return WithHeader("User-Agent", userAgent)
}

// WithContentType sets the Content-Type header
func WithContentType(contentType string) RequestOption {
	return WithHeader("Content-Type", contentType)
}

// WithJSONContentType sets the Content-Type to application/json
func WithJSONContentType() RequestOption {
	return WithContentType("application/json")
}

// WithAuth sets the Authorization header
func WithAuth(token string) RequestOption {
	return WithHeader("Authorization", "Bearer "+token)
}

// WithBasicAuth sets basic authentication
func WithBasicAuth(username, password string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.BasicAuth = &requestconfig.BasicAuth{
			Username: username,
			Password: password,
		}
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.HTTPClient = client
	}
}

// WithRetries sets the number of retries for failed requests
func WithRetries(retries int) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Retries = retries
	}
}

// WithRetryDelay sets the delay between retries
func WithRetryDelay(delay time.Duration) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.RetryDelay = delay
	}
}

// WithRetryPolicy sets both retries and delay
func WithRetryPolicy(retries int, delay time.Duration) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Retries = retries
		cfg.RetryDelay = delay
	}
}

// WithEnvironmentProduction sets production environment defaults
func WithEnvironmentProduction() RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Environment = "production"
		cfg.BaseURL = "https://gateway.knirv.network"
		cfg.EconomicsURL = "https://economics.knirv.network"
	}
}

// WithEnvironmentStaging sets staging environment defaults
func WithEnvironmentStaging() RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Environment = "staging"
		cfg.BaseURL = "https://gateway-staging.knirv.network"
		cfg.EconomicsURL = "https://economics-staging.knirv.network"
	}
}

// WithEnvironmentDevelopment sets development environment defaults
func WithEnvironmentDevelopment() RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Environment = "development"
		cfg.BaseURL = "http://localhost:8000"
		cfg.EconomicsURL = "http://localhost:8090"
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Debug = debug
	}
}

// WithVerbose enables verbose logging
func WithVerbose(verbose bool) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.Verbose = verbose
	}
}

// WithKNIRVChainURL sets the KNIRVCHAIN service URL
func WithKNIRVChainURL(url string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.ServiceURLs == nil {
			cfg.ServiceURLs = make(map[string]string)
		}
		cfg.ServiceURLs["knirvchain"] = url
	}
}

// WithKNIRVNexusURL sets the KNIRVNEXUS service URL
func WithKNIRVNexusURL(url string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.ServiceURLs == nil {
			cfg.ServiceURLs = make(map[string]string)
		}
		cfg.ServiceURLs["knirvnexus"] = url
	}
}

// WithKNIRVRootURL sets the KNIRVORACLE service URL
func WithKNIRVRootURL(url string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.ServiceURLs == nil {
			cfg.ServiceURLs = make(map[string]string)
		}
		cfg.ServiceURLs["knirvoracle"] = url
	}
}

// WithKNIRVGraphURL sets the KNIRVGRAPH service URL
func WithKNIRVGraphURL(url string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.ServiceURLs == nil {
			cfg.ServiceURLs = make(map[string]string)
		}
		cfg.ServiceURLs["knirvgraph"] = url
	}
}

// WithAllKNIRVServices sets all KNIRV service URLs
func WithAllKNIRVServices(chainURL, nexusURL, rootURL, graphURL string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		if cfg.ServiceURLs == nil {
			cfg.ServiceURLs = make(map[string]string)
		}
		cfg.ServiceURLs["knirvchain"] = chainURL
		cfg.ServiceURLs["knirvnexus"] = nexusURL
		cfg.ServiceURLs["knirvoracle"] = rootURL
		cfg.ServiceURLs["knirvgraph"] = graphURL
	}
}

// WithDefaultKNIRVServices sets default KNIRV service URLs for development
func WithDefaultKNIRVServices() RequestOption {
	return WithAllKNIRVServices(
		"http://localhost:8080", // KNIRVCHAIN
		"http://localhost:8081", // KNIRVNEXUS
		"http://localhost:8082", // KNIRVORACLE
		"http://localhost:8083", // KNIRVGRAPH
	)
}

// WithRateLimiting enables rate limiting
func WithRateLimiting(enabled bool, requestsPerSecond int) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.RateLimiting = enabled
		cfg.RequestsPerSecond = requestsPerSecond
	}
}

// WithCircuitBreaker enables circuit breaker pattern
func WithCircuitBreaker(enabled bool, failureThreshold int, timeout time.Duration) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.CircuitBreaker = enabled
		cfg.FailureThreshold = failureThreshold
		cfg.CircuitBreakerTimeout = timeout
	}
}

// WithMetrics enables metrics collection
func WithMetrics(enabled bool) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.MetricsEnabled = enabled
	}
}

// WithTracing enables distributed tracing
func WithTracing(enabled bool, serviceName string) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.TracingEnabled = enabled
		cfg.ServiceName = serviceName
	}
}

// WithHealthCheck configures health check settings
func WithHealthCheck(enabled bool, interval time.Duration) RequestOption {
	return func(cfg *requestconfig.RequestConfig) {
		cfg.HealthCheckEnabled = enabled
		cfg.HealthCheckInterval = interval
	}
}
