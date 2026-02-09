package cmd

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"rapiddns-cli/internal/api"
	"rapiddns-cli/internal/config"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	exportType       string
	exportMaxResults int
	exportCompress   bool
	exportExtract    bool
	exportExtractIPs bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data operations",
}

var exportStartCmd = &cobra.Command{
	Use:   "start [query_input]",
	Short: "Start a data export task, wait for completion, and download result",
	Long: `Starts a data export task, polls the status until completion, and downloads the result to the local 'result' directory.
Default compression is enabled (ZIP). If compressed, it will also extract the file.
Can optionally extract subdomains and IPs from the downloaded result (CSV only).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.GetAPIKey() == "" {
			fmt.Println("Error: API key is required for export operations.")
			fmt.Println("If you are not a PRO or MAX member, please purchase a plan at: https://rapiddns.io/pricing")
			fmt.Println("Then configure your API key using: rapiddns config set-key <YOUR_API_KEY>")
			return
		}
		queryInput := args[0]
		client := api.NewClient()

		fmt.Printf("Starting export task for '%s' (Type: %s, Max: %d)...\n", queryInput, exportType, exportMaxResults)

		// 1. Start Export Task
		data, err := client.ExportData(exportType, queryInput, exportMaxResults, exportCompress)
		if err != nil {
			fmt.Printf("Error starting export: %v\n", err)
			return
		}

		taskID := data.ExportID
		fmt.Printf("Export task started. Task ID: %s\n", taskID)

		// 2. Poll Status
		fmt.Println("Waiting for task completion...")
		var downloadURL string
		for {
			statusData, err := client.CheckExportStatus(taskID)
			if err != nil {
				fmt.Printf("Error checking status: %v. Retrying in 5 seconds...\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			fmt.Printf("Status: %s (Progress: %d%%)\n", statusData.Status, statusData.ProgressPercent)

			if statusData.Status == "completed" {
				downloadURL = statusData.DownloadURL
				break
			} else if statusData.Status == "failed" {
				fmt.Println("Export task failed.")
				return
			}

			time.Sleep(2 * time.Second)
		}

		if downloadURL == "" {
			fmt.Println("Task completed but no download URL found.")
			return
		}

		// 3. Download File
		resultDir := "result"
		if err := os.MkdirAll(resultDir, 0755); err != nil {
			fmt.Printf("Error creating result directory: %v\n", err)
			return
		}

		// Extract filename from URL or generate one
		fileName := filepath.Base(downloadURL)
		if fileName == "" || fileName == "." || fileName == "/" {
			// Fallback filename if URL doesn't have one
			timestamp := time.Now().Format("20060102_150405")
			ext := ".csv"
			if exportCompress {
				ext = ".zip"
			}
			fileName = fmt.Sprintf("rapiddns_export_%s_%s%s", queryInput, timestamp, ext)
			// Clean filename
			fileName = strings.ReplaceAll(fileName, ":", "_")
			fileName = strings.ReplaceAll(fileName, "/", "_")
			fileName = strings.ReplaceAll(fileName, "\\", "_")
		}

		destPath := filepath.Join(resultDir, fileName)
		fmt.Printf("Downloading result to %s...\n", destPath)

		if err := client.DownloadFile(downloadURL, destPath); err != nil {
			fmt.Printf("Error downloading file: %v\n", err)
			return
		}

		fmt.Println("Download completed successfully!")

		var extractedCSVPath string
		// 4. Decompress if needed
		if exportCompress && strings.HasSuffix(strings.ToLower(fileName), ".zip") {
			fmt.Printf("Decompressing %s...\n", fileName)
			unzippedFiles, err := unzip(destPath, resultDir)
			if err != nil {
				fmt.Printf("Error decompressing file: %v\n", err)
			} else {
				fmt.Println("Decompressed files:")
				for _, f := range unzippedFiles {
					fmt.Printf("- %s\n", f)
					// Try to find the CSV file
					if strings.HasSuffix(strings.ToLower(f), ".csv") {
						extractedCSVPath = f
					}
				}
			}
		} else if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
			extractedCSVPath = destPath
		}

		// 5. Extract Subdomains and IPs from CSV
		if (exportExtract || exportExtractIPs) && extractedCSVPath != "" {
			fmt.Println("Processing CSV for extraction...")
			records, err := parseCSV(extractedCSVPath)
			if err != nil {
				fmt.Printf("Error parsing CSV for extraction: %v\n", err)
			} else {
				// Convert to api.SearchData format for reuse of extraction logic
				// Note: parseCSV returns []api.Record
				searchData := &api.SearchData{
					Data: records,
				}
				
				safeKeyword := sanitizeFilename(queryInput)

				if exportExtract {
					subFile := filepath.Join(resultDir, fmt.Sprintf("%s_subdomains.txt", safeKeyword))
					extractSubdomains(searchData, subFile)
				}
				
				if exportExtractIPs {
					ipFile := filepath.Join(resultDir, fmt.Sprintf("%s_ips.txt", safeKeyword))
					statsFile := filepath.Join(resultDir, fmt.Sprintf("%s_ip_stats.txt", safeKeyword))
					extractIPs(searchData, ipFile, statsFile)
				}
			}
		} else if (exportExtract || exportExtractIPs) && extractedCSVPath == "" {
			fmt.Println("Warning: Could not find a CSV file to extract data from.")
		}

		fmt.Println("Export task finished.")
	},
}

// unzip extracts a zip archive to destDir and returns list of extracted file paths
func unzip(src string, destDir string) ([]string, error) {
	var filePaths []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		// Construct the destination path
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path: %s", fpath)
		}

		filePaths = append(filePaths, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return nil, err
		}
	}
	return filePaths, nil
}

// parseCSV reads the exported CSV file and returns records
func parseCSV(filePath string) ([]api.Record, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow variable number of fields if needed, but RapidDNS CSV usually consistent
	// reader.FieldsPerRecord = -1 

	rawRecords, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var records []api.Record
	if len(rawRecords) < 2 {
		return records, nil // Empty or header only
	}

	// Assume first row is header. RapidDNS Export CSV usually: Subdomain, Type, Value, Date
	// Header: "Subdomain", "Type", "Value", "Date"
	
	header := rawRecords[0]
	subdomainIdx := -1
	typeIdx := -1
	valueIdx := -1
	dateIdx := -1

	// If header parsing failed (subdomain index is -1), try default mapping for RapidDNS export
	// Default columns: Subdomain, Value, Type, Date
	if subdomainIdx == -1 {
		subdomainIdx = 0
		valueIdx = 1
		typeIdx = 2
		dateIdx = 3
		
		// If header row looks like data (not "subdomain" etc), include it as data
		firstRowIsData := true
		for _, h := range header {
			hLower := strings.ToLower(h)
			if hLower == "subdomain" || hLower == "type" || hLower == "value" || hLower == "date" {
				firstRowIsData = false
				break
			}
		}
		
		if firstRowIsData {
			// Process the first row as data
			rec := api.Record{}
			if len(header) > subdomainIdx { rec.Subdomain = header[subdomainIdx] }
			if len(header) > typeIdx { rec.Type = header[typeIdx] }
			if len(header) > valueIdx { rec.Value = header[valueIdx] }
			if len(header) > dateIdx { rec.Date = header[dateIdx] }
			records = append(records, rec)
		}
	}

	for _, row := range rawRecords[1:] {
		rec := api.Record{}
		if subdomainIdx != -1 && len(row) > subdomainIdx {
			rec.Subdomain = row[subdomainIdx]
		}
		if typeIdx != -1 && len(row) > typeIdx {
			rec.Type = row[typeIdx]
		}
		if valueIdx != -1 && len(row) > valueIdx {
			rec.Value = row[valueIdx]
		}
		if dateIdx != -1 && len(row) > dateIdx {
			rec.Date = row[dateIdx]
		}
		records = append(records, rec)
	}

	return records, nil
}

var exportStatusCmd = &cobra.Command{
	Use:   "status [task_id]",
	Short: "Check the status of an export task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.GetAPIKey() == "" {
			fmt.Println("Error: API key is required to check export status.")
			fmt.Println("If you are not a PRO or MAX member, please purchase a plan at: https://rapiddns.io/pricing")
			fmt.Println("Then configure your API key using: rapiddns config set-key <YOUR_API_KEY>")
			return
		}
		taskID := args[0]
		client := api.NewClient()

		data, err := client.CheckExportStatus(taskID)
		if err != nil {
			fmt.Printf("Error checking export status: %v\n", err)
			return
		}

		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportStartCmd)
	exportCmd.AddCommand(exportStatusCmd)

	exportStartCmd.Flags().StringVar(&exportType, "type", "subdomain", "Search type: subdomain, sameip, ip_segment, advanced")
	exportStartCmd.Flags().IntVar(&exportMaxResults, "max", 0, "Max records to export (0 means all)")
	exportStartCmd.Flags().BoolVar(&exportCompress, "compress", true, "Compress result as ZIP")
	exportStartCmd.Flags().BoolVar(&exportExtract, "extract-subdomains", false, "Extract and dedup subdomains from exported result")
	exportStartCmd.Flags().BoolVar(&exportExtractIPs, "extract-ips", false, "Extract and dedup IPs from exported result")
}
