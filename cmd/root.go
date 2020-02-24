package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"os"
)

var (
	agentFile string
	port      string
	rootCmd   = &cobra.Command{
		Use:   "reddittales",
		Short: "RedditTales is a narrated reddit video generator",
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&agentFile, "agent", "", "path to graw agent file")
	rootCmd.PersistentFlags().StringVar(&port, "port", os.Getenv("PORT"), "the port for the screenshot server")
	_ = rootCmd.MarkFlagRequired("agent")
	_ = rootCmd.MarkFlagRequired("port")

	rootCmd.AddCommand(CreateCmd())
}

func QuitError(msg interface{}) {
	log.Println(msg)
	os.Exit(1)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
