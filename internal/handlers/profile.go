package handlers

import (
	"log/slog"
	"net/http"
)

func (a *App) HandleProfile(w http.ResponseWriter, r *http.Request) {
	login := r.PathValue("login")

	profile, err := a.DB.GetUserByLogin(r.Context(), login)
	if err != nil {
		slog.Error("get user by login", "login", login, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := a.newPageData(r)

	if profile == nil {
		w.WriteHeader(http.StatusNotFound)
		data.Data = map[string]any{"Profile": nil}
		a.render(w, r, "profile.html", data)
		return
	}

	suggestions, err := a.DB.ListUserSuggestions(r.Context(), profile.ID)
	if err != nil {
		slog.Error("list user suggestions", "user_id", profile.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	votes, err := a.DB.ListUserVotes(r.Context(), profile.ID)
	if err != nil {
		slog.Error("list user votes", "user_id", profile.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data.Data = map[string]any{
		"Profile":     profile,
		"Suggestions": suggestions,
		"Votes":       votes,
	}
	a.render(w, r, "profile.html", data)
}
