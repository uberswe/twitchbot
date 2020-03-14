package botsbyuberswe

import (
	"errors"
	"math/rand"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// TODO refactor these delete functions at some point, they seem overly complicated
func deleteCommand(arr []Command, index int) []Command {
	if index < 0 || index >= len(arr) {
		return arr
	}
	for i := index; i < len(arr)-1; i++ {
		arr[i] = arr[i+1]

	}
	return arr[:len(arr)-1]
}

func deleteClient(arr []Wconn, index int) []Wconn {
	if index < 0 || index >= len(arr) {
		return arr
	}
	for i := index; i < len(arr)-1; i++ {
		arr[i] = arr[i+1]

	}
	return arr[:len(arr)-1]
}

func getClientIndex(arr []Wconn, TwitchID string) (int, error) {
	for a, t := range arr {
		if t.TwitchID == TwitchID {
			return a, nil
		}
	}
	return -1, errors.New("client not found")
}
