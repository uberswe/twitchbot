package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gempir/go-twitch-irc"
	"github.com/matryer/anno"
	"log"
)

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

		jsonString, err := json.Marshal(message)

		if err != nil {
			log.Println(err)
			return
		}

		initmsg := Message{
			Key:   "message",
			Value: string(jsonString),
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

func handleCommand(command string, message twitch.PrivateMessage) {
	fmt.Printf("Command detected: %s\n", command)
}
