package botsbyuberswe

import (
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix"
	"time"
)

// Bot represents twitch users connected via the botCallback route. These should post in users channels, Users should never post in channels
type Bot struct {
	Name                  string         `json:"name,omitempty"`
	UserChannelName       string         `json:"user_channel_name,omitempty"`
	TwitchIRCClient       *twitch.Client `json:"-"`
	TwitchOAuthClient     *helix.Client  `json:"-"`
	TwitchConnectFailures int            `json:"twitch_connect_failures,omitempty"`
	AccessCode            string         `json:"access_code,omitempty"`
	AccessToken           string         `json:"access_token,omitempty"`
	RefreshToken          string         `json:"refresh_token,omitempty"`
	Connected             bool           `json:"connected,omitempty"`
	ConnectAttempts       int            `json:"connect_attempts,omitempty"`
	UserTwitchID          string         `json:"user_twitch_id,omitempty"`
	TokenExpiry           time.Time      `json:"token_expiry,omitempty"`
}

// BotToken is used to identify which user the bot should link to when using the botCallback route
type BotToken struct {
	Token    string
	TwitchID string
}

// store stores a Bot struct in the database
func (b *Bot) store() error {
	return storeStruct(b, "bot", b.UserTwitchID)
}

// store stores a BotToken struct in the database
func (b *BotToken) store() error {
	return storeStruct(b, "bottoken", b.Token)
}
