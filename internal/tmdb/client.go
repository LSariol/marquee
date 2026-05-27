package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const imageBase = "https://image.tmdb.org/t/p/w185"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, httpClient: &http.Client{}}
}

type SearchResult struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
}

func (r *SearchResult) ReleaseYear() *int {
	if len(r.ReleaseDate) < 4 {
		return nil
	}
	y, err := strconv.Atoi(r.ReleaseDate[:4])
	if err != nil {
		return nil
	}
	return &y
}

func (r *SearchResult) PosterURL() string {
	if r.PosterPath == "" {
		return ""
	}
	return imageBase + r.PosterPath
}

type MovieDetails struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	Overview    string  `json:"overview"`
	VoteAverage float64 `json:"vote_average"`
	Genres      []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
}

func (d *MovieDetails) ReleaseYear() *int {
	if len(d.ReleaseDate) < 4 {
		return nil
	}
	y, err := strconv.Atoi(d.ReleaseDate[:4])
	if err != nil {
		return nil
	}
	return &y
}

func (d *MovieDetails) PosterURL() *string {
	if d.PosterPath == "" {
		return nil
	}
	u := imageBase + d.PosterPath
	return &u
}

func (d *MovieDetails) GenreNames() []string {
	names := make([]string, len(d.Genres))
	for i, g := range d.Genres {
		names[i] = g.Name
	}
	return names
}

func (d *MovieDetails) Rating() *float64 {
	if d.VoteAverage == 0 {
		return nil
	}
	r := d.VoteAverage
	return &r
}

func (c *Client) SearchMovies(ctx context.Context, query string) ([]SearchResult, error) {
	u := "https://api.themoviedb.org/3/search/movie?" + url.Values{
		"api_key": {c.apiKey},
		"query":   {query},
		"page":    {"1"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb search returned status %d", resp.StatusCode)
	}

	var result struct {
		Results []SearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode tmdb search: %w", err)
	}
	return result.Results, nil
}

func (c *Client) GetMovieDetails(ctx context.Context, tmdbID int) (*MovieDetails, error) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d?api_key=%s", tmdbID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb movie details returned status %d", resp.StatusCode)
	}

	var details MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decode tmdb details: %w", err)
	}
	return &details, nil
}
