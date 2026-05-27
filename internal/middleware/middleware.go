package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/lsariol/marquee/internal/auth"
	"github.com/lsariol/marquee/internal/database"
	"github.com/lsariol/marquee/internal/models"
)

type contextKey int

const (
	userKey         contextKey = iota
	sessionTokenKey contextKey = iota
	csrfTokenKey    contextKey = iota
)

func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userKey).(*models.User)
	return u
}

func SessionTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionTokenKey).(string)
	return v
}

func CSRFTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(csrfTokenKey).(string)
	return v
}

type Middleware struct {
	db            *database.DB
	sessionSecret string
}

func New(db *database.DB, sessionSecret string) *Middleware {
	return &Middleware{db: db, sessionSecret: sessionSecret}
}

func (m *Middleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start),
		)
	})
}

func (m *Middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Session reads the session cookie, loads the user into context, and derives the CSRF token.
func (m *Middleware) Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			// No session cookie — serve as anonymous.
			next.ServeHTTP(w, r)
			return
		}

		rawToken := cookie.Value
		tokenHash := auth.HashToken(rawToken)

		user, err := m.db.GetSessionUser(r.Context(), tokenHash)
		if err != nil {
			slog.Error("session lookup failed", "error", err)
			next.ServeHTTP(w, r)
			return
		}
		if user == nil {
			// Session not found or expired — clear stale cookie.
			clearCookie(w, auth.SessionCookieName)
			next.ServeHTTP(w, r)
			return
		}

		csrfToken := auth.CSRFToken(m.sessionSecret, rawToken)
		ctx := r.Context()
		ctx = context.WithValue(ctx, userKey, user)
		ctx = context.WithValue(ctx, sessionTokenKey, rawToken)
		ctx = context.WithValue(ctx, csrfTokenKey, csrfToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRF verifies the CSRF token on all non-safe requests.
// Checks X-CSRF-Token header (htmx) then csrf_token form field (plain HTML forms).
// Requests with no active session are passed through (handler enforces auth).
func (m *Middleware) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		safe := r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions
		if safe {
			next.ServeHTTP(w, r)
			return
		}

		rawToken := SessionTokenFromContext(r.Context())
		if rawToken == "" {
			// No session — let handler deal with it.
			next.ServeHTTP(w, r)
			return
		}

		provided := r.Header.Get("X-CSRF-Token")
		if provided == "" {
			if err := r.ParseForm(); err == nil {
				provided = r.FormValue("csrf_token")
			}
		}

		if !auth.ValidCSRF(m.sessionSecret, rawToken, provided) {
			http.Error(w, "invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects unauthenticated requests with a redirect to /auth/login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects requests from users whose role is below the minimum.
func RequireRole(minRole string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}
		if !hasRole(u, minRole) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hasRole(u *models.User, minRole string) bool {
	order := map[string]int{"user": 1, "streamer": 2, "admin": 3, "owner": 4}
	return order[u.Role] >= order[minRole]
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
