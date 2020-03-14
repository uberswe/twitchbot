package botsbyuberswe

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	cookieName   = "botbyuber"
	clients      []Wconn                       // connected websocket clients
	broadcast    = make(chan WebsocketMessage) // broadcast channel
	universalBot = make(chan ConnectChannel)   // broadcast channel
	upgrader     websocket.Upgrader
	db           *leveldb.DB
	clientID     = "***REMOVED***"
	clientSecret = "***REMOVED***"
	redirectURL  = "https://bots.uberswe.com/callback"
	// The twitch IRC clients for users
	clientConnections = make(map[string]User)
	// The botbyuber bot and other custom bots that can write to channels
	botConnections       = make(map[string]Bot)
	r                    *mux.Router
	universalBotTwitchID = ""
)

// Define our message object

type Template struct {
	ModifiedHash string
	BotUrl       string
}

type ConnectChannel struct {
	Name    string
	Connect bool
}

type HashRequest struct {
	Hash string
}

func Init() {
	rand.Seed(time.Now().UnixNano())
}

func Run() {

	var err error

	// Log handling
	f, err := os.OpenFile("uberswe.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	// Database handling
	db, err = leveldb.OpenFile("uberswe.db", nil)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Mux router
	r = mux.NewRouter()

	// Websockets
	// starts it's own thread for websocket connections
	initWebsockets()

	// Routes

	routes()

	// Refresh tokens every 10 min
	go refreshHandler()

	// Connect to twitch channels
	go twitchIRCHandler()

	// Connect to needed channels with the main IRC bot botbyuber
	go handleMainBotConnects()

	log.Println("Listening on port 8010")
	srv := &http.Server{
		Handler: r,
		Addr:    ":8010",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
