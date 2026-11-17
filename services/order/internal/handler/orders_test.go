package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"cagie-shop/order/internal/clients"
	"cagie-shop/order/internal/db"
	"cagie-shop/order/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeStore struct {
	orders map[string]*db.Order
	items  map[string][]domain.Item
	seq    int
}

func newStore() *fakeStore {
	return &fakeStore{orders: map[string]*db.Order{}, items: map[string][]domain.Item{}}
}

func (f *fakeStore) CreateOrder(_ context.Context, uid string, total int64, items []domain.Item) (db.Order, error) {
	f.seq++
	id := "o" + strconv.Itoa(f.seq)
	f.orders[id] = &db.Order{ID: id, UserID: uid, Status: domain.StatusPending, TotalIdr: total}
	f.items[id] = items
	return *f.orders[id], nil
}
func (f *fakeStore) GetOrder(_ context.Context, id string) (db.Order, error) {
	if o, ok := f.orders[id]; ok {
		return *o, nil
	}
	return db.Order{}, pgx.ErrNoRows
}
func (f *fakeStore) ListOrders(_ context.Context, uid string) ([]db.Order, error) {
	var out []db.Order
	for _, o := range f.orders {
		if o.UserID == uid {
			out = append(out, *o)
		}
	}
	return out, nil
}
func (f *fakeStore) ListItems(_ context.Context, ids []string) ([]db.OrderItem, error) {
	var out []db.OrderItem
	for _, id := range ids {
		for _, it := range f.items[id] {
			out = append(out, db.OrderItem{OrderID: id, ProductID: it.ProductID, Name: it.Name, UnitPriceIdr: it.UnitPriceIDR, Quantity: it.Quantity})
		}
	}
	return out, nil
}

type fakeCatalog struct {
	products map[int64]clients.Product
	err      error
}

func (f *fakeCatalog) GetProduct(_ context.Context, id int64) (clients.Product, error) {
	if f.err != nil {
		return clients.Product{}, f.err
	}
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return clients.Product{}, clients.ErrNotFound
}

type env struct {
	store *fakeStore
	cat   *fakeCatalog
	r     *gin.Engine
}

func setup() *env {
	e := &env{
		store: newStore(),
		cat: &fakeCatalog{products: map[int64]clients.Product{
			1: {ID: 1, Name: "Croissant", PriceIDR: 28000},
			4: {ID: 4, Name: "Rose Bouquet", PriceIDR: 350000},
		}},
	}
	gin.SetMode(gin.TestMode)
	e.r = gin.New()
	NewOrders(e.store, e.cat).Routes(e.r)
	return e
}

func (e *env) do(method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

var ana = map[string]string{"X-User-Id": "ana"}

func TestCreateOrderIsAcceptedAsPending(t *testing.T) {
	e := setup()
	// Repeated product lines are merged; any client-sent price is ignored.
	w := e.do("POST", "/orders", `{"items":[{"productId":1,"quantity":2},{"productId":4,"quantity":1},{"productId":1,"quantity":1,"price":1}]}`, ana)
	if w.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d %s", w.Code, w.Body)
	}
	var o orderDTO
	_ = json.Unmarshal(w.Body.Bytes(), &o)
	if o.Status != "pending" || o.TotalIDR != 3*28000+350000 || len(o.Items) != 2 {
		t.Fatalf("unexpected order: %+v", o)
	}
	if w.Header().Get("Location") != "/orders/"+o.ID {
		t.Fatalf("Location = %q", w.Header().Get("Location"))
	}
	if e.store.orders[o.ID].Status != "pending" {
		t.Fatal("stored order should be pending until stock is decided")
	}
}

func TestCreateOrderNeedsUserHeader(t *testing.T) {
	e := setup()
	if w := e.do("POST", "/orders", `{"items":[{"productId":1,"quantity":1}]}`, nil); w.Code != 401 {
		t.Fatalf("want 401, got %d", w.Code)
	}
	if len(e.store.orders) != 0 {
		t.Fatal("no order should be created")
	}
}

