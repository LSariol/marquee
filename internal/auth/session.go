package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const (
	SessionCookieName = "marquee_session"
	StateCookieName   = "marquee_oauth_state"
	SessionDuration   = 30 * 24 * time.Hour
)

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CSRFToken derives a per-session CSRF token via HMAC so no extra DB column is needed.
func CSRFToken(sessionSecret, sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidCSRF(sessionSecret, sessionToken, provided string) bool {
	expected := CSRFToken(sessionSecret, sessionToken)
	return hmac.Equal([]byte(expected), []byte(provided))
}

func SessionExpiresAt() time.Time {
	return time.Now().Add(SessionDuration)
}
