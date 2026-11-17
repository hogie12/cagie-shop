package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cagie-shop/inventory/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type fakeStore struct{ stock map[int64]int32 }

func (f *fakeStore) GetStock(_ context.Context, id int64) (db.Stock, error) {
	q, ok := f.stock[id]
	if !ok {
		return db.Stock{}, pgx.ErrNoRows
	}
	return db.Stock{ProductID: id, Quantity: q}, nil
}
func (f *fakeStore) SetStock(_ context.Context, a db.SetStockParams) (db.Stock, error) {
	f.stock[a.ProductID] = a.Quantity
	return db.Stock{ProductID: a.ProductID, Quantity: a.Quantity}, nil
}

func do(s *fakeStore, method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewStock(s).Routes(r)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetStock(t *testing.T) {
	s := &fakeStore{stock: map[int64]int32{1: 5}}
	if w := do(s, "GET", "/inventory/1", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"quantity":5`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w := do(s, "GET", "/inventory/9", "", nil); w.Code != 404 {
		t.Fatalf("unknown product: %d", w.Code)
	}
	if w := do(s, "GET", "/inventory/x", "", nil); w.Code != 400 {
		t.Fatalf("bad id: %d", w.Code)
	}
}

func TestSetStockRequiresAdminHeader(t *testing.T) {
	s := &fakeStore{stock: map[int64]int32{}}
	if w := do(s, "PUT", "/inventory/1", `{"quantity":7}`, nil); w.Code != 403 {
		t.Fatalf("no header: want 403, got %d", w.Code)
	}
	if w := do(s, "PUT", "/inventory/1", `{"quantity":7}`, map[string]string{"X-User-Role": "customer"}); w.Code != 403 {
		t.Fatalf("customer: want 403, got %d", w.Code)
	}
	if w := do(s, "PUT", "/inventory/1", `{"quantity":7}`, map[string]string{"X-User-Role": "admin"}); w.Code != 200 || s.stock[1] != 7 {
		t.Fatalf("admin: %d, stock=%d", w.Code, s.stock[1])
	}
	if w := do(s, "PUT", "/inventory/1", `{"quantity":-1}`, map[string]string{"X-User-Role": "admin"}); w.Code != 400 {
		t.Fatalf("negative: want 400, got %d", w.Code)
	}
}
