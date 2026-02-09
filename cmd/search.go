package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"rapiddns-cli/internal/api"
	"rapiddns-cli/internal/config"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchPage       int
	searchPageSize   int
	searchType       string
	searchOutput     string
	searchExtract    bool
	searchExtractIPs bool
	searchOutFile    string
	searchColumn     string
	searchSilent     bool
	searchMax        int
)

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search by keyword (domain, IP, or CIDR)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.GetAPIKey() == "" {
			fmt.Fprintln(os.Stderr, "Warning: No API key configured. Results may be limited.")
			fmt.Fprintln(os.Stderr, "If you are not a PRO or MAX member, please purchase a plan at: https://rapiddns.io/pricing")
			fmt.Fprintln(os.Stderr, "Then configure your API key using: rapiddns config set-key <YOUR_API_KEY>")
			fmt.Fprintln(os.Stderr, "")
		}
		keyword := args[0]
		client := api.NewClient()
		
		var data *api.SearchData
		var err error

		// Always use pagination loop since default max is 10000
		if !searchSilent {
			fmt.Fprintf(os.Stderr, "Fetching up to %d records...\n", searchMax)
		}
			
		allRecords := []api.Record{}
		currentPage := searchPage // Start from specified page
			
		for {
			_, pageData, pageErr := client.Search(keyword, currentPage, searchPageSize, searchType)
				if pageErr != nil {
					// If it's the first page and fails, return error
					if len(allRecords) == 0 {
						err = pageErr
					} else {
						// If subsequent page fails, just stop and use what we have
						fmt.Fprintf(os.Stderr, "Warning: Stopped fetching at page %d due to error: %v\n", currentPage, pageErr)
					}
					break
				}

				// Extract records from this page
				var pageRecords []api.Record
				if len(pageData.Data) > 0 {
					pageRecords = pageData.Data
				} else if len(pageData.Result) > 0 {
					pageRecords = pageData.Result
				}

				if len(pageRecords) == 0 {
					break // No more data
				}

				allRecords = append(allRecords, pageRecords...)
				
				if !searchSilent {
					fmt.Fprintf(os.Stderr, "\rFetched %d records...", len(allRecords))
				}

				// Check limits
				if len(allRecords) >= searchMax {
					// Trim excess
					allRecords = allRecords[:searchMax]
					break
				}

				// Check if this was the last page (less than pageSize returned)
				// Note: API might return exact pageSize on last page, so this is an approximation.
				// Reliable way is checking total if available, or just keep fetching until empty.
				// But empty check is done above.
				if len(pageRecords) < searchPageSize {
					break
				}

				currentPage++
			}
			if !searchSilent {
				fmt.Fprintf(os.Stderr, "\nDone.\n")
			}

			// Construct combined data
			data = &api.SearchData{
				Data:   allRecords,
				Status: "ok",
				Total:  len(allRecords),
			}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
			return
		}

		// Ensure result directory exists if we are saving to file
		if searchExtract || searchExtractIPs || searchOutFile != "" {
			if err := os.MkdirAll("result", 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating result directory: %v\n", err)
				return
			}
		}

		// Process Subdomain Extraction
		if searchExtract {
			var subFile string
			if searchOutFile != "" {
				// Use user provided file as base name
				ext := filepath.Ext(searchOutFile)
				base := strings.TrimSuffix(searchOutFile, ext)
				subFile = base + "_subdomains.txt"
			} else {
				// Auto-generate name
				safeKeyword := sanitizeFilename(keyword)
				subFile = fmt.Sprintf("%s_subdomains.txt", safeKeyword)
			}
			
			subFile = resolvePath(subFile)
			extractSubdomains(data, subFile)
		}

		// Process IP Extraction
		if searchExtractIPs {
			var ipFile, statsFile string
			if searchOutFile != "" {
				// Use user provided file as base name
				ext := filepath.Ext(searchOutFile)
				base := strings.TrimSuffix(searchOutFile, ext)
				ipFile = base + "_ips.txt"
				statsFile = base + "_ip_stats.txt"
			} else {
				// Auto-generate name
				safeKeyword := sanitizeFilename(keyword)
				ipFile = fmt.Sprintf("%s_ips.txt", safeKeyword)
				statsFile = fmt.Sprintf("%s_ip_stats.txt", safeKeyword)
			}
			
			ipFile = resolvePath(ipFile)
			statsFile = resolvePath(statsFile)
			extractIPs(data, ipFile, statsFile)
		}

		// Process Main Output
		if searchOutFile != "" {
			finalPath := resolvePath(searchOutFile)
			saveToFile(data, finalPath, searchOutput)
		} 
		
		// Console Output
		// We print to console if:
		// 1. Not silent
		// 2. AND (No file output specified OR User explicitly wants console output?)
		// For now, consistent with standard CLI: If file is specified, silent unless asked. 
		// But user requirement implies flexible control. 
		// If searchOutFile is empty, we MUST output to console unless silent.
		if searchOutFile == "" && !searchSilent {
			printConsoleOutput(data, searchOutput, searchColumn)
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchPage, "page", 1, "Page index to fetch")
	searchCmd.Flags().IntVar(&searchPageSize, "pagesize", 100, "Page size per request")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Force search type: subdomain, same_domain, ip, ip_segment")
	searchCmd.Flags().StringVarP(&searchOutput, "output", "o", "json", "Output format: json, csv, text")
	searchCmd.Flags().BoolVar(&searchExtract, "extract-subdomains", false, "Extract and dedup subdomains to file")
	searchCmd.Flags().BoolVar(&searchExtractIPs, "extract-ips", false, "Extract and dedup IPs to file with subnet stats")
	searchCmd.Flags().StringVarP(&searchOutFile, "file", "f", "", "Output file path (default saved to 'result/' directory)")
	searchCmd.Flags().StringVar(&searchColumn, "column", "", "Output only specific column (subdomain, ip, type, value) to console")
	searchCmd.Flags().BoolVar(&searchSilent, "silent", false, "Suppress console output")
	searchCmd.Flags().IntVar(&searchMax, "max", 10000, "Max records to fetch (pagination will be handled automatically)")
}

