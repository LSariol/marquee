package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/lsariol/marquee/internal/middleware"
)

func (a *App) HandleSuggestPage(w http.ResponseWriter, r *http.Request) {
	data := a.newPageData(r)
	a.render(w, r, "suggest.html", data)
}

func (a *App) HandleSuggestSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	results, err := a.Tmdb.SearchMovies(r.Context(), q)
	if err != nil {
		slog.Error("tmdb search", "error", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	a.renderPartial(w, "partials/search_results.html", map[string]any{
		"Results":   results,
		"CSRFToken": middleware.CSRFTokenFromContext(r.Context()),
	})
}

func (a *App) HandleCreateMovie(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tmdbIDStr := r.FormValue("tmdb_id")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil || tmdbID <= 0 {
		http.Error(w, "invalid tmdb_id", http.StatusBadRequest)
		return
	}

	watchDate := r.FormValue("suggested_watch_date") // "2006-01-02" or ""
	var watchDatePtr *string
	if watchDate != "" {
		watchDatePtr = &watchDate
	}

	details, err := a.Tmdb.GetMovieDetails(r.Context(), tmdbID)
	if err != nil {
		slog.Error("tmdb get details", "tmdb_id", tmdbID, "error", err)
		http.Error(w, "could not fetch movie details", http.StatusInternalServerError)
		return
	}

	_, alreadyExists, err := a.DB.InsertMovie(r.Context(), details, u.ID, watchDatePtr)
	if err != nil {
		slog.Error("insert movie", "tmdb_id", tmdbID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if alreadyExists {
		slog.Info("duplicate movie suggestion", "tmdb_id", tmdbID, "user", u.TwitchLogin)
		http.Redirect(w, r, "/?notice=already_suggested", http.StatusSeeOther)
		return
	}

	slog.Info("movie suggested", "tmdb_id", tmdbID, "title", details.Title, "user", u.TwitchLogin)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
