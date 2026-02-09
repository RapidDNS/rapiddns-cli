package cmd

import (
	"fmt"
	"os"
	"rapiddns-cli/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rapiddns",
	Short: "RapidDNS CLI - A command line interface for RapidDNS API",
	Long: `RapidDNS CLI allows you to query DNS data, search domains, IPs, and export results
directly from your terminal using the RapidDNS API.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config.InitConfig)
}
