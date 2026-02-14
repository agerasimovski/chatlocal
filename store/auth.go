package store

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func RequireAuth(users *UserStore, sessions *SessionStore, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || r.URL.Path == "/register" {
			h.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie == nil || cookie.Value == "" {
			if isAPI(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userID, ok := sessions.Get(cookie.Value)
		if !ok || users.ByID(userID) == nil {
			if isAPI(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := ContextWithUserID(r.Context(), userID)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}

func isAPI(r *http.Request) bool {
	return r.URL.Path == "/prompt" || r.URL.Path == "/chats" ||
		(len(r.URL.Path) > 6 && r.URL.Path[:6] == "/chats/") ||
		r.URL.Path == "/me"
}
