package botsbyuberswe

// Template holds template variables which we pass to html files that render the frontend
type Template struct {
	ModifiedHash string
	BotURL       string
	BotName      string
	BotConnected bool
}
