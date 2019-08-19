package main

import (
	"encoding/json"
	"fmt"
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
)

// Define our message object
type Message struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	// Database handling

	db, err := leveldb.OpenFile("uberswe.db", nil)

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
		tmpl := template.Must(template.ParseFiles("admin.html"))
		err := tmpl.Execute(w, nil)

		cookie, err := r.Cookie(cookieName)
		if err != nil {
			log.Printf("Cant find cookie :/\r\n")
			return
		}

		data, err := db.Get([]byte(cookie.Value), nil)

		log.Println("fetching stored data")
		log.Println(string(data))

		iter := db.NewIterator(nil, nil)
		for iter.Next() {
			// Remember that the contents of the returned slice should not be modified, and
			// only valid until the next call to Next.
			key := iter.Key()
			log.Printf("Iterating key %s \n", string(key))
		}
		iter.Release()
		err = iter.Error()

		if err != nil {
			log.Fatal(err)
		}

		if err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/" {
			scopes := "bits:read clips:edit user:read:broadcast user:read:email"
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
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()
	// Register our new client
	clients[ws] = true

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}
