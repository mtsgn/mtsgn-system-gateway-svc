package utils

import (
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (from proxy)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fallback to remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}
