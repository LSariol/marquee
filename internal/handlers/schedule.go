package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/lsariol/marquee/internal/middleware"
	"github.com/lsariol/marquee/internal/models"
)

type scheduleDay struct {
	Date    string
	Entries []models.ScheduleEntry
}

func (a *App) HandleCalendar(w http.ResponseWriter, r *http.Request) {
	entries, err := a.DB.ListSchedules(r.Context())
	if err != nil {
		slog.Error("list schedules", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var days []scheduleDay
	dateIndex := map[string]int{}
	for _, e := range entries {
		d := e.ScheduledFor.UTC().Format("2006-01-02")
		if idx, ok := dateIndex[d]; ok {
			days[idx].Entries = append(days[idx].Entries, e)
		} else {
			dateIndex[d] = len(days)
			days = append(days, scheduleDay{Date: d, Entries: []models.ScheduleEntry{e}})
		}
	}

	data := a.newPageData(r)
	data.Data = days
	a.render(w, r, "calendar.html", data)
}

func (a *App) HandleSchedulePage(w http.ResponseWriter, r *http.Request) {
	movieID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || movieID <= 0 {
		http.Error(w, "invalid movie id", http.StatusBadRequest)
		return
	}

	u := middleware.UserFromContext(r.Context())
	var userID int64
	if u != nil {
		userID = u.ID
	}

	movie, err := a.DB.GetPoolMovieRow(r.Context(), movieID, userID)
	if err != nil {
		slog.Error("get pool movie row", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if movie == nil {
		http.Error(w, "movie not found", http.StatusNotFound)
		return
	}

	schedules, err := a.DB.ListSchedulesForMovie(r.Context(), movieID)
	if err != nil {
		slog.Error("list schedules for movie", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := a.newPageData(r)
	data.Data = map[string]any{
		"Movie":     movie,
		"Schedules": schedules,
	}
	a.render(w, r, "schedule.html", data)
}

func (a *App) HandleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	movieID, err := strconv.ParseInt(r.FormValue("movie_id"), 10, 64)
	if err != nil || movieID <= 0 {
		http.Error(w, "invalid movie_id", http.StatusBadRequest)
		return
	}

	scheduledForStr := r.FormValue("scheduled_for")
	if scheduledForStr == "" {
		http.Error(w, "scheduled_for is required", http.StatusBadRequest)
		return
	}
	scheduledFor, err := time.Parse("2006-01-02T15:04", scheduledForStr)
	if err != nil {
		http.Error(w, "invalid scheduled_for format", http.StatusBadRequest)
		return
	}

	if _, err = a.DB.InsertSchedule(r.Context(), movieID, u.ID, u.TwitchLogin, scheduledFor); err != nil {
		slog.Error("insert schedule", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	slog.Info("schedule created", "movie_id", movieID, "streamer", u.TwitchLogin, "for", scheduledFor)
	http.Redirect(w, r, "/calendar", http.StatusSeeOther)
}

func (a *App) HandleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	scheduleID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || scheduleID <= 0 {
		http.Error(w, "invalid schedule id", http.StatusBadRequest)
		return
	}

	sched, err := a.DB.GetSchedule(r.Context(), scheduleID)
	if err != nil {
		slog.Error("get schedule", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sched == nil {
		http.Error(w, "schedule not found", http.StatusNotFound)
		return
	}

	if sched.StreamerID != u.ID && !u.IsAdmin() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := a.DB.DeleteSchedule(r.Context(), scheduleID); err != nil {
		slog.Error("delete schedule", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/calendar", http.StatusSeeOther)
}
