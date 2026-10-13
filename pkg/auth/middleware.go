package auth

import "net/http"

// RequireAuth is middleware that requires authentication
func (sm *SessionManager) RequireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sm.IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		handler(w, r)
	}
}

// AddNgrokHeaders adds the ngrok-skip-browser-warning header
func AddNgrokHeaders(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ngrok-skip-browser-warning", "true")
		handler(w, r)
	}
}
