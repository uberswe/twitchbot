package botsbyuberswe

// Channel holds channel information about a Twitch channel
type Channel struct {
	Name     string `json:"name,omitempty"`
	IsMod    bool   `json:"is_mod,omitempty"`
	LastHost string `json:"last_host,omitempty"`
	LastRaid string `json:"last_raid,omitempty"`
}

// ConnectChannel is used to tell Bots to connect to a specific channel
type ConnectChannel struct {
	Name    string
	Connect bool
}
