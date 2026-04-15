package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// ProxySocketPath is the default Unix socket path for backend services
	ProxySocketPath = "/var/run/knirvserver-backend.sock"
)

// PWAHeaders middleware sets appropriate headers for PWA resources
func PWAHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Service Worker must always be fresh (not cached internally)
		if strings.HasSuffix(path, "service-worker.js") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Content-Type", "application/javascript; charset=utf-8")
			c.Header("Service-Worker-Allowed", "/")
		}

		// Manifest file configuration
		if strings.HasSuffix(path, "manifest.json") {
			c.Header("Content-Type", "application/manifest+json; charset=utf-8")
			c.Header("Cache-Control", "public, max-age=3600")
		}

		// PWA icons
		if strings.HasPrefix(path, "/icons/") && strings.HasSuffix(path, ".png") {
			c.Header("Cache-Control", "public, max-age=86400")
			c.Header("Content-Type", "image/png")
		}

		// Screenshots for PWA
		if strings.HasPrefix(path, "/screenshots/") {
			c.Header("Cache-Control", "public, max-age=604800")
		}

		c.Next()
	}
}

// GetProxySocketPath returns the configured Unix socket path
func GetProxySocketPath() string {
	return ProxySocketPath
}

// AddPWARoutes adds PWA-specific routes to the router
func AddPWARoutes(router *gin.Engine) {
	router.GET("/manifest.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/manifest+json; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=3600")
		c.File("frontend/public/manifest.json")
	})

	router.GET("/service-worker.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Service-Worker-Allowed", "/")
		c.File("public/service-worker.js")
	})
}
