package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cagie-shop/payment/internal/provider"
	"github.com/gin-gonic/gin"
)

func do(sw ModeSwitch, method, body, role string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Routes(r, sw)
	req := httptest.NewRequest(method, "/payments/provider", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOnlyAdminsCanUseTheControls(t *testing.T) {
	f := provider.NewFake()
	for _, role := range []string{"", "customer"} {
		if w := do(f, "GET", "", role); w.Code != 403 {
			t.Errorf("GET as %q: want 403, got %d", role, w.Code)
		}
		if w := do(f, "PUT", `{"mode":"down"}`, role); w.Code != 403 || f.Mode() != provider.ModeSucceed {
			t.Errorf("PUT as %q: want 403 and no change, got %d mode=%s", role, w.Code, f.Mode())
		}
	}
}

func TestAdminSwitchesTheMode(t *testing.T) {
	f := provider.NewFake()
	if w := do(f, "PUT", `{"mode":"decline"}`, "admin"); w.Code != http.StatusOK || f.Mode() != provider.ModeDecline {
		t.Fatalf("%d mode=%s", w.Code, f.Mode())
	}
	if w := do(f, "GET", "", "admin"); !strings.Contains(w.Body.String(), `"mode":"decline"`) {
		t.Fatalf("GET: %s", w.Body)
	}
	for _, bad := range []string{`{"mode":"explode"}`, `{}`, `nope`} {
		if w := do(f, "PUT", bad, "admin"); w.Code != 400 || f.Mode() != provider.ModeDecline {
			t.Errorf("%s: want 400 and no change, got %d", bad, w.Code)
		}
	}
}
