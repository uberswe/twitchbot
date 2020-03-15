package botsbyuberswe

import (
	"encoding/json"
	"fmt"
	twitch "github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/websocket"
	"github.com/nicklaw5/helix"
	"log"
	"net/http"
	"time"
)

type Cookie struct {
	TwitchID string    `json:"twitch_id,omitempty"`
	Expiry   time.Time `json:"expiry,omitempty"`
}

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
	BotToken              string         `json:"bot_token,omitempty"`
	TwitchIRCClient       *twitch.Client `json:"-"`
	TwitchOAuthClient     *helix.Client  `json:"-"`
	TwitchConnectFailures int            `json:"twitch_connect_failures,omitempty"`
}

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
	UserTwitchID          string         `json:"user_twitch_id,omitempty"`
	TokenExpiry           time.Time      `json:"token_expiry,omitempty"`
}

type Command struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

type Channel struct {
	Name     string `json:"name,omitempty"`
	IsMod    bool   `json:"is_mod,omitempty"`
	LastHost string `json:"last_host,omitempty"`
	LastRaid string `json:"last_raid,omitempty"`
}

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

type Variable struct {
	Name        string    `json:"name,omitempty"`
	Value       string    `json:"value,omitempty"`
	Description string    `json:"description,omitempty"`
	Expiry      time.Time `json:"expiry,omitempty"`
}

type State struct {
	Commands  []Command  `json:"commands,omitempty"`
	Variables []Variable `json:"variables,omitempty"`
}

type BotToken struct {
	Token    string
	TwitchID string
}

// Short for WebsocketConnection
type Wconn struct {
	Connection *websocket.Conn
	TwitchID   string
}

