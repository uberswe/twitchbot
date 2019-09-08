package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

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
		} else if msg.Key == "disconnect" && authenticated {
			// TODO disconnect from the channel here
			connected = false
		} else if msg.Key == "createcommand" && authenticated {
			// TODO create a command
			connected = false
		} else if msg.Key == "removecommand" && authenticated {
			// TODO remove a command
			connected = false
		}
	}
}
