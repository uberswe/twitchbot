package botsbyuberswe

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Saving to the database
func storeStruct(s interface{}, prefix string, key string) error {
	b, err := json.Marshal(s)
	if err != nil {
		log.Printf("Error: %s", err)
		return err
	}
	return db.Put([]byte(fmt.Sprintf("%s:%s", prefix, key)), b, nil)
}

// getUserFromCookie gets the user struct from a cookie
func getUserFromCookie(cookie *http.Cookie) (User, error) {
	var cookieObj Cookie
	var user User
	log.Printf("cookie val: %s", cookie.Value)
	data, err := db.Get([]byte(fmt.Sprintf("cookie:%s", cookie.Value)), nil)
	err = json.Unmarshal(data, &cookieObj)
	if err != nil {
		return user, nil
	}
	return getUserFromTwitchID(cookieObj.TwitchID)
}

// getUserFromTwitchID gets a user struct based on the twitchID, the twitchID is the key in our database
func getUserFromTwitchID(twitchID string) (User, error) {
	var user User
	data, err := db.Get([]byte(fmt.Sprintf("user:%s", twitchID)), nil)
	if err != nil {
		log.Println(err)
		return user, err
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		log.Println(err)
		return user, err
	}
	return user, nil
}

// getTwitchIDFromChannelName gets the twitch id of a user from the channel name
func getTwitchIDFromChannelName(channelName string) (string, error) {
	var twitchID string
	data, err := db.Get([]byte(fmt.Sprintf("userChannel:%s", channelName)), nil)
	if err != nil {
		log.Println(err)
		return twitchID, err
	}
	err = json.Unmarshal(data, &twitchID)
	if err != nil {
		log.Println(err)
		return twitchID, err
	}
	return twitchID, nil
}
