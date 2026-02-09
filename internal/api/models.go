package api

type Response struct {
	Status  interface{} `json:"status"` // Can be int or string
	Msg     string      `json:"msg"`
	Message interface{} `json:"message"` // Actual data might be here
	Data    interface{} `json:"data"`
}

type SearchData struct {
	Total  int      `json:"total"`
	Status string   `json:"status"`
	Data   []Record `json:"data"`
	Result []Record `json:"result,omitempty"` // For AdvancedQuery
}

type Record struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Timestamp string `json:"timestamp"`
	Date      string `json:"date"`
	Subdomain string `json:"subdomain"`
}

type ExportResponseData struct {
	ExportID string `json:"export_id"`
}

type ExportStatusData struct {
	ID              string `json:"id"`
	Status          string `json:"status"` // pending, processing, completed, failed
	ProgressPercent int    `json:"progress_percent"`
	DownloadURL     string `json:"download_url,omitempty"`
}
