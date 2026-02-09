package cmd

import (
	"fmt"
	"rapiddns-cli/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure RapidDNS CLI settings",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var setKeyCmd = &cobra.Command{
	Use:   "set-key [key]",
	Short: "Set the API key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		err := config.SetAPIKey(key)
		if err != nil {
			fmt.Printf("Error setting API key: %v\n", err)
			return
		}
		fmt.Println("API key set successfully.")
	},
}

var getKeyCmd = &cobra.Command{
	Use:   "get-key",
	Short: "Get the current API key",
	Run: func(cmd *cobra.Command, args []string) {
		key := config.GetAPIKey()
		if key == "" {
			fmt.Println("API key is not set.")
		} else {
			fmt.Printf("Current API key: %s\n", key)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setKeyCmd)
	configCmd.AddCommand(getKeyCmd)
}
