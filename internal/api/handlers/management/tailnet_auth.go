package management

import (
	"net"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// authenticateTailnet trusts identity headers only behind the local Tailscale
// Serve proxy. Serve strips caller-supplied identity headers and supplies its own.
// The listener MUST remain loopback-only; other local processes are trusted.
func (h *Handler) authenticateTailnet(r *http.Request) bool {
	if h == nil || h.cfg == nil || h.tailnetLogin == "" || h.tailnetOrigin == "" {
		return false
	}
	bindIP := net.ParseIP(h.cfg.Host)
	if bindIP == nil || !bindIP.IsLoopback() {
		return false
	}
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || net.ParseIP(peer) == nil || !net.ParseIP(peer).IsLoopback() {
		return false
	}
	origin, err := url.Parse(h.tailnetOrigin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || r.Host != origin.Host {
		return false
	}
	if r.Header.Get("Tailscale-User-Login") != h.tailnetLogin {
		return false
	}
	// Unlike bearer auth, network identity is ambient authority. Reject browser
	// cross-origin requests to prevent other sites from exercising that authority.
	if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" && requestOrigin != h.tailnetOrigin {
		return false
	}
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "none", "same-origin":
		return true
	default:
		return false
	}
}

// GetTailnetSession is a credential-free discovery endpoint for the console.
// It never returns a key or grants a reusable session token.
func (h *Handler) GetTailnetSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"authenticated": h.authenticateTailnet(c.Request)})
}
