package cmd

import (
	"encoding/json"
	"fmt"
	"rapiddns-cli/internal/api"
	"rapiddns-cli/internal/config"

	"github.com/spf13/cobra"
)

var (
	queryPage     int
	queryPageSize int
)

var queryCmd = &cobra.Command{
	Use:   "query [query]",
	Short: "Perform advanced query search",
	Long: `Perform advanced query search using syntax.
Examples:
  rapiddns query 'domain:apple AND tld:com'
  rapiddns query 'type:A AND value:"172.217.3.174"'`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.GetAPIKey() == "" {
			fmt.Println("Warning: No API key configured. Results may be limited.")
			fmt.Println("If you are not a PRO or MAX member, please purchase a plan at: https://rapiddns.io/pricing")
			fmt.Println("Then configure your API key using: rapiddns config set-key <YOUR_API_KEY>")
			fmt.Println("")
		}
		query := args[0]
		client := api.NewClient()

		_, data, err := client.AdvancedQuery(query, queryPage, queryPageSize)
		if err != nil {
			fmt.Printf("Error querying: %v\n", err)
			return
		}

		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().IntVar(&queryPage, "page", 1, "Page index to fetch")
	queryCmd.Flags().IntVar(&queryPageSize, "pagesize", 100, "Page size per request")
}
