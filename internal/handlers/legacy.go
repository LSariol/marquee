package handlers

import (
	"log/slog"
	"net/http"
)

func (a *App) HandleLegacy(w http.ResponseWriter, r *http.Request) {
	movies, err := a.DB.ListWatched(r.Context())
	if err != nil {
		slog.Error("list watched", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := a.newPageData(r)
	data.Data = movies
	a.render(w, r, "legacy.html", data)
}
