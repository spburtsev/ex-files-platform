package middleware

import (
	"context"
	"net/http"
	"time"
)

type cookieJar struct {
	setToken string
	clear    bool
}

type cookieJarKey struct{}

// SessionCookieTTL is the lifetime applied when SetSessionCookie is used.
const SessionCookieTTL = 8 * time.Hour

// SetSessionCookie tells the surrounding WithCookieJar middleware to set a
// `session` cookie containing token on the next response.
func SetSessionCookie(ctx context.Context, token string) {
	if jar, ok := ctx.Value(cookieJarKey{}).(*cookieJar); ok {
		jar.setToken = token
	}
}

// ClearSessionCookie tells the surrounding WithCookieJar middleware to
// invalidate the session cookie.
func ClearSessionCookie(ctx context.Context) {
	if jar, ok := ctx.Value(cookieJarKey{}).(*cookieJar); ok {
		jar.clear = true
	}
}

// WithCookieJar threads a cookie jar through the request context so handlers
// can stash a `session` cookie command. The jar is consumed when the wrapped
// handler calls WriteHeader (or Write).
func WithCookieJar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), cookieJarKey{}, &cookieJar{})
		cw := &cookieWriter{ResponseWriter: w, ctx: ctx}
		next.ServeHTTP(cw, r.WithContext(ctx))
	})
}

type cookieWriter struct {
	http.ResponseWriter
	ctx     context.Context
	written bool
}

func (w *cookieWriter) WriteHeader(code int) {
	if w.written {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.written = true
	if jar, ok := w.ctx.Value(cookieJarKey{}).(*cookieJar); ok {
		switch {
		case jar.setToken != "":
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    jar.setToken,
				MaxAge:   int(SessionCookieTTL.Seconds()),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		case jar.clear:
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    "",
				MaxAge:   -1,
				Path:     "/",
				HttpOnly: true,
			})
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *cookieWriter) Write(p []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *cookieWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
