package api

import (
	"net/http"

	"backend_server/internal/ebpf"

	"github.com/gin-gonic/gin"
)

// RegisterEBPFMetrics registers a simple metrics endpoint
func RegisterEBPFMetrics(router *gin.Engine, mgr *ebpf.Manager) {
	router.GET("/api/v1/ebpf/metrics", func(c *gin.Context) {
		metrics := mgr.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"programs_loaded": metrics.ProgramsAttached,
			"initialized":     metrics.Initialized,
		})
	})
}
