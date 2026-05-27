package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/lsariol/marquee/internal/middleware"
)

var validRoles = map[string]bool{
	"user": true, "streamer": true, "admin": true, "owner": true,
}

func (a *App) HandleAdminPage(w http.ResponseWriter, r *http.Request) {
	users, err := a.DB.ListAllUsers(r.Context())
	if err != nil {
		slog.Error("list all users", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := a.newPageData(r)
	data.Data = users
	a.render(w, r, "admin.html", data)
}

func (a *App) HandleSetUserRole(w http.ResponseWriter, r *http.Request) {
	actor := middleware.UserFromContext(r.Context())
	if actor == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if targetID == actor.ID {
		http.Error(w, "cannot change your own role", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	role := r.FormValue("role")
	if !validRoles[role] {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	if err := a.DB.SetUserRole(r.Context(), targetID, role); err != nil {
		slog.Error("set user role", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tid := targetID
	if err := a.DB.LogModeration(r.Context(), actor.ID, "role_change", nil, &tid, &role); err != nil {
		slog.Error("log moderation", "error", err)
	}

	slog.Info("user role changed", "target_id", targetID, "role", role, "by", actor.TwitchLogin)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
