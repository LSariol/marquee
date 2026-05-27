package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/lsariol/marquee/internal/auth"
	"golang.org/x/oauth2"
)

func (a *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := auth.GenerateToken()
	if err != nil {
		slog.Error("generate oauth state", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.StateCookieName,
		Value:    state,
		Path:     "/auth",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.Config.SecureCookies,
	})

	url := a.OAuth.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (a *App) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.Warn("oauth error from twitch", "error", errParam, "description", r.URL.Query().Get("error_description"))
		http.Redirect(w, r, "/?error=login_failed", http.StatusSeeOther)
		return
	}

	stateCookie, err := r.Cookie(auth.StateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	// Clear the state cookie immediately.
	http.SetCookie(w, &http.Cookie{
		Name:     auth.StateCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
	})

	token, err := a.OAuth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		slog.Error("oauth token exchange", "error", err)
		http.Redirect(w, r, "/?error=login_failed", http.StatusSeeOther)
		return
	}

	twitchUser, err := a.Twitch.GetUser(r.Context(), token.AccessToken)
	if err != nil {
		slog.Error("fetch twitch user", "error", err)
		http.Redirect(w, r, "/?error=login_failed", http.StatusSeeOther)
		return
	}

	user, err := a.DB.UpsertUser(r.Context(),
		twitchUser.ID, twitchUser.Login, twitchUser.DisplayName, twitchUser.ProfileImageURL,
		a.Config.OwnerTwitchID,
	)
	if err != nil {
		slog.Error("upsert user", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sessionToken, err := auth.GenerateToken()
	if err != nil {
		slog.Error("generate session token", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	tokenHash := auth.HashToken(sessionToken)
	expiresAt := auth.SessionExpiresAt()

	if err := a.DB.CreateSession(r.Context(), tokenHash, user.ID, expiresAt); err != nil {
		slog.Error("create session", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.Config.SecureCookies,
	})

	slog.Info("user logged in", "twitch_login", user.TwitchLogin, "role", user.Role)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		tokenHash := auth.HashToken(cookie.Value)
		if err := a.DB.DeleteSession(r.Context(), tokenHash); err != nil {
			slog.Error("delete session", "error", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
