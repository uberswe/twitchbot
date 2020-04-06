package botsbyuberswe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/matryer/anno"
	"log"
	"strings"
)

// Command has in input such as `!so` which if it is detected in a Twitch channel message will trigger the Output to be sent by a Bot to the Twitch channel
type Command struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

// createCommand adds a command to a User struct
func (u *User) createCommand(command Command) bool {
	for _, c := range u.State.Commands {
		if c.Input == command.Input {
			return false
		}
	}
	u.State.Commands = append(u.State.Commands, command)
	return true
}

// removeCommand removes a command from a User struct
func (u *User) removeCommand(command Command) bool {
	for i, c := range u.State.Commands {
		if c.Input == command.Input {
			u.State.Commands = deleteCommand(u.State.Commands, i)
			return true
		}
	}
	return false
}

// handleCommand takes an incoming message, parses it for variables and then tries to match it to configured commands.
// If a matching command is found output will be sent to the corresponding channel
func handleCommand(bot Bot, message twitch.PrivateMessage, client *twitch.Client) {
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

	for _, c := range user.State.Commands {

		variables := anno.FieldFunc("variable", func(s []byte) (bool, []byte) {
			return bytes.HasPrefix(s, []byte("{")) && bytes.HasSuffix(s, []byte("}")), s
		})
		pieces := strings.Fields(message.Message)
		inputPieces := strings.Fields(c.Input)

		if len(pieces) > 0 && strings.ToLower(pieces[0]) == strings.ToLower(inputPieces[0]) && len(pieces) == len(inputPieces) {

			// Make sure the command matches
			for i, piece := range pieces {
				pieceCheck, err := anno.FindManyString(inputPieces[i], variables)
				if err != nil {
					log.Println(err)
					return
				}
				// We don't compare variables
				if len(pieceCheck) == 0 {
					if strings.ToLower(piece) != strings.ToLower(inputPieces[i]) {
						// The command does not match so we return and exit
						return
					}
				}
			}

			log.Printf("Command detected: %s\n", message.Message)

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
						log.Printf("Replacing {user} \"%s\" in \"%s\" with \"%s\"\n", string(inputNote.Val), output, strings.Trim(pieces[findIndexOfMatchingString(strings.Fields(c.Input), string(inputNote.Val))], "@"))
						output = strings.Replace(output, string(inputNote.Val), strings.Trim(pieces[findIndexOfMatchingString(strings.Fields(c.Input), string(inputNote.Val))], "@"), -1)
					} else {
						log.Printf("Replacing \"%s\" in \"%s\" with \"%s\"\n", string(inputNote.Val), output, pieces[findIndexOfMatchingString(strings.Fields(c.Input), string(inputNote.Val))])
						output = strings.Replace(output, string(inputNote.Val), pieces[findIndexOfMatchingString(strings.Fields(c.Input), string(inputNote.Val))], -1)
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

			client.Say(message.Channel, output)
			log.Printf("Bot responded to %s in channel %s: %s\n", message.Message, message.Channel, output)
		}
	}
}

func findIndexOfMatchingString(strs []string, s string) int {
	for i, st := range strs {
		if strings.ToLower(st) == strings.ToLower(s) {
			return i
		}
	}
	return 0
}
