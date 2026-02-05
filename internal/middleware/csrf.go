package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"context"
)

type contextKey string

const CSRFTokenKey contextKey = "csrf_token"

func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get or Create Token
		cookie, err := r.Cookie("csrf_token")
		token := ""
		if err != nil || cookie.Value == "" {
			token = GenerateToken()
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				// Secure:   true, // Uncomment in prod
			})
		} else {
			token = cookie.Value
		}

		// 2. Validate on POST
		if r.Method == "POST" {
			reqToken := r.FormValue("csrf_token")
			if reqToken == "" {
				reqToken = r.Header.Get("X-CSRF-Token")
			}
			if reqToken != token {
				http.Error(w, "Invalid CSRF Token", http.StatusForbidden)
				return
			}
		}

		// 3. Inject into Context for Templates
		ctx := context.WithValue(r.Context(), CSRFTokenKey, token)
		next(w, r.WithContext(ctx))
	}
}
