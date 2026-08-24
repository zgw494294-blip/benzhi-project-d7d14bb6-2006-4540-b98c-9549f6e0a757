package httpapi

import (
	"net/http"
	"strings"
)

func contentTypeOK(r *http.Request) bool {
	return r.Method == http.MethodGet || strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}
func pathID(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
