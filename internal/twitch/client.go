package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var Endpoint = struct{ AuthURL, TokenURL string }{
	AuthURL:  "https://id.twitch.tv/oauth2/authorize",
	TokenURL: "https://id.twitch.tv/oauth2/token",
}

type HelixUser struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	ProfileImageURL string `json:"profile_image_url"`
}

type Client struct {
	clientID   string
	httpClient *http.Client
}

func NewClient(clientID string) *Client {
	return &Client{
		clientID:   clientID,
		httpClient: &http.Client{},
	}
}

func (c *Client) GetUser(ctx context.Context, accessToken string) (*HelixUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitch helix /users returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []HelixUser `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode twitch response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("twitch helix /users returned no users")
	}
	return &result.Data[0], nil
}
