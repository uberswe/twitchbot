package botsbyuberswe

import (
	"fmt"
	twitch "github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

// WebsocketMessage is the struct for the data sent over websocket connections
type WebsocketMessage struct {
	Key            string                `json:"key,omitempty"`
	Channel        string                `json:"channel,omitempty"`
	Command        string                `json:"command,omitempty"`
	Text           string                `json:"text,omitempty"`
	MsgParams      map[string]string     `json:"msg_params,omitempty"`
	PrivateMessage twitch.PrivateMessage `json:"private_message,omitempty"`
	State          State                 `json:"state,omitempty"`
	AlertType      string                `json:"alert_type,omitempty"`
	BotName        string                `json:"bot_name,omitempty"`
	TwitchID       string                `json:"-"`
}

// Wconn is Short for WebsocketConnection
type Wconn struct {
	Connection *websocket.Conn
	TwitchID   string
}

func initWebsockets() {
	// Configure the upgrader
	upgrader = websocket.Upgrader{}

	r.HandleFunc("/ws", handleConnections)

	go handleMessages()
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for i, client := range clients {
			if client.TwitchID == msg.TwitchID {
				err := client.Connection.WriteJSON(msg)
				log.Printf("Writing message to %s client\n", msg.TwitchID)
				if err != nil {
					log.Printf("error: %v", err)
					err := client.Connection.Close()
					if err != nil {
						log.Println(err)
					}
					clients = deleteClient(clients, i)
				}
			}
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	var user User
	authenticated := false

	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		log.Printf("Cant find cookie :/\r\n")
	} else {
		user, err = getUserFromCookie(cookie)
		if err != nil {
			log.Println(err)
			return
		}
		authenticated = true
		log.Println("Authenticated")

		clients = append(clients, Wconn{
			Connection: ws,
			TwitchID:   user.TwitchID,
		})

		sendStateMessage(ws, user)

	}

	if authenticated {
		if !user.Connected {
			log.Printf("Connect to channel %s: %s\n", user.AccessToken, user.Channel.Name)
			log.Println("connectToTwitch")
			user.TwitchIRCClient = connectToTwitch(user)

			user.Connected = true

			clientConnections[user.TwitchID] = user

			err = user.store()
			handleWsError(err, user.TwitchID)
			log.Println("Connect started")
		} else if user.Connected {
			log.Println("user already connected")
			sendChannelMessage(ws, user)
		} else {
			log.Println("invalid channel name")
		}

		// check if the bot is connected
		if val, ok := botConnections[user.TwitchID]; ok {
			if val.Connected {
				sendChannelMessageForBot(ws, user, val.Name)
			}
		} else {
			// If the user does not have it's own bot then it's part of the universal bot
			for _, connectedChannel := range universalConnectedChannels {
				if connectedChannel == user.Channel.Name {
					sendChannelMessageForBot(ws, user, botConnections[universalBotTwitchID].Name)
				}
			}
		}
	}

	for {
		var msg WebsocketMessage
		user, err = getUserFromTwitchID(user.TwitchID)
		if err != nil {
			log.Println(err)
			return
		}
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			index, err := getClientIndex(clients, user.TwitchID)
			if err != nil {
				log.Printf("Error: %s", err)
			} else {
				clients = deleteClient(clients, index)
				return
			}
			break
		}
		log.Printf("%s: %v\n", msg.Key, msg)
		log.Printf("Authenticated: %t\n", authenticated)
		if authenticated {
			if msg.Key == "createcommand" {
				// TODO add some validation here to ensure commands follow the correct pattern?
				handleCreateCommand(ws, msg, user)
			} else if msg.Key == "removecommand" {
				handleRemoveCommand(ws, msg, user)
			} else if msg.Key == "disconnectbot" {
				handleDisconnectBot(ws, user)
			} else if msg.Key == "logout" {
				handleLogoutMessage(ws, user)
				// We are logged out, no need to keep the connection open
				return
			} else {
				log.Printf("No matching command found: '%s'\n", msg.Key)
			}
		}
	}
}

func handleCreateCommand(ws *websocket.Conn, msg WebsocketMessage, user User) {
	log.Println(msg.Command, msg.Text)

	user.createCommand(Command{
		Input:  msg.Command,
		Output: msg.Text,
	})
	err := user.store()

	handleWsError(err, user.TwitchID)

	sendStateMessage(ws, user)

	sendAlertMessage(ws, "Command added successfully", "success", user.TwitchID)
}

func handleRemoveCommand(ws *websocket.Conn, msg WebsocketMessage, user User) {
	log.Println(msg.Command, msg.Text)
	for _, command := range user.State.Commands {
		if command.Input == msg.Text {
			user.removeCommand(command)
		}
	}

	err := user.store()

	handleWsError(err, user.TwitchID)

	sendStateMessage(ws, user)

	sendAlertMessage(ws, "Command removed successfully", "success", user.TwitchID)
}

