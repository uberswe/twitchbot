package botsbyuberswe

import "time"

// Cookie is stored in the database and identified by a key stored in a http.cookie. This key can then be used to fetch a User via the TwitchID as an identifier
type Cookie struct {
	TwitchID string    `json:"twitch_id,omitempty"`
	Expiry   time.Time `json:"expiry,omitempty"`
}

// store stores a Cookie struct in the database using a key as an identifier, this key would be stored in a http.cookie
func (c *Cookie) store(key string) error {
	return storeStruct(c, "cookie", key)
}
