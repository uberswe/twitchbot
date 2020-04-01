package botsbyuberswe

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	cookieName     = "botbyuber"
	defaultBot     = "botbyuber"
	clients        []Wconn                       // connected websocket clients
	broadcast      = make(chan WebsocketMessage) // broadcast channel
	universalBot   = make(chan ConnectChannel)   // broadcast channel
	upgrader       websocket.Upgrader
	db             *leveldb.DB
	clientID       string
	clientSecret   string
	redirectURL    = "/callback"
	botRedirectURL = "/bot/callback"
	// The twitch IRC clients for users
	clientConnections = make(map[string]User)
	// The botbyuber bot and other custom bots that can write to channels
	botConnections             = make(map[string]Bot)
	r                          *mux.Router
	universalBotTwitchID       = ""
	universalConnectedChannels []string
)

// Init is called before run and currently makes a seed for random number generators
func Init() {
	rand.Seed(time.Now().UnixNano())
}

// Run runs the application, it loads the config, starts everything and keeps it running
func Run() {
	// Load environmental variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	cookieName = os.Getenv("COOKIE_NAME")
	defaultBot = os.Getenv("UNIVERSAL_BOT")

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

	// Initialize routes
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
