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
	"strings"
	"time"
)

func twitchIRCHandler() {
	// Iterate and connect bots
	BotIter := db.NewIterator(util.BytesPrefix([]byte("bot:")), nil)
	for BotIter.Next() {
		var bot Bot
		err := json.Unmarshal(BotIter.Value(), &bot)
		if err != nil {
			log.Println(err)
			continue
		}

		if _, ok := botConnections[bot.UserTwitchID]; !ok {
			if bot.Name == defaultBot {
				log.Printf("Universal bot id found: %s\n", bot.UserTwitchID)
				universalBotTwitchID = bot.UserTwitchID
			}
			bot.TwitchIRCClient = connectBotToTwitch(bot)
			bot.Connected = true
			botConnections[bot.UserTwitchID] = bot
		}
	}
	BotIter.Release()
	err := BotIter.Error()
	if err != nil {
		log.Println(err)
	}
	// Iterate and connect users
	iter := db.NewIterator(util.BytesPrefix([]byte("user:")), nil)
	for iter.Next() {
		var user User
		err := json.Unmarshal(iter.Value(), &user)
		if err != nil {
			log.Println(err)
			continue
		}

		if _, ok := clientConnections[user.TwitchID]; !ok {
			user.TwitchIRCClient = connectToTwitch(user)
			clientConnections[user.TwitchID] = user
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
	err = iter.Error()
	if err != nil {
		log.Println(err)
	}
}

// refreshHandler refreshes tokens every 10 minutes if needed
func refreshHandler() {
	for {
		// After 10 minutes we try to refresh our tokens
		iter := db.NewIterator(util.BytesPrefix([]byte("user:")), nil)
		for iter.Next() {
			// Use key/value.
			log.Printf("Refreshing tokens of user %s: %s\n", string(iter.Key()), string(iter.Value()))
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
		// Renew bot tokens

		// After 10 minutes we try to refresh our tokens
		botIter := db.NewIterator(util.BytesPrefix([]byte("bot:")), nil)
		for botIter.Next() {
			log.Printf("Refreshing tokens of bot %s: %s\n", string(botIter.Key()), string(botIter.Value()))

			var bot Bot
			err := json.Unmarshal(botIter.Value(), &bot)
			if err != nil {
				log.Println(err)
				continue
			}

			// if user token expires in the next 10 min
			if bot.TokenExpiry.Before(time.Now().Add(2 * time.Hour)) {
				log.Printf("Refreshing tokens for bot: %s\n", bot.UserTwitchID)
				client, err := helix.NewClient(&helix.Options{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					RedirectURI:  redirectURL,
				})
				if err != nil {
					log.Println(err)
					continue
				}
				refreshResponse, err := client.RefreshUserAccessToken(bot.RefreshToken)
				if err != nil {
					log.Println(err)
					continue
				}
				bot.RefreshToken = refreshResponse.Data.RefreshToken
				bot.AccessToken = refreshResponse.Data.AccessToken

				tokenExpiry := time.Now().Add(time.Duration(refreshResponse.Data.ExpiresIn) * time.Second)

				log.Printf("Bot Refreshed: New tokens should refresh at %s", tokenExpiry.String())

				bot.TokenExpiry = tokenExpiry

				b, err := json.Marshal(bot)
				if err != nil {
					log.Printf("Error: %s", err)
					return
				}

				// We store the user object with the twitchID for reference
				err = db.Put([]byte(fmt.Sprintf("bot:%s", bot.UserTwitchID)), b, nil)

				if err != nil {
					log.Println(err)
					return
				}
			}
		}
		botIter.Release()
		err = botIter.Error()
		if err != nil {
			log.Println(err)
		}
		// Sleep for 10 minutes
		time.Sleep(10 * time.Minute)
	}
}

func reconnectHandler(user User) {

	log.Printf("Reconnecting to Twitch %s\n", user.TwitchID)

	data, err := db.Get([]byte(fmt.Sprintf("user:%s", user.TwitchID)), nil)

	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(data, &user)

	if err != nil {
		log.Println(err)
		return
	}

	user.Connected = false

	user.TwitchIRCClient = connectToTwitch(user)

	user.Connected = true
	user.TwitchConnectFailures++

	clientConnections[user.TwitchID] = user

	b, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	db.Put([]byte(fmt.Sprintf("user:%s", user.TwitchID)), b, nil)

	log.Println("Connect started for reconnect")
}

func reconnectBotHandler(bot Bot) {

	data, err := db.Get([]byte(fmt.Sprintf("bot:%s", bot.UserTwitchID)), nil)

	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(data, &bot)

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Reconnecting bot to Twitch %s\n", bot.UserTwitchID)

	bot.TwitchIRCClient = connectBotToTwitch(bot)

	bot.Connected = true

	botConnections[bot.UserTwitchID] = bot

}

func connectBotToTwitch(bot Bot) *twitch.Client {
	log.Println("creating twitch client")
	client := twitch.NewClient(bot.Name, fmt.Sprintf("oauth:%s", bot.AccessToken))

	log.Println("configuring twitch bot client")
	client.OnConnect(func() {
		log.Println("Client bot connected")

		initmsg := WebsocketMessage{
			Key:      "channel",
			Channel:  bot.UserChannelName,
			BotName:  bot.Name,
			TwitchID: bot.UserTwitchID,
		}

		broadcast <- initmsg
	})

	client.OnPingMessage(func(message twitch.PingMessage) {
		log.Printf("Bot Ping received: %s\n", message.Message)
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
				go handleCommand(bot, message.Message, client)
			}
		}

		initmsg := WebsocketMessage{
			Key:            "notice",
			PrivateMessage: message,
			TwitchID:       bot.UserTwitchID,
		}

		broadcast <- initmsg
	})

	client.Join(bot.UserChannelName)

	if bot.UserTwitchID == universalBotTwitchID {
		// loop through all users that don't have their own bot and connect to their channels if this is universal bot
		iter := db.NewIterator(util.BytesPrefix([]byte("user:")), nil)
		for iter.Next() {
			var user User
			err := json.Unmarshal(iter.Value(), &user)
			if err != nil {
				log.Println(err)
				continue
			}
			if _, ok := botConnections[user.TwitchID]; !ok {
				connect := ConnectChannel{
					Name:    user.Channel.Name,
					Connect: true,
				}
				universalBot <- connect
			}
		}
		iter.Release()
		err := iter.Error()
		if err != nil {
			log.Println(err)
		}
	} else {
		// if this is not the universal bot then remove this user from the universal bot
		connect := ConnectChannel{
			Name:    bot.UserChannelName,
			Connect: false,
		}
		universalBot <- connect
	}

	go func() {
		err := client.Connect()
		if err != nil {
			log.Println(err)
			time.Sleep(10 * time.Second)
			reconnectBotHandler(bot)
		}
	}()

	return client
}

func handleMainBotConnects() {
	for {
		// Grab the next message from the broadcast channel
		connect := <-universalBot

		if universalBotTwitchID != "" {
			_, ok := botConnections[universalBotTwitchID]
			if ok {
				if connect.Connect {
					log.Printf("Universal bot %s is joining %s\n", universalBotTwitchID, connect.Name)
					botConnections[universalBotTwitchID].TwitchIRCClient.Join(connect.Name)
					universalConnectedChannels = append(universalConnectedChannels, connect.Name)
				} else {
					log.Printf("Universal bot %s is leaving %s\n", universalBotTwitchID, connect.Name)
					botConnections[universalBotTwitchID].TwitchIRCClient.Depart(connect.Name)
				}
			}
		}
	}
}

func connectToTwitch(user User) *twitch.Client {
	log.Println("creating twitch client")
	client := twitch.NewClient(user.Channel.Name, fmt.Sprintf("oauth:%s", user.AccessToken))

	log.Println("configuring twitch client")
	client.OnConnect(func() {
		log.Println("Client connected")

		initmsg := WebsocketMessage{
			Key:      "channel",
			Channel:  user.Channel.Name,
			TwitchID: user.TwitchID,
		}

		broadcast <- initmsg
	})

	client.OnPingMessage(func(message twitch.PingMessage) {
		log.Printf("Ping received: %s\n", message.Message)
	})

	// Ths user listens to notices and the bot listens to commands
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
			TwitchID:  user.TwitchID,
		}

		broadcast <- initmsg
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {

		log.Println(fmt.Sprintf("New message detected: [%s] %s", message.Channel, message.Message))
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

func handleCommand(bot Bot, command string, client *twitch.Client) {
	data, err := db.Get([]byte(fmt.Sprintf("user:%s", bot.UserTwitchID)), nil)

	if err != nil {
		log.Println(err)
		return
	}

	var user User
	err = json.Unmarshal(data, &user)

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Command detected: %s\n", command)
	for _, c := range user.State.Commands {

		variables := anno.FieldFunc("variable", func(s []byte) (bool, []byte) {
			return bytes.HasPrefix(s, []byte("{")) && bytes.HasSuffix(s, []byte("}")), s
		})
		pieces := strings.Fields(command)
		inputPieces := strings.Fields(c.Input)

		if len(pieces) > 0 && pieces[0] == inputPieces[0] && len(pieces) == len(inputPieces) {
			inputVariables, err := anno.FindManyString(c.Input, variables)
			if err != nil {
				log.Println(err)
				return
			}

			outputVariables, err := anno.FindManyString(c.Output, variables)
			if err != nil {
				log.Println(err)
				return
			}

			output := c.Output

			for index, inputNote := range inputVariables {
				log.Printf("Found a %s at position %d: \"%s\"\n", inputNote.Kind, inputNote.Start, inputNote.Val)
				log.Printf("Length of pieces is %d greater than index %d\n", len(pieces), index+1)
				if len(pieces) > (index + 1) {
					if string(inputNote.Val) == "{user}" {
						log.Printf("Replacing {user} \"%s\" in \"%s\" with \"%s\"\n", string(inputNote.Val), output, strings.Trim(pieces[index+1], "@"))
						output = strings.Replace(output, string(inputNote.Val), strings.Trim(pieces[index+1], "@"), -1)
					} else {
						log.Printf("Replacing \"%s\" in \"%s\" with \"%s\"\n", string(inputNote.Val), output, pieces[index+1])
						output = strings.Replace(output, string(inputNote.Val), pieces[index+1], -1)
					}
				}
			}

			for _, outputNote := range outputVariables {
				log.Printf("Found a %s at position %d: \"%s\"\n", outputNote.Kind, outputNote.Start, outputNote.Val)

				for _, variable := range user.State.Variables {
					if len(variable.Value) > 0 && variable.Name == strings.Trim(strings.Trim(string(outputNote.Val), "{"), "}") {
						log.Printf("Replacing \"%s\" in \"%s\" with \"%s\"\n", string(outputNote.Val), output, variable.Value)
						output = strings.Replace(output, string(outputNote.Val), variable.Value, -1)
					}
				}

			}

			client.Say(user.Channel.Name, output)
			log.Printf("Bot responded to %s in channel %s: %s\n", command, user.Channel.Name, output)
		}
	}
}
