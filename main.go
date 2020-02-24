package botsbyuberswe

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
	broadcast  = make(chan WebsocketMessage)    // broadcast channel
	upgrader   websocket.Upgrader
	db         *leveldb.DB
)

// Define our message object

type Template struct {
	AuthToken    string
	ModifiedHash string
}

type HashRequest struct {
	Hash string
}

func Init() {
	rand.Seed(time.Now().UnixNano())
}

func Run() {

	var err error
	// Database handling

	db, err = leveldb.OpenFile("uberswe.db", nil)

	if err != nil {
		panic(err)
	}

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
