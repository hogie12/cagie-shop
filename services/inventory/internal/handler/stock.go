// Package handler holds the HTTP layer of the inventory service.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"cagie-shop/inventory/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Set by the gateway from a verified token. Services behind the gateway
// trust it, so they must never be reachable from outside.
const headerUserRole = "X-User-Role"

type Store interface {
	GetStock(ctx context.Context, productID int64) (db.Stock, error)
	SetStock(ctx context.Context, arg db.SetStockParams) (db.Stock, error)
}

type Stock struct{ store Store }

func NewStock(store Store) *Stock { return &Stock{store: store} }

// Stock is reserved by reacting to events (see internal/store), not by REST calls.
func (h *Stock) Routes(r gin.IRouter) {
	r.GET("/inventory/:id", h.get)
	r.PUT("/inventory/:id", requireAdmin, h.set)
}

type stockDTO struct {
	ProductID int64 `json:"productId"`
	Quantity  int32 `json:"quantity"`
}

type setInput struct {
	Quantity int32 `json:"quantity" binding:"gte=0,lte=100000"`
}

func requireAdmin(c *gin.Context) {
	if c.GetHeader(headerUserRole) != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}

func (h *Stock) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	s, err := h.store.GetStock(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "no stock record for product")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, stockDTO{s.ProductID, s.Quantity})
}

func (h *Stock) set(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in setInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	s, err := h.store.SetStock(c.Request.Context(), db.SetStockParams{ProductID: id, Quantity: in.Quantity})
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, stockDTO{s.ProductID, s.Quantity})
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func fail(c *gin.Context, status int, msg string) { c.JSON(status, gin.H{"error": msg}) }

// serverError hides internal details from the client but keeps them in logs.
func serverError(c *gin.Context, err error) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, "internal error")
}
