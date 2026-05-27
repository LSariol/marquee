package models

import "time"

type User struct {
	ID          int64
	TwitchID    string
	TwitchLogin string
	DisplayName string
	AvatarURL   string
	Role        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (u *User) IsStreamer() bool { return u.Role == "streamer" || u.IsAdmin() }
func (u *User) IsAdmin() bool    { return u.Role == "admin" || u.IsOwner() }
func (u *User) IsOwner() bool    { return u.Role == "owner" }

type Session struct {
	TokenHash string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Movie struct {
	ID                 int64
	TMDBId             int
	Title              string
	ReleaseYear        *int
	PosterURL          *string
	TMDBRating         *float64
	Genres             []string
	Overview           *string
	Status             string
	SuggestedBy        int64
	SuggestedWatchDate *time.Time
	CreatedAt          time.Time
	WatchedAt          *time.Time
	WatchedChannel     *string
}

// PoolMovie is a row in the voting pool with aggregated data.
type PoolMovie struct {
	ID                 int64
	Title              string
	ReleaseYear        *int
	PosterURL          *string
	TMDBRating         *float64
	Genres             []string
	SuggesterLogin     string
	SuggesterDisplay   string
	NetVotes           int64
	SuggestedWatchDate *time.Time
	UserVote           int // 0 = no vote, 1 = upvote, -1 = downvote
	WatchingSoon       *time.Time
	WatchingChannel    *string
}

type Schedule struct {
	ID           int64
	MovieID      int64
	StreamerID   int64
	Channel      string
	ScheduledFor time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ScheduleSimple struct {
	ID              int64
	Channel         string
	ScheduledFor    time.Time
	StreamerLogin   string
	StreamerDisplay string
}

type ScheduleEntry struct {
	ID              int64
	MovieID         int64
	MovieTitle      string
	PosterURL       *string
	Channel         string
	ScheduledFor    time.Time
	StreamerID      int64
	StreamerLogin   string
	StreamerDisplay string
}

type WatchedMovie struct {
	ID               int64
	Title            string
	ReleaseYear      *int
	PosterURL        *string
	TMDBRating       *float64
	Genres           []string
	SuggesterLogin   string
	SuggesterDisplay string
	WatchedAt        *time.Time
	WatchedChannel   *string
}

type UserSuggestion struct {
	ID          int64
	Title       string
	ReleaseYear *int
	PosterURL   *string
	Status      string
	NetVotes    int64
	WatchedAt   *time.Time
}

type UserVote struct {
	MovieID     int64
	Title       string
	ReleaseYear *int
	PosterURL   *string
	Value       int
	Status      string
}