// sanitizeFilename replaces characters that are illegal/unsafe in filenames
func sanitizeFilename(name string) string {
	// Replace directory separators and common illegal chars
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	safe := reg.ReplaceAllString(name, "_")
	// Trim spaces and dots from ends
	safe = strings.Trim(safe, " .")
	if safe == "" {
		return "search_result"
	}
	return safe
}

func resolvePath(path string) string {
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, "result"+string(os.PathSeparator)) && !strings.HasPrefix(path, "result/") {
		return filepath.Join("result", path)
	}
	return path
}

func extractSubdomains(data *api.SearchData, outFile string) {
	subdomains := make(map[string]bool)
	var records []api.Record
	if len(data.Data) > 0 {
		records = data.Data
	} else if len(data.Result) > 0 {
		records = data.Result
	}

	for _, record := range records {
		if record.Subdomain != "" {
			subdomains[record.Subdomain] = true
		}
	}

	file, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for sub := range subdomains {
		fmt.Fprintln(writer, sub)
	}
	writer.Flush()
	
	absPath, _ := filepath.Abs(outFile)
	if !searchSilent {
		fmt.Fprintf(os.Stderr, "Extracted %d unique subdomains to %s\n", len(subdomains), absPath)
	} else {
		// Even in silent mode, print the file path to stdout for piping/scripting usage
		fmt.Println(absPath)
	}
}

