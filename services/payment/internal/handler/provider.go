// Package handler holds the HTTP layer of the payment service: only the
// switch that puts the fake provider into a failure mode.
package handler

import (
	"net/http"

	"cagie-shop/payment/internal/provider"
	"github.com/gin-gonic/gin"
)

type ModeSwitch interface {
	Mode() provider.Mode
	SetMode(provider.Mode) error
}

// Set by the gateway from a verified token. Services behind the gateway trust it.
const headerUserRole = "X-User-Role"

// Routes registers the controls. The caller only does so when
// FAKE_PROVIDER_CONTROLS=true, so a real deployment never exposes them.
func Routes(r gin.IRouter, sw ModeSwitch) {
	g := r.Group("/payments/provider", requireAdmin)
	g.GET("", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"mode": sw.Mode()}) })
	g.PUT("", func(c *gin.Context) {
		var in struct {
			Mode string `json:"mode" binding:"required,oneof=succeed decline down"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be succeed, decline or down"})
			return
		}
		if err := sw.SetMode(provider.Mode(in.Mode)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"mode": sw.Mode()})
	})
}

func requireAdmin(c *gin.Context) {
	if c.GetHeader(headerUserRole) != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}
