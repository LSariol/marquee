package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/lsariol/marquee/internal/config"
	"github.com/lsariol/marquee/internal/database"
	"github.com/lsariol/marquee/internal/handlers"
	"github.com/lsariol/marquee/internal/middleware"
	"github.com/lsariol/marquee/internal/tmdb"
	"github.com/lsariol/marquee/internal/twitch"
	"github.com/lsariol/marquee/web"
	"golang.org/x/oauth2"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(cfg.MigratorDatabaseURL); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := database.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database pool failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	db := database.New(pool)
	mw := middleware.New(db, cfg.SessionSecret)

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.TwitchClientID,
		ClientSecret: cfg.TwitchClientSecret,
		RedirectURL:  cfg.BaseURL + "/auth/callback",
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://id.twitch.tv/oauth2/authorize",
			TokenURL: "https://id.twitch.tv/oauth2/token",
		},
	}

	tmplFS, err := fs.Sub(web.Templates, "templates")
	if err != nil {
		slog.Error("templates error", "error", err)
		os.Exit(1)
	}

	app := &handlers.App{
		DB:        db,
		Config:    cfg,
		OAuth:     oauthCfg,
		Twitch:    twitch.NewClient(cfg.TwitchClientID),
		Tmdb:      tmdb.NewClient(cfg.TMDBAPIKey),
		Templates: tmplFS,
	}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(web.StaticFiles, "static")
	if err != nil {
		slog.Error("static files error", "error", err)
		os.Exit(1)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Pool.Ping(r.Context()); err != nil {
			slog.Error("healthz db ping failed", "error", err)
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "ok")
	})

	// Public routes
	mux.HandleFunc("GET /", app.HandleHome)
	mux.HandleFunc("GET /auth/login", app.HandleLogin)
	mux.HandleFunc("GET /auth/callback", app.HandleCallback)
	mux.HandleFunc("POST /auth/logout", app.HandleLogout)

	// Suggest (auth required enforced in handler)
	mux.Handle("GET /suggest", middleware.RequireAuth(http.HandlerFunc(app.HandleSuggestPage)))
	mux.Handle("GET /suggest/search", middleware.RequireAuth(http.HandlerFunc(app.HandleSuggestSearch)))
	mux.Handle("POST /movies", middleware.RequireAuth(http.HandlerFunc(app.HandleCreateMovie)))

	// Voting
	mux.HandleFunc("POST /movies/{id}/vote", app.HandleVote)

	// Calendar & scheduling
	mux.HandleFunc("GET /calendar", app.HandleCalendar)
	mux.HandleFunc("GET /schedule/{id}", app.HandleSchedulePage)
	mux.Handle("POST /schedules", middleware.RequireRole("streamer", http.HandlerFunc(app.HandleCreateSchedule)))
	mux.Handle("POST /schedules/{id}/delete", middleware.RequireRole("streamer", http.HandlerFunc(app.HandleDeleteSchedule)))

	// Watched, legacy & moderation
	mux.HandleFunc("GET /legacy", app.HandleLegacy)
	mux.Handle("POST /movies/{id}/watched", middleware.RequireRole("streamer", http.HandlerFunc(app.HandleMarkWatched)))
	mux.Handle("POST /movies/{id}/back-to-pool", middleware.RequireRole("streamer", http.HandlerFunc(app.HandleBackToPool)))
	mux.Handle("POST /movies/{id}/remove", middleware.RequireRole("admin", http.HandlerFunc(app.HandleRemoveMovie)))

	// Admin
	mux.Handle("GET /admin", middleware.RequireRole("owner", http.HandlerFunc(app.HandleAdminPage)))
	mux.Handle("POST /admin/users/{id}/role", middleware.RequireRole("owner", http.HandlerFunc(app.HandleSetUserRole)))

	// User profiles
	mux.HandleFunc("GET /u/{login}", app.HandleProfile)

	// Redirect /me to the current user's profile (Phase 7)
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		u := middleware.UserFromContext(r.Context())
		if u == nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/u/"+u.TwitchLogin, http.StatusSeeOther)
	})

	handler := mw.Logger(mw.Recovery(mw.Session(mw.CSRF(mux))))

	addr := ":" + cfg.Port
	slog.Info("listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