func extractIPs(data *api.SearchData, ipFile, statsFile string) {
	ips := make(map[string]bool)
	var records []api.Record
	if len(data.Data) > 0 {
		records = data.Data
	} else if len(data.Result) > 0 {
		records = data.Result
	}

	subnetStats := make(map[string]int)

	for _, record := range records {
		val := record.Value
		if net.ParseIP(val) != nil {
			if !ips[val] {
				ips[val] = true
				
				ip := net.ParseIP(val)
				if ip.To4() != nil {
					mask := net.CIDRMask(24, 32)
					maskedIP := ip.Mask(mask)
					subnet := maskedIP.String() + "/24"
					subnetStats[subnet]++
				} else {
					mask := net.CIDRMask(64, 128)
					maskedIP := ip.Mask(mask)
					subnet := maskedIP.String() + "/64"
					subnetStats[subnet]++
				}
			}
		}
	}

	// Write IPs to file
	file, err := os.Create(ipFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating IP file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	
	var sortedIPs []string
	for ip := range ips {
		sortedIPs = append(sortedIPs, ip)
	}
	sort.Strings(sortedIPs)

	for _, ip := range sortedIPs {
		fmt.Fprintln(writer, ip)
	}
	writer.Flush()
	
	ipAbsPath, _ := filepath.Abs(ipFile)
	if !searchSilent {
		fmt.Fprintf(os.Stderr, "Extracted %d unique IPs to %s\n", len(ips), ipAbsPath)
	} else {
		fmt.Println(ipAbsPath)
	}

	// Write Stats to file
	sFile, err := os.Create(statsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Stats file: %v\n", err)
		return
	}
	defer sFile.Close()

	sWriter := bufio.NewWriter(sFile)
	
	var sortedSubnets []string
	for subnet := range subnetStats {
		sortedSubnets = append(sortedSubnets, subnet)
	}
	sort.Strings(sortedSubnets)

	for _, subnet := range sortedSubnets {
		fmt.Fprintf(sWriter, "%s: %d IPs\n", subnet, subnetStats[subnet])
	}
	sWriter.Flush()
	
	statsAbsPath, _ := filepath.Abs(statsFile)
	if !searchSilent {
		fmt.Fprintf(os.Stderr, "Extracted IP statistics to %s\n", statsAbsPath)
	} else {
		fmt.Println(statsAbsPath)
	}

	// Still print stats to console (Stderr) for convenience
	if !searchSilent {
		fmt.Fprintln(os.Stderr, "IP Segment Statistics:")
		for _, subnet := range sortedSubnets {
			fmt.Fprintf(os.Stderr, "  %s: %d\n", subnet, subnetStats[subnet])
		}
	}
}

func saveToFile(data *api.SearchData, outFile, format string) {
	file, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	var records []api.Record
	if len(data.Data) > 0 {
		records = data.Data
	} else if len(data.Result) > 0 {
		records = data.Result
	}

	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encoder.Encode(data)
	case "csv":
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write([]string{"Subdomain", "Type", "Value", "Date", "Timestamp"})
		for _, r := range records {
			writer.Write([]string{r.Subdomain, r.Type, r.Value, r.Date, r.Timestamp})
		}
	case "text":
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		for _, r := range records {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", r.Subdomain, r.Type, r.Value, r.Date)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
	}
	
	absPath, _ := filepath.Abs(outFile)
	if !searchSilent {
		fmt.Fprintf(os.Stderr, "Saved output to %s\n", absPath)
	} else {
		fmt.Println(absPath)
	}
}

func printConsoleOutput(data *api.SearchData, format, column string) {
	var records []api.Record
	if len(data.Data) > 0 {
		records = data.Data
	} else if len(data.Result) > 0 {
		records = data.Result
	}

	// If a column is specified, we filter the data first
	if column != "" {
		column = strings.ToLower(column)
		// Collect values
		var values []string
		seen := make(map[string]bool)
		
		for _, r := range records {
			var val string
			switch column {
			case "subdomain":
				val = r.Subdomain
			case "ip":
				// Attempt to extract IP from value
				if net.ParseIP(r.Value) != nil {
					val = r.Value
				}
			case "value":
				val = r.Value
			case "type":
				val = r.Type
			}
			
			if val != "" && !seen[val] {
				seen[val] = true
				values = append(values, val)
			}
		}
		sort.Strings(values)

		// Print based on format
		if strings.ToLower(format) == "json" {
			// Print as JSON array
			output, _ := json.MarshalIndent(values, "", "  ")
			fmt.Println(string(output))
		} else {
			// Text/CSV: just print lines for single column
			for _, v := range values {
				fmt.Println(v)
			}
		}
		return
	}

	// Standard full output
	switch strings.ToLower(format) {
	case "json":
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
	case "csv":
		writer := csv.NewWriter(os.Stdout)
		defer writer.Flush()
		writer.Write([]string{"Subdomain", "Type", "Value", "Date", "Timestamp"})
		for _, r := range records {
			writer.Write([]string{r.Subdomain, r.Type, r.Value, r.Date, r.Timestamp})
		}
	case "text":
		writer := bufio.NewWriter(os.Stdout)
		defer writer.Flush()
		for _, r := range records {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", r.Subdomain, r.Type, r.Value, r.Date)
		}
	default:
		// Default to JSON if unknown
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
	}
}
