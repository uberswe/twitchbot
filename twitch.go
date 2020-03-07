package botsbyuberswe

import (
	"bytes"
	"encoding/json"
	"fmt"
	twitch "github.com/gempir/go-twitch-irc/v2"
	"github.com/matryer/anno"
	"github.com/nicklaw5/helix"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"time"
)

func twitchIRCHandler() {
	iter := db.NewIterator(util.BytesPrefix([]byte("user:")), nil)
	for iter.Next() {
		var user User
		err := json.Unmarshal(iter.Value(), &user)
		if err != nil {
			log.Println(err)
			continue
		}

		if _, ok := clientConnections[user.TwitchID]; !ok {
			clientConnections[user.TwitchID] = connectToTwitch(user)
			user.Connected = true
			b, err := json.Marshal(user)
			if err != nil {
				log.Printf("Error: %s", err)
				return
			}

			// We store the user object with the twitchID for reference
			err = db.Put([]byte(fmt.Sprintf("user:%s", user.TwitchID)), b, nil)

			if err != nil {
				log.Println(err)
				return
			}
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Println(err)
	}
	// Sleep for 10 minutes
	time.Sleep(5 * time.Second)
}

// refreshHandler refreshes tokens every 10 minutes if needed
func refreshHandler() {
	for {
		// After 10 minutes we try to refresh our tokens
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

			// if user token expires in the next 10 min
			if user.TokenExpiry.Before(time.Now().Add(2 * time.Hour)) {
				log.Printf("Refreshing tokens for: %s\n", user.TwitchID)
				client, err := helix.NewClient(&helix.Options{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					RedirectURI:  redirectURL,
				})
				if err != nil {
					log.Println(err)
					continue
				}
				refreshResponse, err := client.RefreshUserAccessToken(user.RefreshToken)
				if err != nil {
					log.Println(err)
					continue
				}
				user.RefreshToken = refreshResponse.Data.RefreshToken
				user.AccessToken = refreshResponse.Data.AccessToken

				tokenExpiry := time.Now().Add(time.Duration(refreshResponse.Data.ExpiresIn) * time.Second)

				log.Printf("Refreshed: New tokens should refresh at %s", tokenExpiry.String())

				user.TokenExpiry = tokenExpiry

				b, err := json.Marshal(user)
				if err != nil {
					log.Printf("Error: %s", err)
					return
				}

				// We store the user object with the twitchID for reference
				err = db.Put([]byte(fmt.Sprintf("user:%s", user.TwitchID)), b, nil)

				if err != nil {
					log.Println(err)
					return
				}
			}
		}
		iter.Release()
		err := iter.Error()
		if err != nil {
			log.Println(err)
		}
		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
	}
}

func reconnectHandler(user User) {
	var updatedUser User

	log.Printf("Reconnecting to Twitch %s\n", user.TwitchID)

	data, err := db.Get([]byte(fmt.Sprintf("user:%s", user.TwitchID)), nil)

	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(data, &updatedUser)

	if err != nil {
		log.Println(err)
	}

	updatedUser.Connected = false

	client := connectToTwitch(updatedUser)

	clientConnections[user.TwitchID] = client

	b, err := json.Marshal(updatedUser)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	updatedUser.Connected = true
	db.Put([]byte(fmt.Sprintf("user:%s", updatedUser.TwitchID)), b, nil)

	log.Println("Connect started for reconnect")
}

func connectToTwitch(user User) *twitch.Client {
	log.Println("creating twitch client")
	client := twitch.NewClient(user.Channel.Name, fmt.Sprintf("oauth:%s", user.AccessToken))

	log.Println("configuring twitch client")
	client.OnConnect(func() {
		log.Println("Client connected")

		initmsg := WebsocketMessage{
			Key:     "channel",
			Channel: user.Channel.Name,
		}

		broadcast <- initmsg
	})

	client.OnPingMessage(func(message twitch.PingMessage) {
		log.Printf("Ping received: %s\n", message.Message)
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {

		commands := anno.FieldFunc("command", func(s []byte) (bool, []byte) {
			return bytes.HasPrefix(s, []byte("!")), s
		})

		n, err := anno.FindString(commands, message.Message)

		if err != nil {
			panic(err)
		}

		for _, note := range n {
			if note.Start == 0 {
				go handleCommand(string(note.Val), message)
			}
		}

		initmsg := WebsocketMessage{
			Key:            "notice",
			PrivateMessage: message,
		}

		broadcast <- initmsg
	})

	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		jsonString, err := json.Marshal(message.MsgParams)

		log.Println(fmt.Sprintf("New notice: %s %s", string(jsonString), message.MsgID))

		if message.MsgID == "raid" {
			// 2019/08/25 12:20:15 New notice: {"msg-param-displayName":"El_Funko","msg-param-login":"el_funko",
			// "msg-param-profileImageURL":"https://static-cdn.jtvnw.net/jtv_user_pictures/823e29e0-2bef-42a3-b0df-3d8755dbde53-profile_image-70x70.png",
			// "msg-param-viewerCount":"38"} raid
		} else if message.MsgID == "host" {

		}

		// 2019/08/25 13:07:31 New notice: {"msg-param-cumulative-months":"1","msg-param-months":"0","msg-param-should-share-streak":"0","msg-param-sub-plan":"Prime","msg-param-sub-plan-name":"Conscript for war"} sub
		// 2019/08/25 13:07:42 New notice: {"msg-param-cumulative-months":"7","msg-param-months":"0","msg-param-should-share-streak":"0","msg-param-sub-plan":"1000","msg-param-sub-plan-name":"Conscript for war"} resub
		// 2019/08/25 11:33:06 New notice: {"msg-param-months":"1","msg-param-origin-id":"da 39 a3 ee 5e 6b 4b 0d 32 55 bf ef 95 60 18 90 af d8 07 09","msg-param-recipient-display-name":"clearancewater","msg-param-recipient-id":"229767697","msg-param-recipient-user-name":"clearancewater","msg-param-sender-count":"0","msg-param-sub-plan":"1000","msg-param-sub-plan-name":"Conscript for war"} subgift
		// 2019/08/25 11:33:05 New notice: {"msg-param-mass-gift-count":"1","msg-param-origin-id":"22 a3 a4 cd 7e 82 bd e9 2d ba e8 12 34 54 44 08 11 15 a6 e5","msg-param-sender-count":"1","msg-param-sub-plan":"1000"} submysterygift
		// 2019/08/25 11:32:14 New notice: {"msg-param-sender-login":"robust_meu","msg-param-sender-name":"RobUst_meu"} giftpaidupgrade
		// 2019/08/25 11:02:35 New notice: {"msg-param-bits-amount":"500","msg-param-domain":"seasonal-food-fight","msg-param-min-cheer-amount":"200","msg-param-selected-count":"10"} rewardgift

		if err != nil {
			log.Println(err)
			return
		}

		initmsg := WebsocketMessage{
			Key:       "notice",
			MsgParams: message.MsgParams,
		}

		broadcast <- initmsg
	})

	client.Join(user.Channel.Name)

	go func() {
		err := client.Connect()
		if err != nil {
			log.Printf("Error in twitch irc connection for %s\n", user.TwitchID)
			log.Println(err)
			time.Sleep(10 * time.Second)
			user.Connected = false
			reconnectHandler(user)
		}
	}()

	return client
}

func handleCommand(command string, message twitch.PrivateMessage) {
	log.Printf("Command detected: %s\n", command)
}
