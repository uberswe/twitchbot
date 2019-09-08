package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/syndtr/goleveldb/leveldb"
	"math/rand"
	"net/http"
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

func main() {

	var err error
	// Database handling

	db, err = leveldb.OpenFile("uberswe.db", nil)

	defer db.Close()

	// Websockets

	initWebsockets()

	// Routes

	routes()

	fmt.Println("Listening on port 8010")
	err = http.ListenAndServe(":8010", nil)

	if err != nil {
		panic(err)
	}
}
