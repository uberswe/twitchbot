package botsbyuberswe

// State represents variables we show in the frontend, part of the User struct
type State struct {
	Commands  []Command  `json:"commands,omitempty"`
	Variables []Variable `json:"variables,omitempty"`
}
