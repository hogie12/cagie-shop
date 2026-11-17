// Package handler holds the HTTP layer of the order service.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"cagie-shop/order/internal/clients"
	"cagie-shop/order/internal/db"
	"cagie-shop/order/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Set by the gateway from a verified token. Services behind the gateway
// trust them, so they must never be reachable from outside.
const (
	headerUserID   = "X-User-Id"
	headerUserRole = "X-User-Role"
)

type Store interface {
	CreateOrder(ctx context.Context, userID string, total int64, items []domain.Item) (db.Order, error)
	GetOrder(ctx context.Context, id string) (db.Order, error)
	ListOrders(ctx context.Context, userID string) ([]db.Order, error)
	ListItems(ctx context.Context, orderIDs []string) ([]db.OrderItem, error)
}

type Catalog interface {
	GetProduct(ctx context.Context, id int64) (clients.Product, error)
}

type Orders struct {
	store   Store
	catalog Catalog
}

func NewOrders(s Store, c Catalog) *Orders { return &Orders{store: s, catalog: c} }

func (h *Orders) Routes(r gin.IRouter) {
	g := r.Group("", requireUser)
	g.POST("/orders", h.create)
	g.GET("/orders", h.list)
	g.GET("/orders/:id", h.get)
}

type lineInput struct {
	ProductID int64 `json:"productId" binding:"required,gt=0"`
	Quantity  int32 `json:"quantity" binding:"required,gte=1,lte=99"`
}

type createInput struct {
	Items []lineInput `json:"items" binding:"required,min=1,max=20,dive"`
}

type itemDTO struct {
	ProductID    int64  `json:"productId"`
	Name         string `json:"name"`
	UnitPriceIDR int64  `json:"unitPriceIdr"`
	Quantity     int32  `json:"quantity"`
}

type orderDTO struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	CancelReason string    `json:"cancelReason,omitempty"`
	TotalIDR     int64     `json:"totalIdr"`
	CreatedAt    time.Time `json:"createdAt"`
	Items        []itemDTO `json:"items"`
}

// requireUser refuses requests that did not come through the gateway.
func requireUser(c *gin.Context) {
	if c.GetHeader(headerUserID) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}

func (h *Orders) create(c *gin.Context) {
	var in createInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	merged, err := mergeLines(in.Items)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ctx := c.Request.Context()

	// 1. Ask the catalog for names and prices (never trust prices from the client).
	items := make([]domain.Item, 0, len(merged))
	var total int64
	for _, l := range merged {
		p, err := h.catalog.GetProduct(ctx, l.ProductID)
		if errors.Is(err, clients.ErrNotFound) {
			fail(c, http.StatusUnprocessableEntity, fmt.Sprintf("product %d not found", l.ProductID))
			return
		}
		if err != nil {
			log.Printf("catalog call failed: %v", err)
			fail(c, http.StatusBadGateway, "catalog unavailable")
			return
		}
		items = append(items, domain.Item{ProductID: p.ID, Name: p.Name, UnitPriceIDR: p.PriceIDR, Quantity: l.Quantity})
		total += p.PriceIDR * int64(l.Quantity)
	}

	// 2. Save the order as pending. The order.created event is stored in the same
	// transaction and published later, so stock is reserved without us waiting for it.
	order, err := h.store.CreateOrder(ctx, c.GetHeader(headerUserID), total, items)
	if err != nil {
		serverError(c, err)
		return
	}
	// 202 Accepted: the order exists but is not decided yet. Read it again to see
	// the outcome (confirmed or cancelled).
	c.Header("Location", "/orders/"+order.ID)
	c.JSON(http.StatusAccepted, toDTO(order, items))
}

func (h *Orders) list(c *gin.Context) {
	ctx := c.Request.Context()
	orders, err := h.store.ListOrders(ctx, c.GetHeader(headerUserID))
	if err != nil {
		serverError(c, err)
		return
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	rows, err := h.store.ListItems(ctx, ids)
	if err != nil {
		serverError(c, err)
		return
	}
	byOrder := map[string][]itemDTO{}
	for _, r := range rows {
		byOrder[r.OrderID] = append(byOrder[r.OrderID], itemDTO{r.ProductID, r.Name, r.UnitPriceIdr, r.Quantity})
	}
	out := make([]orderDTO, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderDTO{o.ID, o.Status, o.CancelReason.String, o.TotalIdr, o.CreatedAt.Time, orEmpty(byOrder[o.ID])})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Orders) get(c *gin.Context) {
	ctx := c.Request.Context()
	o, err := h.store.GetOrder(ctx, c.Param("id"))
	// A stranger gets the same 404 as a missing order, so ids cannot be probed.
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && o.UserID != c.GetHeader(headerUserID) && c.GetHeader(headerUserRole) != "admin") {
		fail(c, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	rows, err := h.store.ListItems(ctx, []string{o.ID})
	if err != nil {
		serverError(c, err)
		return
	}
	items := make([]itemDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, itemDTO{r.ProductID, r.Name, r.UnitPriceIdr, r.Quantity})
	}
	c.JSON(http.StatusOK, orderDTO{o.ID, o.Status, o.CancelReason.String, o.TotalIdr, o.CreatedAt.Time, items})
}

// mergeLines combines repeated products into one line, in a stable order.
func mergeLines(lines []lineInput) ([]lineInput, error) {
	sum := map[int64]int32{}
	for _, l := range lines {
		sum[l.ProductID] += l.Quantity
		if sum[l.ProductID] > 99 {
			return nil, fmt.Errorf("quantity for product %d exceeds 99", l.ProductID)
		}
	}
	out := make([]lineInput, 0, len(sum))
	for id, q := range sum {
		out = append(out, lineInput{ProductID: id, Quantity: q})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out, nil
}

func toDTO(o db.Order, items []domain.Item) orderDTO {
	dto := orderDTO{ID: o.ID, Status: o.Status, CancelReason: o.CancelReason.String, TotalIDR: o.TotalIdr, CreatedAt: o.CreatedAt.Time}
	for _, it := range items {
		dto.Items = append(dto.Items, itemDTO{it.ProductID, it.Name, it.UnitPriceIDR, it.Quantity})
	}
	return dto
}

func orEmpty(s []itemDTO) []itemDTO {
	if s == nil {
		return []itemDTO{}
	}
	return s
}

func fail(c *gin.Context, status int, msg string) { c.JSON(status, gin.H{"error": msg}) }

// serverError hides internal details from the client but keeps them in logs.
func serverError(c *gin.Context, err error) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, "internal error")
}
