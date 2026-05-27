package handlers

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/lsariol/marquee/internal/config"
	"github.com/lsariol/marquee/internal/database"
	"github.com/lsariol/marquee/internal/middleware"
	"github.com/lsariol/marquee/internal/tmdb"
	"github.com/lsariol/marquee/internal/twitch"
	"golang.org/x/oauth2"
)

var funcMap = template.FuncMap{
	"add":  func(a, b int) int { return a + b },
	"join": strings.Join,
	"fmtRating": func(f *float64) string {
		if f == nil {
			return ""
		}
		return fmt.Sprintf("%.1f", *f)
	},
	"fmtDate": func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format("Jan 2, 2006")
	},
	"fmtDateTime": func(t time.Time) string {
		return t.UTC().Format("Mon, Jan 2 2006 at 15:04 UTC")
	},
	"fmtTime": func(t time.Time) string {
		return t.UTC().Format("15:04 UTC")
	},
}

type App struct {
	DB        *database.DB
	Config    *config.Config
	OAuth     *oauth2.Config
	Twitch    *twitch.Client
	Tmdb      *tmdb.Client
	Templates fs.FS
}

type PageData struct {
	User      interface{}
	CSRFToken string
	Data      interface{}
}

func (a *App) newPageData(r *http.Request) PageData {
	return PageData{
		User:      middleware.UserFromContext(r.Context()),
		CSRFToken: middleware.CSRFTokenFromContext(r.Context()),
	}
}

func (a *App) render(w http.ResponseWriter, r *http.Request, tmpl string, data PageData) {
	t, err := template.New("").Funcs(funcMap).ParseFS(a.Templates, "base.html", tmpl)
	if err != nil {
		slog.Error("template parse error", "template", tmpl, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("template execute error", "template", tmpl, "error", err)
	}
}

func (a *App) renderPartial(w http.ResponseWriter, tmpl string, data any) {
	t, err := template.New("").Funcs(funcMap).ParseFS(a.Templates, tmpl)
	if err != nil {
		slog.Error("template parse error", "template", tmpl, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, path.Base(tmpl), data); err != nil {
		slog.Error("template execute error", "template", tmpl, "error", err)
	}
}
