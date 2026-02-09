# RapidDNS CLI

[English Documentation](README.md)

一个强大的命令行工具，用于与 [RapidDNS API](https://rapiddns.io/help/api) 进行交互。该工具允许您直接从终端执行 DNS 搜索、高级查询和大规模数据导出。

## 功能特性

*   **DNS 搜索**：支持按域名、IP 或 CIDR 搜索。
*   **高级查询**：使用强大的查询语法（例如 `domain:example.com AND type:A`）。
*   **数据导出**：自动管理导出任务：启动任务、轮询状态、下载文件以及解压。
*   **数据提取**：
    *   提取并去重 **子域名** (Subdomains) 到列表文件。
    *   提取并去重 **IP地址** (IPs) 到列表文件。
    *   生成 **IP段统计** (IP Segment Statistics)（子网计数）。
*   **灵活输出**：支持保存结果为 JSON、CSV 或 纯文本 (Text) 格式。
*   **管道支持**：专为自动化设计，数据输出到 stdout，日志/错误输出到 stderr。
*   **配置管理**：简便的 API Key 管理命令。

## 安装指南

### 源码编译

环境要求：Go 1.16+

1.  克隆仓库：
    ```bash
    git clone https://github.com/rapiddns/rapiddns-cli.git
    cd rapiddns-cli
    ```

2.  根据您的平台进行编译：

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

## 配置说明

要使用 RapidDNS 的全部功能（特别是导出和无限制搜索），您需要一个 API Key。

1.  从 [RapidDNS 用户中心](https://rapiddns.io/user/profile) 获取您的 API Key。
2.  在 CLI 中进行配置：

```bash
./rapiddns-cli.exe config set-key <YOUR_API_KEY>
```

查看当前 Key：
```bash
./rapiddns-cli.exe config get-key
```

> **注意**：如果没有 API Key，搜索结果可能会受限，且导出功能将无法使用。

## 使用指南

### 1. 基础搜索 (Search)

搜索域名、IP 或 CIDR。

```bash
./rapiddns-cli.exe search tesla.com
```

**选项参数：**
*   `--page`: 页码 (默认为 1)。
*   `--type`: 过滤类型 (`subdomain`, `same_domain`, `ip`, `ip_segment`)。
*   `-o, --output`: 输出格式 (`json`, `csv`, `text`)。默认值：`json`。
*   `-f, --file`: 将输出保存到指定文件。
*   `--column`: 仅输出指定列到控制台 (`subdomain`, `ip`, `value`, `type`)。
*   `--silent`: 静默模式，关闭控制台输出 (通常用于仅提取文件时)。
*   `--max`: 自动获取最多 N 条记录 (默认为 10000, 自动翻页)。

**示例：**

```bash
# 保存为 CSV
./rapiddns-cli.exe search tesla.com -o csv -f results.csv

# 自动获取 1000 条记录
./rapiddns-cli.exe search tesla.com --max 1000

# 仅提取子域名
./rapiddns-cli.exe search tesla.com --extract-subdomains

# 仅提取 IP 和统计信息
./rapiddns-cli.exe search tesla.com --extract-ips

# 同时提取两者
./rapiddns-cli.exe search tesla.com --extract-subdomains --extract-ips
```

### 2. 管道与控制台输出

专为黑客习惯和自动化管线设计。标准输出 (stdout) 仅包含干净的数据，而状态/错误信息输出到 stderr。

**仅输出子域名 (纯文本列表):**
```bash
./rapiddns-cli.exe search tesla.com --column subdomain -o text
```
*输出:*
```text
api.tesla.com
www.tesla.com
...
```

**仅输出 IP (JSON 数组):**
```bash
./rapiddns-cli.exe search tesla.com --column ip -o json
```
*输出:*
```json
[
  "1.2.3.4",
  "5.6.7.8"
]
```

**静默模式 (仅提取文件):**
```bash
./rapiddns-cli.exe search tesla.com --extract-subdomains --silent
```

### 3. 高级查询 (Advanced Query)

使用 RapidDNS 语法执行复杂查询。

```bash
./rapiddns-cli.exe query "domain:apple.com AND type:A"
```

### 4. 数据导出 (Export) - 推荐用于大数据量

Export 命令处理整个工作流：请求导出、等待完成、下载文件并进行处理。

```bash
./rapiddns-cli.exe export start tesla.com
```

**选项参数：**
*   `--max`: 导出的最大记录数 (0 表示全部, 默认为 0)。
*   `--compress`: 将结果压缩为 ZIP (默认 `true`)。
*   `--extract-subdomains`: 从下载的 CSV 中提取子域名。
*   `--extract-ips`: 从下载的 CSV 中提取 IP 并生成子网统计。

**完整工作流示例：**

```bash
./rapiddns-cli.exe export start tesla.com --max 10000 --extract-subdomains --extract-ips
```

该命令将执行以下操作：
1.  触发 `tesla.com` 的导出任务。
2.  等待服务器处理完成。
3.  下载 ZIP 文件到 `result/` 目录。
4.  解压文件内容。
5.  生成 `tesla.com_subdomains.txt` (子域名列表)。
6.  生成 `tesla.com_ips.txt` (唯一 IP 列表)。
7.  生成 `tesla.com_ip_stats.txt` (IP 子网统计)。

## 输出目录结构

默认情况下，所有结果都保存在 `result/` 目录下。

*   `{keyword}_subdomains.txt`: 清洗后的唯一子域名列表。
*   `{keyword}_ips.txt`: 清洗后的唯一 IP 地址列表。
*   `{keyword}_ip_stats.txt`: 每个子网的 IP 计数统计 (IPv4 /24, IPv6 /64)。
*   `rapiddns_export_{keyword}_{date}.csv`: 原始导出数据。

## 帮助信息

运行带有 `--help` 的任何命令以查看更多详细信息。

```bash
./rapiddns-cli.exe --help
./rapiddns-cli.exe search --help
./rapiddns-cli.exe export --help
```