func TestCreateOrderValidation(t *testing.T) {
	e := setup()
	var many []string
	for i := 1; i <= 21; i++ {
		many = append(many, `{"productId":`+strconv.Itoa(i)+`,"quantity":1}`)
	}
	cases := map[string]string{
		"no items":       `{"items":[]}`,
		"missing items":  `{}`,
		"zero quantity":  `{"items":[{"productId":1,"quantity":0}]}`,
		"huge quantity":  `{"items":[{"productId":1,"quantity":100}]}`,
		"bad product id": `{"items":[{"productId":-1,"quantity":1}]}`,
		"merged over 99": `{"items":[{"productId":1,"quantity":60},{"productId":1,"quantity":60}]}`,
		"too many lines": `{"items":[` + strings.Join(many, ",") + `]}`,
	}
	for name, body := range cases {
		if w := e.do("POST", "/orders", body, ana); w.Code != 400 {
			t.Errorf("%s: want 400, got %d", name, w.Code)
		}
	}
	if len(e.store.orders) != 0 {
		t.Fatal("invalid requests must not create orders")
	}
}

func TestUnknownProductAndCatalogDown(t *testing.T) {
	e := setup()
	if w := e.do("POST", "/orders", `{"items":[{"productId":99,"quantity":1}]}`, ana); w.Code != 422 {
		t.Fatalf("unknown product: want 422, got %d", w.Code)
	}
	e.cat.err = errors.New("connection refused")
	if w := e.do("POST", "/orders", `{"items":[{"productId":1,"quantity":1}]}`, ana); w.Code != 502 {
		t.Fatalf("catalog down: want 502, got %d", w.Code)
	}
	if len(e.store.orders) != 0 {
		t.Fatal("nothing should be saved before prices are known")
	}
}

func TestReadingOrders(t *testing.T) {
	e := setup()
	w := e.do("POST", "/orders", `{"items":[{"productId":1,"quantity":1}]}`, ana)
	var o orderDTO
	_ = json.Unmarshal(w.Body.Bytes(), &o)

	if w := e.do("GET", "/orders/"+o.ID, "", ana); w.Code != 200 || !strings.Contains(w.Body.String(), "Croissant") {
		t.Fatalf("owner: %d %s", w.Code, w.Body)
	}
	bob := map[string]string{"X-User-Id": "bob"}
	if w := e.do("GET", "/orders/"+o.ID, "", bob); w.Code != 404 {
		t.Fatalf("stranger: want 404, got %d", w.Code)
	}
	if w := e.do("GET", "/orders/nope", "", ana); w.Code != 404 {
		t.Fatalf("missing: want 404, got %d", w.Code)
	}
	admin := map[string]string{"X-User-Id": "boss", "X-User-Role": "admin"}
	if w := e.do("GET", "/orders/"+o.ID, "", admin); w.Code != http.StatusOK {
		t.Fatalf("admin: want 200, got %d", w.Code)
	}
	if w := e.do("GET", "/orders", "", bob); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("bob list: %d %s", w.Code, w.Body)
	}
	if w := e.do("GET", "/orders", "", ana); !strings.Contains(w.Body.String(), o.ID) {
		t.Fatalf("ana list: %s", w.Body)
	}
}

func TestCancelledOrderShowsItsReason(t *testing.T) {
	e := setup()
	w := e.do("POST", "/orders", `{"items":[{"productId":1,"quantity":1}]}`, ana)
	var o orderDTO
	_ = json.Unmarshal(w.Body.Bytes(), &o)
	e.store.orders[o.ID].Status = "cancelled"
	e.store.orders[o.ID].CancelReason = pgtype.Text{String: "insufficient stock for product 1", Valid: true}
	got := e.do("GET", "/orders/"+o.ID, "", ana).Body.String()
	if !strings.Contains(got, `"status":"cancelled"`) || !strings.Contains(got, "insufficient stock for product 1") {
		t.Fatalf("got %s", got)
	}
}
