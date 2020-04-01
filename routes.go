package botsbyuberswe

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nicklaw5/helix"
	"github.com/syndtr/goleveldb/leveldb/util"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func routes() {

	fs := http.FileServer(http.Dir("assets"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	r.HandleFunc("/callback", callback)

	// TODO redirect index if not authenticated
	r.HandleFunc("/admin", admin)

	r.HandleFunc("/login", login)

	// TODO redirect index if authenticated already
	r.HandleFunc("/", index)

	r.HandleFunc("/bot/callback", botCallback)

	r.HandleFunc("/bot/{key}", addBot)
}

func addBot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Key: %v\n", vars["key"])
	scopes := "chat:read channel:moderate chat:edit whispers:read whispers:edit"
	rURL := fmt.Sprintf("https://%s%s", r.Host, botRedirectURL)
	if strings.Contains(r.Host, "localhost") {
		rURL = fmt.Sprintf("http://%s%s", r.Host, botRedirectURL)
	}
	responseType := "code"
	http.Redirect(w, r, fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s&force_verify=%s&state=%s", clientID, rURL, responseType, scopes, "true", vars["key"]), 302)
	return
}

// getModHash returns a string based on when the file or any included files was last modified, currently just a nano timestamp
func getModHash(file string) string {
	hasher := sha1.New()
	info, err := os.Stat(file)
	if err != nil {
		log.Println(err)
		hasher.Write([]byte("-1"))
		return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	}
	modTime := info.ModTime()
	files, err := ioutil.ReadDir("assets/css/")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.ModTime().After(modTime) {
			hasher.Write([]byte(strconv.FormatInt(f.ModTime().UnixNano(), 10)))
			return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		}
	}
	files, err = ioutil.ReadDir("assets/js/")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.ModTime().After(modTime) {
			hasher.Write([]byte(strconv.FormatInt(f.ModTime().UnixNano(), 10)))
			return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		}
	}
	hasher.Write([]byte(strconv.FormatInt(modTime.UnixNano(), 10)))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func loadTemplateFile(file string, w http.ResponseWriter) {
	tmpl := template.Must(template.ParseFiles(file))
	err := tmpl.Execute(w, Template{
		ModifiedHash: getModHash(file),
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	loadTemplateFile("assets/html/index.html", w)
}

func callback(w http.ResponseWriter, r *http.Request) {
	rURL := fmt.Sprintf("https://%s%s", r.Host, redirectURL)
	if strings.Contains(r.Host, "localhost") {
		rURL = fmt.Sprintf("http://%s%s", r.Host, redirectURL)
	}

	// The following is an example of the callback request
	// ?code=1b4h2pcqfgpzu5r6z4we5st0qe7nri&scope=chat%3Aread+user%3Aread%3Abroadcast+bits%3Aread+channel%3Aread%3Asubscriptions+analytics%3Aread%3Agames+analytics%3Aread%3Aextensions&state=uberstate
	if val, ok := r.URL.Query()["code"]; ok && len(val) > 0 {
		log.Println(val)
		// TODO we can keep the client for the entire application?? No need to make a new one every time?
		client, err := helix.NewClient(&helix.Options{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  rURL,
		})

		if err != nil {
			http.Error(w, "Unexpected response from Twitch, please try again!", 500)
			return
		}

		resp, err := client.GetUserAccessToken(val[0])
		if err != nil {
			http.Error(w, "Unexpected response from Twitch, please try again!", 500)
			return
		}

		log.Printf("%+v\n", resp)

		// This key is generated for our cookie
		// TODO consider using HMAC with IP/useragent validation for additional verification
		// TODO or change this to use UUID for generation
		key := RandString(155)

		tokenExpiry := time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)

		log.Printf("Tokens should refresh at %s", tokenExpiry.String())

		client.SetUserAccessToken(resp.Data.AccessToken)

		userResponse, err := client.GetUsers(&helix.UsersParams{})

		if err != nil {
			log.Println(err)
			return
		}

		channelName := ""
		twitchID := ""
		email := ""
		cookieExpiry := time.Now().AddDate(1, 0, 0)

		for _, user := range userResponse.Data.Users {
			log.Printf("%+v\n", user)
			channelName = user.Login
			twitchID = user.ID
			email = user.Email
		}

		cookieModel := Cookie{
			TwitchID: twitchID,
			Expiry:   cookieExpiry,
		}

		user, err := getUserFromTwitchID(twitchID)

		// If the user does not exist we create a new user
		if err != nil {
			botToken := RandString(30)
			user = User{
				TwitchID:     twitchID,
				Email:        email,
				AccessCode:   val[0],
				AccessToken:  resp.Data.AccessToken,
				RefreshToken: resp.Data.RefreshToken,
				TokenExpiry:  tokenExpiry,
				Scopes:       resp.Data.Scopes,
				TokenType:    "code",
				BotToken:     botToken,
				Channel: Channel{
					Name: channelName,
				},
				State: State{
					Commands: []Command{
						// This is the default command for every new user
						{
							Input:  "!so {user}",
							Output: "Check out and follow @{user}! https://twitch.tv/{user}",
						},
					},
					Variables: []Variable{
						{
							Name:        "user",
							Description: "If your command specified a user such as <b>@uberswe</b>.",
						},
						{
							Name:        "lasthost",
							Description: "This will be the user who last hosted your channel.",
						},
						{
							Name:        "lastraid",
							Description: "This will be the user who last raided your channel.",
						},
						{
							Name:        "lasthostraid",
							Description: "This will be the user who last hosted or raided your channel.",
						},
					},
				},
			}

			btoken := BotToken{
				Token:    botToken,
				TwitchID: twitchID,
			}

			// We store the bot token object as a reference
			err = btoken.store()

			// We store the user object with the twitchID for reference
			err = user.store()

			if err != nil {
				log.Println(err)
				return
			}

			// Connect universal bot for new user
			connect := ConnectChannel{
				Name:    user.Channel.Name,
				Connect: true,
			}
			universalBot <- connect
		}

		// We then store the cookie which has a reference to the twitchID
		err = cookieModel.store(key)

		if err != nil {
			log.Println(err)
			return
		}

		cookie := http.Cookie{
			Name:    cookieName,
			Value:   key, // TODO is this random key safe?
			Expires: cookieExpiry,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/admin", 302)

		return
	}
	http.Error(w, "Unexpected response from Twitch, please try again!", 500)
	return

}

func botCallback(w http.ResponseWriter, r *http.Request) {
	rURL := fmt.Sprintf("https://%s%s", r.Host, botRedirectURL)
	if strings.Contains(r.Host, "localhost") {
		rURL = fmt.Sprintf("http://%s%s", r.Host, botRedirectURL)
	}
	// For a bot callback we want to check the state and code, the state is our bot token
	state, stateOk := r.URL.Query()["state"]
	code, codeOk := r.URL.Query()["code"]
	if stateOk && codeOk && len(state) > 0 && len(code) > 0 {
		log.Printf("botCallback state: %s", state[0])
		log.Printf("botCallback code: %s", code[0])

		// This client is used to get the initial user access token, we should not need this after this initial authentication
		client, err := helix.NewClient(&helix.Options{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  rURL,
		})

		if err != nil {
			http.Error(w, "Unexpected response from Twitch, please try again!", 500)
			return
		}

		resp, err := client.GetUserAccessToken(code[0])
		if err != nil {
			http.Error(w, "Unexpected response from Twitch, please try again!", 500)
			return
		}

		log.Printf("%+v\n", resp)

		client.SetUserAccessToken(resp.Data.AccessToken)

		userResponse, err := client.GetUsers(&helix.UsersParams{})

		if err != nil {
			log.Println(err)
			return
		}

		channelName := ""

		for _, u := range userResponse.Data.Users {
			log.Printf("%+v\n", u)
			channelName = u.Login
		}

		iter := db.NewIterator(util.BytesPrefix([]byte("user:")), nil)
		for iter.Next() {
			// Use key/value.
			log.Println(string(iter.Key()))
			log.Println(string(iter.Value()))
			var user User
			err := json.Unmarshal(iter.Value(), &user)
			if err != nil {
				log.Println(err)
				continue
			}

			if user.BotToken == state[0] {

				data2, err := db.Get([]byte(fmt.Sprintf("bot:%s", user.TwitchID)), nil)

				if err != nil {
					tokenExpiry := time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)

					if channelName == defaultBot {
						log.Printf("Universal bot id found: %s\n", user.TwitchID)
						universalBotTwitchID = user.TwitchID
					}

					bot := Bot{
						Name:            channelName,
						UserTwitchID:    user.TwitchID,
						TokenExpiry:     tokenExpiry,
						AccessToken:     resp.Data.AccessToken,
						RefreshToken:    resp.Data.RefreshToken,
						UserChannelName: user.Channel.Name,
					}

					bot.TwitchIRCClient = connectBotToTwitch(bot)

					bot.Connected = true

					botConnections[bot.UserTwitchID] = bot

					// We store the user object with the twitchID for reference
					err = bot.store()

					if err != nil {
						log.Println(err)
						return
					}
				} else {
					var bot Bot

					err = json.Unmarshal(data2, &bot)
					if err != nil {
						log.Println(err)
					}
				}

			}
		}

		filename := "assets/html/success.html"

		t := Template{
			ModifiedHash: getModHash(filename),
		}

		tmpl := template.Must(template.ParseFiles(filename))
		err = tmpl.Execute(w, t)
		return
	}
	http.Error(w, "Unexpected response from Twitch, please try again!", 500)
	return

}

func admin(w http.ResponseWriter, r *http.Request) {
	filename := "assets/html/admin.html"
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		log.Printf("Cant find cookie :/\r\n")
		return
	}

	data, err := db.Get([]byte(fmt.Sprintf("cookie:%s", cookie.Value)), nil)

	var cookieObj Cookie

	err = json.Unmarshal(data, &cookieObj)
	if err != nil {
		log.Println(err)
	}

	data2, err := db.Get([]byte(fmt.Sprintf("user:%s", cookieObj.TwitchID)), nil)

	var userObj User

	err = json.Unmarshal(data2, &userObj)
	if err != nil {
		log.Println(err)
	}

	log.Println(fmt.Sprintf("oauth:%s", userObj.AccessToken))

	botURL := fmt.Sprintf("http://%s/bot/%s", r.Host, userObj.BotToken)
	if strings.Contains(r.Host, "localhost") {
		botURL = fmt.Sprintf("http://%s/bot/%s", r.Host, userObj.BotToken)
	}

	t := Template{
		ModifiedHash: getModHash(filename),
		BotURL:       botURL,
		BotName:      getBotName(userObj),
		BotConnected: isBotConnected(userObj),
	}

	tmpl := template.Must(template.ParseFiles(filename))
	err = tmpl.Execute(w, t)

	if err != nil {
		log.Println(err)
		return
	}

}

func getBotName(user User) string {
	bot, ok := botConnections[user.TwitchID]
	if !ok {
		bot2, ok2 := botConnections[universalBotTwitchID]
		if !ok2 {
			return ""
		}
		return bot2.Name
	}
	return bot.Name
}

func isBotConnected(user User) bool {
	_, ok := botConnections[user.TwitchID]
	if !ok {
		return false
	}
	return true
}

func login(w http.ResponseWriter, r *http.Request) {
	scopes := "chat:read user:read:broadcast bits:read channel:read:subscriptions analytics:read:games analytics:read:extensions"
	rURL := fmt.Sprintf("https://%s%s", r.Host, redirectURL)
	if strings.Contains(r.Host, "localhost") {
		rURL = fmt.Sprintf("http://%s%s", r.Host, redirectURL)
	}
	responseType := "code"
	// TODO set the state to a CSRF token and verify
	http.Redirect(w, r, fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s&force_verify=%s&state=%s", clientID, rURL, responseType, scopes, "true", "uberstate"), 302)
	return

}
