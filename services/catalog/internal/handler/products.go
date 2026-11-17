// Package handler holds the HTTP layer. It talks to the database only
// through the Store interface, which keeps handlers easy to test.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"cagie-shop/catalog/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	ListProducts(ctx context.Context, category pgtype.Text) ([]db.Product, error)
	GetProduct(ctx context.Context, id int64) (db.Product, error)
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error)
	UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error)
	DeactivateProduct(ctx context.Context, id int64) (int64, error)
}

type Products struct {
	store Store
}

func NewProducts(store Store) *Products { return &Products{store: store} }

// Routes registers the product endpoints.
// Writes need the admin role. The gateway already checks the token; this check
// is a second layer, so a request that skips the gateway is still refused.
func (h *Products) Routes(r gin.IRouter) {
	r.GET("/products", h.list)
	r.GET("/products/:id", h.get)
	r.POST("/products", requireAdmin, h.create)
	r.PUT("/products/:id", requireAdmin, h.update)
	r.DELETE("/products/:id", requireAdmin, h.remove)
}

// X-User-Role is set by the gateway from a verified token. Services behind
// the gateway trust it, so they must never be reachable from outside.
func requireAdmin(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}

// productDTO is the public JSON shape. DB structs are never exposed
// directly, so a schema change does not silently change the API.
type productDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	PriceIDR    int64  `json:"priceIdr"`
	ImageURL    string `json:"imageUrl"`
}

type productInput struct {
	Name        string `json:"name" binding:"required,max=120"`
	Category    string `json:"category" binding:"required,oneof=pastry flower"`
	Description string `json:"description" binding:"max=1000"`
	PriceIDR    int64  `json:"priceIdr" binding:"gte=0"`
	ImageURL    string `json:"imageUrl" binding:"max=500"`
}

func toDTO(p db.Product) productDTO {
	return productDTO{p.ID, p.Name, p.Category, p.Description, p.PriceIdr, p.ImageUrl}
}

func (h *Products) list(c *gin.Context) {
	category := pgtype.Text{}
	if v := c.Query("category"); v != "" {
		if v != "pastry" && v != "flower" {
			fail(c, http.StatusBadRequest, "category must be pastry or flower")
			return
		}
		category = pgtype.Text{String: v, Valid: true}
	}
	rows, err := h.store.ListProducts(c.Request.Context(), category)
	if err != nil {
		serverError(c, err)
		return
	}
	out := make([]productDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, toDTO(p))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Products) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.store.GetProduct(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(p))
}

func (h *Products) create(c *gin.Context) {
	var in productInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.store.CreateProduct(c.Request.Context(), db.CreateProductParams{
		Name: in.Name, Category: in.Category, Description: in.Description,
		PriceIdr: in.PriceIDR, ImageUrl: in.ImageURL,
	})
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(p))
}

func (h *Products) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in productInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.store.UpdateProduct(c.Request.Context(), db.UpdateProductParams{
		ID: id, Name: in.Name, Category: in.Category, Description: in.Description,
		PriceIdr: in.PriceIDR, ImageUrl: in.ImageURL,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(p))
}

func (h *Products) remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	n, err := h.store.DeactivateProduct(c.Request.Context(), id)
	if err != nil {
		serverError(c, err)
		return
	}
	if n == 0 {
		fail(c, http.StatusNotFound, "product not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// serverError hides internal details from the client but keeps them in logs.
func serverError(c *gin.Context, err error) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, "internal error")
}
