package handlers

import (
	"log/slog"
	"net/http"

	"github.com/lsariol/marquee/internal/middleware"
)

func (a *App) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var userID int64
	if u := middleware.UserFromContext(r.Context()); u != nil {
		userID = u.ID
	}

	pool, err := a.DB.ListPool(r.Context(), userID)
	if err != nil {
		slog.Error("list pool", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := a.newPageData(r)
	data.Data = pool
	a.render(w, r, "index.html", data)
}
