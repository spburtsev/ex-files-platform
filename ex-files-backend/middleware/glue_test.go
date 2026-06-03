package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setCookieByName(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestWithCookieJar_SetsSessionCookie(t *testing.T) {
	h := WithCookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetSessionCookie(r.Context(), "tok123")
		_, _ = w.Write([]byte("ok")) // Write -> cookieWriter.WriteHeader applies the cookie
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	c := setCookieByName(rec.Result(), "session")
	require.NotNil(t, c)
	assert.Equal(t, "tok123", c.Value)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, int(SessionCookieTTL.Seconds()), c.MaxAge)
}

func TestWithCookieJar_ClearsSessionCookie(t *testing.T) {
	h := WithCookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ClearSessionCookie(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	c := setCookieByName(rec.Result(), "session")
	require.NotNil(t, c)
	assert.Less(t, c.MaxAge, 0, "cleared cookie has a negative max-age")
}

func TestCookieWriter_Flush(t *testing.T) {
	h := WithCookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.True(t, rec.Flushed)
}

func TestRecovery_PanicReturns500(t *testing.T) {
	h := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "internal server error")
}

func TestChain_AppliesOutermostFirst(t *testing.T) {
	var order []string
	mw := func(tag string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, tag)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusNoContent)
	})
	h := Chain(final, mw("first"), mw("second"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, []string{"first", "second", "handler"}, order)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRequestLogger_StatusRecorderWriteAndFlush(t *testing.T) {
	h := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write without WriteHeader -> statusRecorder.Write auto-sets 200.
		_, _ = w.Write([]byte("ok"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestClientIP_ForwardedHeaders(t *testing.T) {
	mk := func(set func(*http.Request)) string {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		set(r)
		return clientIP(r)
	}
	assert.Equal(t, "203.0.113.7", mk(func(r *http.Request) {
		r.Header.Set("X-Forwarded-For", "203.0.113.7, 70.41.3.18")
	}))
	assert.Equal(t, "198.51.100.2", mk(func(r *http.Request) {
		r.Header.Set("X-Real-Ip", "198.51.100.2")
	}))
	// Falls back to RemoteAddr host (httptest sets 192.0.2.1:1234).
	assert.Equal(t, "192.0.2.1", mk(func(r *http.Request) {}))
}
