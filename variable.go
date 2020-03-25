package botsbyuberswe

import "time"

// Variable can be used in commands, this might be things like the last user that raided or hosted a channel or the number of subscribers a channel has
type Variable struct {
	Name        string    `json:"name,omitempty"`
	Value       string    `json:"value,omitempty"`
	Description string    `json:"description,omitempty"`
	Expiry      time.Time `json:"expiry,omitempty"`
}
