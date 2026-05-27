package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/lsariol/marquee/internal/middleware"
)

func (a *App) HandleMarkWatched(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	movieID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || movieID <= 0 {
		http.Error(w, "invalid movie id", http.StatusBadRequest)
		return
	}

	found, err := a.DB.MarkWatched(r.Context(), movieID, u.TwitchLogin)
	if err != nil {
		slog.Error("mark watched", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "movie not found or not in pool", http.StatusNotFound)
		return
	}

	mid := movieID
	if err := a.DB.LogModeration(r.Context(), u.ID, "watched", &mid, nil, nil); err != nil {
		slog.Error("log moderation", "error", err)
	}

	slog.Info("movie marked watched", "movie_id", movieID, "streamer", u.TwitchLogin)
	http.Redirect(w, r, "/legacy", http.StatusSeeOther)
}

func (a *App) HandleBackToPool(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	movieID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || movieID <= 0 {
		http.Error(w, "invalid movie id", http.StatusBadRequest)
		return
	}

	found, err := a.DB.MarkBackToPool(r.Context(), movieID)
	if err != nil {
		slog.Error("mark back to pool", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "movie not found or not in watched status", http.StatusNotFound)
		return
	}

	mid := movieID
	if err := a.DB.LogModeration(r.Context(), u.ID, "back_to_pool", &mid, nil, nil); err != nil {
		slog.Error("log moderation", "error", err)
	}

	slog.Info("movie returned to pool", "movie_id", movieID, "actor", u.TwitchLogin)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) HandleRemoveMovie(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	movieID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || movieID <= 0 {
		http.Error(w, "invalid movie id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	reason := r.FormValue("reason")
	if reason == "" {
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	}

	found, err := a.DB.MarkRemoved(r.Context(), movieID)
	if err != nil {
		slog.Error("mark removed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "movie not found or already removed", http.StatusNotFound)
		return
	}

	mid := movieID
	if err := a.DB.LogModeration(r.Context(), u.ID, "removed", &mid, nil, &reason); err != nil {
		slog.Error("log moderation", "error", err)
	}

	slog.Info("movie removed", "movie_id", movieID, "admin", u.TwitchLogin, "reason", reason)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
