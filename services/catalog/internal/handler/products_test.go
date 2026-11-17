package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cagie-shop/catalog/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeStore struct {
	products []db.Product
}

func (f *fakeStore) ListProducts(_ context.Context, cat pgtype.Text) ([]db.Product, error) {
	var out []db.Product
	for _, p := range f.products {
		if !cat.Valid || p.Category == cat.String {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeStore) GetProduct(_ context.Context, id int64) (db.Product, error) {
	for _, p := range f.products {
		if p.ID == id {
			return p, nil
		}
	}
	return db.Product{}, pgx.ErrNoRows
}

func (f *fakeStore) CreateProduct(_ context.Context, a db.CreateProductParams) (db.Product, error) {
	p := db.Product{ID: int64(len(f.products) + 1), Name: a.Name, Category: a.Category, PriceIdr: a.PriceIdr}
	f.products = append(f.products, p)
	return p, nil
}

func (f *fakeStore) UpdateProduct(_ context.Context, a db.UpdateProductParams) (db.Product, error) {
	return db.Product{}, pgx.ErrNoRows
}

func (f *fakeStore) DeactivateProduct(_ context.Context, id int64) (int64, error) {
	return 0, nil
}

func newRouter(s Store) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewProducts(s).Routes(r)
	return r
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return doAs(r, "", method, path, body)
}

func doAs(r http.Handler, role, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListFiltersByCategory(t *testing.T) {
	r := newRouter(&fakeStore{products: []db.Product{
		{ID: 1, Name: "Croissant", Category: "pastry"},
		{ID: 2, Name: "Rose", Category: "flower"},
	}})
	w := do(r, "GET", "/products?category=flower", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Rose") || strings.Contains(w.Body.String(), "Croissant") {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}

func TestListRejectsUnknownCategory(t *testing.T) {
	if w := do(newRouter(&fakeStore{}), "GET", "/products?category=car", ""); w.Code != 400 {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestGetValidatesID(t *testing.T) {
	r := newRouter(&fakeStore{})
	if w := do(r, "GET", "/products/abc", ""); w.Code != 400 {
		t.Fatalf("want 400, got %d", w.Code)
	}
	if w := do(r, "GET", "/products/99", ""); w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestCreateValidation(t *testing.T) {
	r := newRouter(&fakeStore{})
	cases := map[string]string{
		"missing name":   `{"category":"pastry","priceIdr":1000}`,
		"bad category":   `{"name":"x","category":"car","priceIdr":1000}`,
		"negative price": `{"name":"x","category":"pastry","priceIdr":-1}`,
	}
	for name, body := range cases {
		if w := doAs(r, "admin", "POST", "/products", body); w.Code != 400 {
			t.Errorf("%s: want 400, got %d", name, w.Code)
		}
	}
	if w := doAs(r, "admin", "POST", "/products", `{"name":"Eclair","category":"pastry","priceIdr":32000}`); w.Code != 201 {
		t.Errorf("valid create: want 201, got %d %s", w.Code, w.Body.String())
	}
}

func TestDeleteMissingReturns404(t *testing.T) {
	if w := doAs(newRouter(&fakeStore{}), "admin", "DELETE", "/products/5", ""); w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestWritesRequireAdminRole(t *testing.T) {
	r := newRouter(&fakeStore{})
	body := `{"name":"Eclair","category":"pastry","priceIdr":32000}`
	for _, role := range []string{"", "customer"} {
		for _, req := range [][3]string{{"POST", "/products", body}, {"PUT", "/products/1", body}, {"DELETE", "/products/1", ""}} {
			if w := doAs(r, role, req[0], req[1], req[2]); w.Code != 403 {
				t.Errorf("role %q %s %s: want 403, got %d", role, req[0], req[1], w.Code)
			}
		}
	}
	if w := do(r, "GET", "/products", ""); w.Code != 200 {
		t.Fatalf("reads stay public, got %d", w.Code)
	}
}
