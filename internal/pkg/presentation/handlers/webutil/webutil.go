package webutil

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

// BuildPublicURL reconstructs the public-facing URL using forwarded headers.
// Works behind WSO2 API Gateway, Nginx, Envoy, API Gateway, Cloudflare, etc.
func BuildPublicURL(r *http.Request) *url.URL {
	u := *r.URL // shallow copy keeps Path and RawQuery

	// ---- Scheme ----
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		// Try RFC 7239 Forwarded: proto=https; host=...
		if f := r.Header.Get("Forwarded"); f != "" {
			lower := strings.ToLower(f)
			if strings.Contains(lower, "proto=https") {
				scheme = "https"
			} else if strings.Contains(lower, "proto=http") {
				scheme = "http"
			}
		}
	}
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	// ---- Host ----
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		// Try RFC 7239 Forwarded header
		if f := r.Header.Get("Forwarded"); f != "" {
			for _, part := range strings.Split(f, ";") {
				kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
				if len(kv) == 2 && strings.ToLower(kv[0]) == "host" {
					host = strings.Trim(kv[1], "\"")
					break
				}
			}
		}
	}
	if host == "" {
		host = r.Host
	}

	// ---- Prefix (optional) ----
	prefix := r.Header.Get("X-Forwarded-Prefix")
	if prefix != "" {
		prefix = "/" + strings.Trim(prefix, "/")
		u.Path = path.Join(prefix, u.Path)
	}

	u.Scheme = scheme
	u.Host = host

	return &u
}