func handleDisconnectBot(ws *websocket.Conn, user User) {
	if _, ok := botConnections[user.TwitchID]; ok {
		// TODO add error messages for failures here
		err := botConnections[user.TwitchID].TwitchIRCClient.Disconnect()
		if err != nil {
			log.Printf("Error: %s", err)
			return
		}
		err = db.Delete([]byte(fmt.Sprintf("bot:%s", user.TwitchID)), nil)
		if err != nil {
			log.Printf("Error: %s", err)
			return
		}
		delete(botConnections, user.TwitchID)
	}

	sendAlertMessage(ws, "Bot disconnected", "success", user.TwitchID)

	sendBotDisconnectedMessage(ws, user)
}

func handleLogoutMessage(ws *websocket.Conn, user User) {
	if _, ok := clientConnections[user.TwitchID]; ok {
		err := clientConnections[user.TwitchID].TwitchIRCClient.Disconnect()
		if err != nil {
			log.Printf("Error: %s", err)
		}
		err = db.Delete([]byte(fmt.Sprintf("user:%s", user.TwitchID)), nil)
		if err != nil {
			log.Printf("Error: %s", err)
		}
		delete(clientConnections, user.TwitchID)
	}
	if _, ok := botConnections[user.TwitchID]; ok {
		err := botConnections[user.TwitchID].TwitchIRCClient.Disconnect()
		if err != nil {
			log.Printf("Error: %s", err)
		}
		err = db.Delete([]byte(fmt.Sprintf("bot:%s", user.TwitchID)), nil)
		if err != nil {
			log.Printf("Error: %s", err)
		}
		delete(botConnections, user.TwitchID)
	}

	sendLogoutMessage(ws, user)
}

// handleWsError handles websocket errors. If a websocket has an error we disconnect the websocket and remove it from our clients array
func handleWsError(err error, twitchID string) {
	if err != nil {
		log.Printf("error: %v", err)
		index, err := getClientIndex(clients, twitchID)
		if err != nil {
			log.Printf("Error: %s", err)
		} else {
			clients = deleteClient(clients, index)
			return
		}
		return
	}
}

func sendLogoutMessage(ws *websocket.Conn, user User) {
	logoutMsg := WebsocketMessage{
		Key:      "logout",
		TwitchID: user.TwitchID,
	}

	_ = ws.WriteJSON(logoutMsg)
	log.Printf("Logging out %s\n", user.TwitchID)
	index, err := getClientIndex(clients, user.TwitchID)
	if err != nil {
		log.Printf("Error: %s", err)
	} else {
		clients = deleteClient(clients, index)
	}
}

func sendBotDisconnectedMessage(ws *websocket.Conn, user User) {
	disconnectMsg := WebsocketMessage{
		Key:      "botdisconnected",
		TwitchID: user.TwitchID,
	}

	err := ws.WriteJSON(disconnectMsg)
	handleWsError(err, user.TwitchID)
}

// sendStateMessage sends a websocket state message to the frontend
func sendStateMessage(ws *websocket.Conn, user User) {
	statemsg := WebsocketMessage{
		Key:      "state",
		Channel:  user.Channel.Name,
		State:    user.State,
		TwitchID: user.TwitchID,
	}

	err := ws.WriteJSON(statemsg)

	handleWsError(err, user.TwitchID)
}

// sendAlertMessage sends a websocket alert message to the frontend
func sendAlertMessage(ws *websocket.Conn, text string, alertType string, twitchID string) {
	alertmsg := WebsocketMessage{
		Key:       "alert",
		Text:      text,
		AlertType: alertType,
		TwitchID:  twitchID,
	}

	err := ws.WriteJSON(alertmsg)

	handleWsError(err, twitchID)
}

// sendChannelMessage is used to communicate when a user has connected to a Twitch channel
func sendChannelMessage(ws *websocket.Conn, user User) {
	channelmsg := WebsocketMessage{
		Key:      "channel",
		Channel:  user.Channel.Name,
		TwitchID: user.TwitchID,
	}

	err := ws.WriteJSON(channelmsg)

	handleWsError(err, user.TwitchID)
}

// sendChannelMessageForBot is used to communicate when a bot has connected to a users Twitch channel
func sendChannelMessageForBot(ws *websocket.Conn, user User, botName string) {
	channelmsg := WebsocketMessage{
		Key:      "channel",
		Channel:  user.Channel.Name,
		TwitchID: user.TwitchID,
		BotName:  botName,
	}

	err := ws.WriteJSON(channelmsg)

	handleWsError(err, user.TwitchID)
}

func broadcastMessage(msg WebsocketMessage) {
	broadcast <- msg
}
