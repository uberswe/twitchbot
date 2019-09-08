package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
)

func routes() {
	http.HandleFunc("/callback", callback)

	http.HandleFunc("/admin", admin)

	http.HandleFunc("/login", login)

	http.HandleFunc("/", index)

	http.HandleFunc("/auth", auth)
}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
	}
}

func callback(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("callback.html"))
	// The following is an example of the callback request
	// https://bot.uberswe.com/callback#access_token=fau80sjur5xhks8px0sq28jsy1hnak&scope=bits%3Aread+clips%3Aedit+user%3Aread%3Abroadcast+user%3Aread%3Aemail&token_type=bearer
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
	}
}

func admin(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		log.Printf("Cant find cookie :/\r\n")
		return
	}

	data, err := db.Get([]byte(cookie.Value), nil)

	var userObj User

	err = json.Unmarshal(data, &userObj)
	if err != nil {
		log.Println(err)
	}

	log.Println(fmt.Sprintf("oauth:%s", userObj.AccessToken))

	t := Template{
		AuthToken: cookie.Value,
	}

	tmpl := template.Must(template.ParseFiles("admin.html"))
	err = tmpl.Execute(w, t)

	if err != nil {
		log.Println(err)
		return
	}

}

func login(w http.ResponseWriter, r *http.Request) {
	scopes := "bits:read clips:edit user:read:broadcast chat:read chat:edit channel:moderate whispers:read whispers:edit channel_editor"
	clientID := "3en0x0g6wt7xcm0ra8z0p4fvq5bc34"
	redirectURL := "https://bot.uberswe.com/callback"
	responseType := "token"
	http.Redirect(w, r, fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s", clientID, redirectURL, responseType, scopes), 302)
	return

}

func auth(w http.ResponseWriter, r *http.Request) {
	// Parse the URL and ensure there are no errors.

	decoder := json.NewDecoder(r.Body)
	var hr HashRequest
	err := decoder.Decode(&hr)
	if err != nil {
		log.Println(err)
	}

	vals, err := url.ParseQuery(hr.Hash)
	if err != nil {
		log.Println(err)
		return
	}

	key := RandString(155)

	if vals["#access_token"] != nil {
		user := User{
			AccessToken: vals["#access_token"][0],
			Scopes:      vals["scope"],
			TokenType:   vals["token_type"][0],
		}

		b, err := json.Marshal(user)
		if err != nil {
			fmt.Printf("Error: %s", err)
			return
		}

		err = db.Put([]byte(key), b, nil)

		if err != nil {
			log.Println(err)
			return
		}

		cookie := http.Cookie{
			Name:  cookieName,
			Value: key,
		}

		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}
