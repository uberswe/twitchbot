package botsbyuberswe

import (
	"encoding/json"
	"fmt"
	twitch "github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type User struct {
	AccessToken string    `json:"access_token,omitempty"`
	Scopes      []string  `json:"scopes,omitempty"`
	TokenType   string    `json:"token_type,omitempty"`
	Channel     Channel   `json:"channel,omitempty"`
	Commands    []Command `json:"commands,omitempty"`
	State       State     `json:"state,omitempty"`
}

type Command struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

type Channel struct {
	Name     string         `json:"name,omitempty"`
	Client   *twitch.Client `json:"client,omitempty"`
	IsMod    bool           `json:"is_mod,omitempty"`
	LastHost string         `json:"last_host,omitempty"`
	LastRaid string         `json:"last_raid,omitempty"`
}

type WebsocketMessage struct {
	Key            string                `json:"key,omitempty"`
	Channel        string                `json:"channel,omitempty"`
	Command        string                `json:"command,omitempty"`
	Text           string                `json:"text,omitempty"`
	MsgParams      map[string]string     `json:"msg_params,omitempty"`
	PrivateMessage twitch.PrivateMessage `json:"private_message,omitempty"`
	State          State                 `json:"state,omitempty"`
}

type State struct {
	Commands []Command `json:"commands,omitempty"`
}

func initWebsockets() {
	// Configure the upgrader
	upgrader = websocket.Upgrader{}

	http.HandleFunc("/ws", handleConnections)
	// Start listening for incoming chat messages
	go handleMessages()
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				err := client.Close()
				if err != nil {
					log.Println(err)
				}
				delete(clients, client)
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
	// Register our new client
	clients[ws] = true

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		log.Printf("Cant find cookie :/\r\n")
	} else {
		log.Printf("cookie val: %s", cookie.Value)

		data, err := db.Get([]byte(cookie.Value), nil)

		err = json.Unmarshal(data, &user)
		if err != nil {
			log.Println(err)
			delete(clients, ws)
			return
		} else {
			authenticated = true
			log.Println("Authenticated")
			statemsg := WebsocketMessage{
				Key:     "state",
				Channel: user.Channel.Name,
				State:   user.State,
			}

			broadcast <- statemsg
		}
	}

	for {
		var msg WebsocketMessage
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		fmt.Printf("%s: %v\n", msg.Key, msg)
		fmt.Printf("Authenticated: %t\n", authenticated)
		if msg.Key == "connect" && authenticated {
			if len(msg.Channel) > 1 && len(msg.Channel) < 70 && !user.isConnected(msg.Channel) {
				var channel Channel
				log.Printf("Connect to channel %s: %s\n", user.AccessToken, msg.Channel)
				log.Println("connectToTwitch")
				client := connectToTwitch(user.AccessToken, msg.Channel)
				channel.Name = msg.Channel
				// Client is returned but the state might not be connected
				channel.Client = client

				user.Channel = channel

				b, err := json.Marshal(user)
				if err != nil {
					fmt.Printf("Error: %s", err)
					return
				}
				db.Put([]byte(cookie.Value), b, nil)

				fmt.Println("Connect started")
			} else {
				log.Println("user already connected or invalid channel name")
			}
		} else if msg.Key == "disconnect" && authenticated {
			// TODO disconnect from the channel here
			user.disconnect(msg.Channel)
			log.Println(msg.Command, msg.Text)
		} else if msg.Key == "createcommand" && authenticated {
			// TODO create a command
			log.Println(msg.Command, msg.Text)
		} else if msg.Key == "removecommand" && authenticated {
			// TODO remove a command
			log.Println(msg.Command, msg.Text)
		} else {
			log.Printf("No matching command found: '%s'\n", msg.Key)
		}
	}
}

func (u User) isConnected(channel string) bool {
	if u.Channel.Name == channel {
		return true
	}

	return false
}

func (u User) disconnect(channel string) {
	if u.Channel.Name == channel {
		err := u.Channel.Client.Disconnect()
		if err != nil {
			log.Println(err)
		}
		return
	}

}

func (u User) createCommand(command Command) bool {
	for _, c := range u.Commands {
		if c.Input == command.Input {
			return false
		}
	}
	u.Commands = append(u.Commands, command)
	return true
}

func (u User) removeCommand(command Command) bool {
	for i, c := range u.Commands {
		if c.Input == command.Input {
			u.Commands = append(u.Commands[:i], u.Commands[i+1:]...)
			return true
		}
	}
	return false
}
