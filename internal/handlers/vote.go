package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/lsariol/marquee/internal/middleware"
)

func (a *App) HandleVote(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/auth/login")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
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

	submitted, err := strconv.Atoi(r.FormValue("value"))
	if err != nil || (submitted != 1 && submitted != -1) {
		http.Error(w, "value must be 1 or -1", http.StatusBadRequest)
		return
	}

	current, err := a.DB.GetVote(r.Context(), movieID, u.ID)
	if err != nil {
		slog.Error("get vote", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if current == submitted {
		// Same value — toggle off (un-vote).
		if err := a.DB.DeleteVote(r.Context(), movieID, u.ID); err != nil {
			slog.Error("delete vote", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		if err := a.DB.UpsertVote(r.Context(), movieID, u.ID, submitted); err != nil {
			slog.Error("upsert vote", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	row, err := a.DB.GetPoolMovieRow(r.Context(), movieID, u.ID)
	if err != nil {
		slog.Error("get pool movie row", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if row == nil {
		http.Error(w, "movie not found", http.StatusNotFound)
		return
	}

	a.renderPartial(w, "partials/vote_cell.html", map[string]any{
		"Movie":     row,
		"CSRFToken": middleware.CSRFTokenFromContext(r.Context()),
	})
}
