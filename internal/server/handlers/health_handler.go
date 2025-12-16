package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Version information (set at build time).
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// VersionResponse represents a version response.
type VersionResponse struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
}

// Health handles health check requests.
// @Summary Health check
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// GetVersion handles version requests.
// @Summary Get server version
// @Tags system
// @Produce json
// @Success 200 {object} VersionResponse
// @Router /version [get]
func (h *Handler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Version:   Version,
		BuildDate: BuildDate,
	})
}

