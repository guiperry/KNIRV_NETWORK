package launcher

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// registerGitTCPNotFound answers /git/* on the public TCP router with an explicit 404.
// /git/* is served only on the git Unix socket (startKnirvRouteSockets). Without this
// route, the SPA NoRoute fallback would return the frontend HTML for those paths.
func registerGitTCPNotFound(r *gin.Engine) {
	r.Any("/git/*path", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
}
