package botsbyuberswe

// Command has in input such as `!so` which if it is detected in a Twitch channel message will trigger the Output to be sent by a Bot to the Twitch channel
type Command struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

// createCommand adds a command to a User struct
func (u *User) createCommand(command Command) bool {
	for _, c := range u.State.Commands {
		if c.Input == command.Input {
			return false
		}
	}
	u.State.Commands = append(u.State.Commands, command)
	return true
}

// removeCommand removes a command from a User struct
func (u *User) removeCommand(command Command) bool {
	for i, c := range u.State.Commands {
		if c.Input == command.Input {
			u.State.Commands = deleteCommand(u.State.Commands, i)
			return true
		}
	}
	return false
}
