package botsbyuberswe

import (
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix"
	"time"
)

// User represents a Twitch user connected to our application
type User struct {
	TwitchID              string         `json:"twitch_id,omitempty"`
	Email                 string         `json:"email,omitempty"`
	AccessCode            string         `json:"access_code,omitempty"`
	AccessToken           string         `json:"access_token,omitempty"`
	RefreshToken          string         `json:"refresh_token,omitempty"`
	TokenExpiry           time.Time      `json:"token_expiry,omitempty"`
	Scopes                []string       `json:"scopes,omitempty"`
	TokenType             string         `json:"token_type,omitempty"`
	Channel               Channel        `json:"channel,omitempty"`
	State                 State          `json:"state,omitempty"`
	Connected             bool           `json:"connected,omitempty"`
	ConnectAttempts       int            `json:"connect_attempts,omitempty"`
	BotToken              string         `json:"bot_token,omitempty"`
	TwitchIRCClient       *twitch.Client `json:"-"`
	TwitchOAuthClient     *helix.Client  `json:"-"`
	TwitchConnectFailures int            `json:"twitch_connect_failures,omitempty"`
}

// store stores a User struct
func (u *User) store() error {
	return storeStruct(u, "user", u.TwitchID)
}
