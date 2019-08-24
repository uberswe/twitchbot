package main

import (
	"encoding/json"
	"fmt"
	"github.com/gempir/go-twitch-irc"
	"github.com/gorilla/websocket"
	"github.com/syndtr/goleveldb/leveldb"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

var (
	cookieName = "botbyuber"
	clients    = make(map[*websocket.Conn]bool) // connected clients
	broadcast  = make(chan Message)             // broadcast channel
	upgrader   websocket.Upgrader
	db         *leveldb.DB
)

// Define our message object
type Message struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Template struct {
	AuthToken string
}

type HashRequest struct {
	Hash string
}

type User struct {
	AccessToken string
	Scopes      []string
	TokenType   string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func main() {

	var err error
	// Database handling

	db, err = leveldb.OpenFile("uberswe.db", nil)

	defer db.Close()

	// Configure the upgrader
	upgrader = websocket.Upgrader{}

	http.HandleFunc("/ws", handleConnections)
	// Start listening for incoming chat messages
	go handleMessages()

	// Routes

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("callback.html"))
		// https://bot.uberswe.com/callback#access_token=fau80sjur5xhks8px0sq28jsy1hnak&scope=bits%3Aread+clips%3Aedit+user%3Aread%3Abroadcast+user%3Aread%3Aemail&token_type=bearer
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			log.Printf("Cant find cookie :/\r\n")
			return
		}

		data, err := db.Get([]byte(cookie.Value), nil)

		var userObj User

		err = json.Unmarshal(data, &userObj)
		if err != nil {
			log.Println(err)
		}

		log.Println(fmt.Sprintf("oauth:%s", userObj.AccessToken))

		t := Template{
			AuthToken: cookie.Value,
		}

		tmpl := template.Must(template.ParseFiles("admin.html"))
		err = tmpl.Execute(w, t)

		if err != nil {
			log.Println(err)
			return
		}

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/" {
			scopes := "bits:read clips:edit user:read:broadcast chat:read chat:edit channel:moderate whispers:read whispers:edit channel_editor"
			clientID := "3en0x0g6wt7xcm0ra8z0p4fvq5bc34"
			redirectURL := "https://bot.uberswe.com/callback"
			responseType := "token"
			http.Redirect(w, r, fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s", clientID, redirectURL, responseType, scopes), 302)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		// Parse the URL and ensure there are no errors.

		decoder := json.NewDecoder(r.Body)
		var hr HashRequest
		err := decoder.Decode(&hr)
		if err != nil {
			log.Println(err)
		}

		vals, err := url.ParseQuery(hr.Hash)
		if err != nil {
			log.Println(err)
			return
		}

		key := RandString(155)

		if vals["#access_token"] != nil {
			user := User{
				AccessToken: vals["#access_token"][0],
				Scopes:      vals["scope"],
				TokenType:   vals["token_type"][0],
			}

			b, err := json.Marshal(user)
			if err != nil {
				fmt.Printf("Error: %s", err)
				return
			}

			err = db.Put([]byte(key), b, nil)

			if err != nil {
				log.Println(err)
				return
			}

			cookie := http.Cookie{
				Name:  cookieName,
				Value: key,
			}

			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	})

	fmt.Println("Listening on port 8010")
	err = http.ListenAndServe(":8010", nil)

	if err != nil {
		panic(err)
	}
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
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	var user User
	connected := false
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
		}
	}

	initmsg := Message{
		Key:   "init",
		Value: "welcome",
	}

	broadcast <- initmsg

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		if msg.Key == "connect" && authenticated {
			if len(msg.Value) > 1 && len(msg.Value) < 70 && !connected {
				log.Printf("Connect to channel %s: %s\n", user.AccessToken, msg.Value)
				connectToTwitch(user.AccessToken, msg.Value)
				connected = true
			}
		}
	}
}

func connectToTwitch(accessToken string, channel string) {
	client := twitch.NewClient("uberswe", fmt.Sprintf("oauth:%s", accessToken))

	client.OnConnect(func() {
		log.Println("Client connected")

		initmsg := Message{
			Key:   "channel",
			Value: channel,
		}

		broadcast <- initmsg
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		log.Println(message.User.Name)
		log.Println(message.User.ID)
		log.Println(message.ID)
		log.Println(message.Channel)
		log.Println(message.Bits)
		log.Println(message.Tags)
		initmsg := Message{
			Key:   "message",
			Value: message.Message,
		}

		broadcast <- initmsg
	})

	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		//  Raw       string
		//	Type      MessageType
		//      UNSET is for message types we currently don't support
		//	UNSET MessageType = -1
		//	// WHISPER private messages
		//	WHISPER MessageType = 0
		//	// PRIVMSG standard chat message
		//	PRIVMSG MessageType = 1
		//	// CLEARCHAT timeout messages
		//	CLEARCHAT MessageType = 2
		//	// ROOMSTATE changes like sub mode
		//	ROOMSTATE MessageType = 3
		//	// USERNOTICE messages like subs, resubs, raids, etc
		//	USERNOTICE MessageType = 4
		//	// USERSTATE messages
		//	USERSTATE MessageType = 5
		//	// NOTICE messages like sub mode, host on
		//	NOTICE MessageType = 6
		//	// JOIN whenever a user joins a channel
		//	JOIN MessageType = 7
		//	// PART whenever a user parts from a channel
		//	PART MessageType = 8
		//	// RECONNECT is sent from Twitch when they request the client to reconnect (i.e. for an irc server restart) https://dev.twitch.tv/docs/irc/commands/#reconnect-twitch-commands
		//	RECONNECT MessageType = 9
		//	// NAMES (or 353 https://www.alien.net.au/irc/irc2numerics.html#353) is the response sent from the server when the client requests a list of names for a channel
		//	NAMES MessageType = 10
		//	// PING is a message that can be sent from the IRC server. go-twitch-irc responds to PINGs automatically
		//	PING MessageType = 11
		//	// PONG is a message that should be sent from the IRC server as a response to us sending a PING message.
		//	PONG MessageType = 12
		//	// CLEARMSG whenever a single message is deleted
		//	CLEARMSG MessageType = 13
		//
		//	RawType   string
		//	Tags      map[string]string
		//	Message   string
		//	Channel   string
		//	RoomID    string
		//	ID        string
		//	Time      time.Time
		//	Emotes    []*Emote
		//	MsgID     string
		//	MsgParams map[string]string
		//	SystemMsg string
		jsonString, err := json.Marshal(message.MsgParams)

		if err != nil {
			log.Println(err)
			return
		}

		log.Println(fmt.Printf("New notice: %s", string(jsonString)))

		initmsg := Message{
			Key:   "notice",
			Value: string(jsonString),
		}

		broadcast <- initmsg
	})

	client.Join(channel)

	go func() {
		err := client.Connect()
		if err != nil {
			log.Println(err)
		}
	}()
}
