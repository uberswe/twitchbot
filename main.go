package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		scopes := "bits:read channel:read:subscriptions	clips:edit user:read:broadcast user:read:email"
		clientID := "***REMOVED***"
		redirectURL := "https://bot.uberswe.com/callback"
		responseType := "token"
		http.Redirect(w, r, fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s", clientID, redirectURL, responseType, scopes), 302)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		log.Println(body)

		err = r.ParseForm()

		if err != nil {
			log.Printf("Error parsing form: %v", err)
			http.Error(w, "can't parsing form", http.StatusBadRequest)
			return
		}

		log.Println(r.PostForm)

		_, err = w.Write([]byte("OK"))

		if err != nil {
			log.Printf("Error writing status: %v", err)
			http.Error(w, "can't write status", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":8010", nil)

	if err != nil {
		panic(err)
	}
}
