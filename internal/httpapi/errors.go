package httpapi

import "net/http"

func method(w http.ResponseWriter) {
	w.Header().Set("Allow", http.MethodPost+", "+http.MethodGet)
	write(w, 405, map[string]string{"error": "method not allowed"})
}
