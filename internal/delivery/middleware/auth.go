package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const SessionKey contextKey = "session"

func AuthRequired(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sid := GetSessionID(r)
			if sid == "" {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			session, ok := store.Get(sid)
			if !ok {
				ClearSessionCookie(w)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := r.Context()
			ctx = withSession(ctx, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NivelRequerido(store *SessionStore, nivelMinimo int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := SessionFromContext(r)
			if session == nil || session.Nivel < nivelMinimo {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SessionFromContext(r *http.Request) *SessionData {
	s, _ := r.Context().Value(SessionKey).(*SessionData)
	return s
}

func withSession(ctx context.Context, s *SessionData) context.Context {
	return context.WithValue(ctx, SessionKey, s)
}
