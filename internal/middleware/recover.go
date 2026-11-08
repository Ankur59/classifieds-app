package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recover turns a panic in a handler into a 500 response instead of
// dropping the connection, and logs the stack trace.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				// http.ErrAbortHandler is the sanctioned way to abort a response; re-panic.
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Printf("panic: %s %s: %v\n%s", r.Method, r.URL.Path, v, debug.Stack())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
