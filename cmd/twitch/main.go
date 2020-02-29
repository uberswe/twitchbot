package main

import (
	"fmt"
	"github.com/markustenghamn/botsbyuberswe"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "twitch",
	Short: "Twitch is a bot made by Uberswe",
	Long: `A Twitch bot made with
                love by uberswe in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		botsbyuberswe.Init()
		botsbyuberswe.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `All software has versions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Twitch bot by Uberswe v0.1")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	Execute()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
