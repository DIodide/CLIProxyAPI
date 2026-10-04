package management

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func TestTailnetManagementIdentity(t *testing.T) {
	const origin = "https://mini.example.ts.net:8443"
	for _, tc := range []struct {
		name    string
		alter   func(*Handler, *http.Request)
		allowed bool
	}{
		{"serve identity", func(h *Handler, r *http.Request) {}, true},
		{"same origin", func(h *Handler, r *http.Request) {
			r.Header.Set("Origin", origin)
			r.Header.Set("Sec-Fetch-Site", "same-origin")
		}, true},
		{"disabled", func(h *Handler, r *http.Request) { h.tailnetLogin = "" }, false},
		{"no identity", func(h *Handler, r *http.Request) { r.Header.Del("Tailscale-User-Login") }, false},
		{"other user", func(h *Handler, r *http.Request) { r.Header.Set("Tailscale-User-Login", "other@example.com") }, false},
		{"public listener", func(h *Handler, r *http.Request) { h.cfg.Host = "0.0.0.0" }, false},
		{"all interfaces", func(h *Handler, r *http.Request) { h.cfg.Host = "" }, false},
		{"forged forwarded peer", func(h *Handler, r *http.Request) {
			r.RemoteAddr = "100.64.0.2:1234"
			r.Header.Set("X-Forwarded-For", "127.0.0.1")
		}, false},
		{"other host", func(h *Handler, r *http.Request) { r.Host = "evil.example" }, false},
		{"cross origin", func(h *Handler, r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, false},
		{"opaque origin", func(h *Handler, r *http.Request) { r.Header.Set("Origin", "null") }, false},
		{"cross site", func(h *Handler, r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, false},
		{"same site other origin", func(h *Handler, r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-site") }, false},
		{"http origin config", func(h *Handler, r *http.Request) { h.tailnetOrigin = "http://mini.example.ts.net:8443" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{cfg: &config.Config{Host: "127.0.0.1"}, tailnetLogin: "owner@example.com", tailnetOrigin: origin, envSecret: "recovery-key", failedAttempts: make(map[string]*attemptInfo)}
			r := httptest.NewRequest(http.MethodGet, origin+"/v8/management/config", nil)
			r.RemoteAddr = "127.0.0.1:1234"
			r.Header.Set("Tailscale-User-Login", "owner@example.com")
			tc.alter(h, r)
			engine := gin.New()
			engine.GET("/v8/management/config", h.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, r)
			if (rec.Code == http.StatusOK) != tc.allowed {
				t.Fatalf("status %d; allowed=%v", rec.Code, tc.allowed)
			}
		})
	}
}

func TestTailnetDiscoveryDoesNotAuthenticateLocalhost(t *testing.T) {
	h := &Handler{cfg: &config.Config{Host: "127.0.0.1"}, tailnetLogin: "owner@example.com", tailnetOrigin: "https://mini.example.ts.net:8443", envSecret: "recovery-key", failedAttempts: make(map[string]*attemptInfo)}
	engine := gin.New()
	engine.GET("/management-session", h.GetTailnetSession)
	engine.GET("/v8/management/config", h.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8317/management-session", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, r)
	if rec.Body.String() != `{"authenticated":false}` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected discovery: %s", rec.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8317/v8/management/config", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("Authorization", "Bearer recovery-key")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("recovery key failed: %d", rec.Code)
	}
}