func initWebsockets() {
	// Configure the upgrader
	upgrader = websocket.Upgrader{}

	r.HandleFunc("/ws", handleConnections)
	// Start listening for incoming chat messages
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
		} else {
			authenticated = true
			log.Println("Authenticated")
			statemsg := WebsocketMessage{
				Key:      "state",
				Channel:  user.Channel.Name,
				State:    user.State,
				TwitchID: user.TwitchID,
			}

			clients = append(clients, Wconn{
				Connection: ws,
				TwitchID:   user.TwitchID,
			})

			err := ws.WriteJSON(statemsg)
			if err != nil {
				log.Printf("Error: %s", err)
				index, err := getClientIndex(clients, user.TwitchID)
				if err != nil {
					log.Printf("Error: %s", err)
				} else {
					clients = deleteClient(clients, index)
					return
				}
			}
		}
	}

	if authenticated {
		if !user.Connected {
			log.Printf("Connect to channel %s: %s\n", user.AccessToken, user.Channel.Name)
			log.Println("connectToTwitch")
			user.TwitchIRCClient = connectToTwitch(user)

			user.Connected = true

			clientConnections[user.TwitchID] = user

			err = user.store()
			if err != nil {
				log.Printf("Error: %s", err)
				index, err := getClientIndex(clients, user.TwitchID)
				if err != nil {
					log.Printf("Error: %s", err)
				} else {
					clients = deleteClient(clients, index)
					return
				}
				return
			}
			log.Println("Connect started")
		} else if user.Connected {
			log.Println("user already connected")
			initmsg := WebsocketMessage{
				Key:      "channel",
				Channel:  user.Channel.Name,
				TwitchID: user.TwitchID,
			}

			err := ws.WriteJSON(initmsg)
			if err != nil {
				log.Printf("Error: %s", err)
				index, err := getClientIndex(clients, user.TwitchID)
				if err != nil {
					log.Printf("Error: %s", err)
				} else {
					clients = deleteClient(clients, index)
					return
				}
				return
			}
			// TODO check if the bot is connected
			if val, ok := botConnections[user.TwitchID]; ok {
				if val.Connected {
					initmsg := WebsocketMessage{
						Key:      "channel",
						Channel:  user.Channel.Name,
						TwitchID: user.TwitchID,
						BotName:  val.Name,
					}
					err := ws.WriteJSON(initmsg)
					if err != nil {
						log.Printf("Error: %s", err)
						index, err := getClientIndex(clients, user.TwitchID)
						if err != nil {
							log.Printf("Error: %s", err)
						} else {
							clients = deleteClient(clients, index)
							return
						}
						return
					}
				}
			} else {
				// If the user does not have it's own bot then it's part of the universal bot
				for _, connectedChannel := range universalConnectedChannels {
					if connectedChannel == user.Channel.Name {
						initmsg := WebsocketMessage{
							Key:      "channel",
							Channel:  user.Channel.Name,
							TwitchID: user.TwitchID,
							BotName:  botConnections[universalBotTwitchID].Name,
						}
						err := ws.WriteJSON(initmsg)
						if err != nil {
							log.Printf("Error: %s", err)
							index, err := getClientIndex(clients, user.TwitchID)
							if err != nil {
								log.Printf("Error: %s", err)
							} else {
								clients = deleteClient(clients, index)
								return
							}
							return
						}
					}
				}
			}
		} else {
			log.Println("invalid channel name")
		}
	}

	for {
		var msg WebsocketMessage
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
				// TODO validation?
				log.Println(msg.Command, msg.Text)
				user.createCommand(Command{
					Input:  msg.Command,
					Output: msg.Text,
				})
				err = user.store()

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

				statemsg := WebsocketMessage{
					Key:      "state",
					Channel:  user.Channel.Name,
					State:    user.State,
					TwitchID: user.TwitchID,
				}

				err := ws.WriteJSON(statemsg)
				if err != nil {
					log.Printf("Error: %s", err)
					index, err := getClientIndex(clients, user.TwitchID)
					if err != nil {
						log.Printf("Error: %s", err)
					} else {
						clients = deleteClient(clients, index)
						return
					}
					return
				}

				alertmsg := WebsocketMessage{
					Key:       "alert",
					Text:      "Command added successfully",
					AlertType: "success",
					TwitchID:  user.TwitchID,
				}

				err = ws.WriteJSON(alertmsg)
				if err != nil {
					log.Printf("Error: %s", err)
					index, err := getClientIndex(clients, user.TwitchID)
					if err != nil {
						log.Printf("Error: %s", err)
					} else {
						clients = deleteClient(clients, index)
						return
					}
					return
				}
			} else if msg.Key == "removecommand" {
				log.Println(msg.Command, msg.Text)
				for _, command := range user.State.Commands {
					if command.Input == msg.Text {
						user.removeCommand(command)
					}
				}

				err = user.store()

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

				statemsg := WebsocketMessage{
					Key:      "state",
					Channel:  user.Channel.Name,
					State:    user.State,
					TwitchID: user.TwitchID,
				}

				err := ws.WriteJSON(statemsg)
				if err != nil {
					log.Printf("Error: %s", err)
					index, err := getClientIndex(clients, user.TwitchID)
					if err != nil {
						log.Printf("Error: %s", err)
					} else {
						clients = deleteClient(clients, index)
						return
					}
					return
				}

				alertmsg := WebsocketMessage{
					Key:       "alert",
					Text:      "Command removed successfully",
					AlertType: "success",
					TwitchID:  user.TwitchID,
				}

				err = ws.WriteJSON(alertmsg)
				if err != nil {
					log.Printf("Error: %s", err)
					index, err := getClientIndex(clients, user.TwitchID)
					if err != nil {
						log.Printf("Error: %s", err)
					} else {
						clients = deleteClient(clients, index)
						return
					}
					return
				}
			} else {
				log.Printf("No matching command found: '%s'\n", msg.Key)
			}
		}
	}
}

func getUserFromCookie(cookie *http.Cookie) (User, error) {
	var cookieObj Cookie
	var user User
	log.Printf("cookie val: %s", cookie.Value)
	data, err := db.Get([]byte(fmt.Sprintf("cookie:%s", cookie.Value)), nil)
	err = json.Unmarshal(data, &cookieObj)
	if err != nil {
		return user, nil
	}
	return getUserFromTwitchID(cookieObj.TwitchID)
}

func getUserFromTwitchID(twitchID string) (User, error) {
	var user User
	data, err := db.Get([]byte(fmt.Sprintf("user:%s", twitchID)), nil)
	if err != nil {
		log.Println(err)
		return user, err
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		log.Println(err)
		return user, err
	}
	return user, nil
}

func (u *User) store() error {
	b, err := json.Marshal(u)
	if err != nil {
		log.Printf("Error: %s", err)
		return err
	}
	return db.Put([]byte(fmt.Sprintf("user:%s", u.TwitchID)), b, nil)
}

func (u *User) createCommand(command Command) bool {
	for _, c := range u.State.Commands {
		if c.Input == command.Input {
			return false
		}
	}
	u.State.Commands = append(u.State.Commands, command)
	return true
}

func (u *User) removeCommand(command Command) bool {
	for i, c := range u.State.Commands {
		if c.Input == command.Input {
			u.State.Commands = deleteCommand(u.State.Commands, i)
			return true
		}
	}
	return false
}
