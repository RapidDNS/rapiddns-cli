# RapidDNS CLI

[中文文档](README_zh.md)

A powerful command-line interface for interacting with the [RapidDNS API](https://rapiddns.io/help/api). This tool allows you to perform DNS searches, advanced queries, and large-scale data exports directly from your terminal.

## Features

*   **DNS Search**: Search by domain, IP, or CIDR.
*   **Advanced Query**: Use powerful query syntax (e.g., `domain:example.com AND type:A`).
*   **Data Export**: Automatically manage export tasks: start, poll status, download, and decompress.
*   **Data Extraction**:
    *   Extract and deduplicate **Subdomains** to a list.
    *   Extract and deduplicate **IPs** to a list.
    *   Generate **IP Segment Statistics** (subnet counts).
*   **Flexible Output**: Save results in JSON, CSV, or Text formats.
*   **Pipeline Support**: Clean stdout/stderr separation for chaining with other tools.
*   **Configuration**: Easy API key management.

## Installation

### Build from Source

Requirements: Go 1.16+

1.  Clone the repository:
    ```bash
    git clone https://github.com/rapiddns/rapiddns-cli.git
    cd rapiddns-cli
    ```

2.  Build for your platform:

    **Windows:**
    ```bash
    go build -o rapiddns-cli.exe
    ```

    **Linux:**
    ```bash
    GOOS=linux GOARCH=amd64 go build -o rapiddns-cli
    ```

    **macOS (Intel):**
    ```bash
    GOOS=darwin GOARCH=amd64 go build -o rapiddns-cli
    ```

    **macOS (Apple Silicon/M1/M2):**
    ```bash
    GOOS=darwin GOARCH=arm64 go build -o rapiddns-cli
    ```

## Configuration

To use the full features of RapidDNS (especially Export and unlimited Search), you need an API Key.

1.  Get your API Key from [RapidDNS Profile](https://rapiddns.io/user/profile).
2.  Configure it in the CLI:

```bash
./rapiddns-cli.exe config set-key <YOUR_API_KEY>
```

Check current key:
```bash
./rapiddns-cli.exe config get-key
```

> **Note**: Without an API Key, search results may be limited, and export functionality will be disabled.

## Usage

### 1. Basic Search

Search for a domain, IP, or CIDR.

```bash
./rapiddns-cli.exe search tesla.com
```

**Options:**
*   `--page`: Page number (default 1).
*   `--type`: Filter by type (`subdomain`, `same_domain`, `ip`, `ip_segment`).
*   `-o, --output`: Output format (`json`, `csv`, `text`). Default: `json`.
*   `-f, --file`: Save output to a specific file.
*   `--column`: Output only a specific column to console (`subdomain`, `ip`, `value`, `type`).
*   `--silent`: Suppress console output (useful when saving to file or extracting).
*   `--max`: Automatically fetch up to N records (pagination handled automatically).

**Examples:**

```bash
# Save as CSV
./rapiddns-cli.exe search tesla.com -o csv -f results.csv

# Fetch up to 1000 records automatically
./rapiddns-cli.exe search tesla.com --max 1000

# Extract Subdomains only
./rapiddns-cli.exe search tesla.com --extract-subdomains

# Extract IPs and Stats only
./rapiddns-cli.exe search tesla.com --extract-ips

# Extract BOTH
./rapiddns-cli.exe search tesla.com --extract-subdomains --extract-ips
```

### 2. Pipeline & Console Output

Designed for hackers and automation. Standard output (stdout) is clean data, while status/errors go to stderr.

**Output only subdomains (text list):**
```bash
./rapiddns-cli.exe search tesla.com --column subdomain -o text
```
*Output:*
```text
api.tesla.com
www.tesla.com
...
```

**Output only IPs (JSON array):**
```bash
./rapiddns-cli.exe search tesla.com --column ip -o json
```
*Output:*
```json
[
  "1.2.3.4",
  "5.6.7.8"
]
```

**Silent mode (only extract to files):**
```bash
./rapiddns-cli.exe search tesla.com --extract-subdomains --silent
```

### 3. Advanced Query

Perform complex queries using RapidDNS syntax.

```bash
./rapiddns-cli.exe query "domain:apple.com AND type:A"
```

### 4. Data Export (Recommended for Large Data)

The export command handles the entire workflow: requesting the export, waiting for completion, downloading the file, and processing it.

```bash
./rapiddns-cli.exe export start tesla.com
```

**Options:**
*   `--type`: Search type (`subdomain`, `sameip`, `ip_segment`, `advanced`). Use `advanced` for query syntax.
*   `--max`: Maximum records to export (0 means all, default 0).
*   `--compress`: Compress result as ZIP (default `true`).
*   `--extract-subdomains`: Extract subdomains from the downloaded CSV.
*   `--extract-ips`: Extract IPs and generate subnet statistics from the downloaded CSV.

**Full Workflow Example:**

```bash
./rapiddns-cli.exe export start tesla.com --max 10000 --extract-subdomains --extract-ips
```

**Advanced Export Example:**

To use advanced query syntax for export, set `--type advanced`:

```bash
./rapiddns-cli.exe export start "domain:apple AND type:A" --type advanced
```

This command will:
1.  Trigger an export task for `tesla.com`.
2.  Wait for the server to process.
3.  Download the ZIP file to the `result/` directory.
4.  Unzip the content.
5.  Generate `tesla.com_subdomains.txt` (list of subdomains).
6.  Generate `tesla.com_ips.txt` (list of unique IPs).
7.  Generate `tesla.com_ip_stats.txt` (IP subnet statistics).

## Output Structure

All results are saved by default in the `result/` directory.

*   `{keyword}_subdomains.txt`: Clean list of unique subdomains.
*   `{keyword}_ips.txt`: Clean list of unique IP addresses.
*   `{keyword}_ip_stats.txt`: Statistics of IP counts per subnet (IPv4 /24, IPv6 /64).
*   `rapiddns_export_{keyword}_{date}.csv`: Raw exported data.

## Help

Run any command with `--help` to see more details.

```bash
./rapiddns-cli.exe --help
./rapiddns-cli.exe search --help
./rapiddns-cli.exe export --help
```
