package http

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware returns a middleware that checks if the IP address
// from the X-Real-IP header belongs to the trusted subnet specified by trustedSubnetStr.
// If trustedSubnetStr is empty, the check is skipped and the middleware just passes the request through.
// If the header is missing, the IP is invalid, or not in the trusted subnet,
// it responds with HTTP 403 Forbidden.
// If the trustedSubnetStr is invalid CIDR, the middleware always responds with HTTP 500 Internal Server Error.
func TrustedSubnetMiddleware(trustedSubnetStr string) func(http.Handler) http.Handler {
	if trustedSubnetStr == "" {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	_, trustedNet, err := net.ParseCIDR(trustedSubnetStr)
	if err != nil {
		// Return middleware that responds 500 on every request
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			ip := net.ParseIP(ipStr)
			if ip == nil || !trustedNet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
