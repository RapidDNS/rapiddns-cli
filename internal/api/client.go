package api

import (
	"encoding/json"
	"fmt"
	"rapiddns-cli/internal/config"
	"strconv"

	"github.com/go-resty/resty/v2"
)

const (
	BaseURL = "https://rapiddns.io/api"
)

type Client struct {
	restyClient *resty.Client
}

func NewClient() *Client {
	client := resty.New()
	client.SetBaseURL(BaseURL)
	return &Client{restyClient: client}
}

func (c *Client) getAuthHeader() map[string]string {
	apiKey := config.GetAPIKey()
	if apiKey == "" {
		return nil
	}
	return map[string]string{"X-API-KEY": apiKey}
}

// DownloadFile downloads a file from url to destPath
func (c *Client) DownloadFile(url string, destPath string) error {
	resp, err := c.restyClient.R().SetOutput(destPath).Get(url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("download failed with status: %s", resp.Status())
	}
	return nil
}

// Search performs a keyword search
func (c *Client) Search(keyword string, page, pageSize int, searchType string) (*Response, *SearchData, error) {
	req := c.restyClient.R().
		SetHeaders(c.getAuthHeader()).
		SetQueryParam("page", strconv.Itoa(page)).
		SetQueryParam("pagesize", strconv.Itoa(pageSize))

	if searchType != "" {
		req.SetQueryParam("search_type", searchType)
	}

	// Initialize Data as map first to avoid unmarshal error if structure changes or is generic
	// But here we know it's SearchData structure inside Data field for successful response
	// However, the outer structure has "data" field which can be SearchData.
	// Let's decode into specific struct.
	// Wait, the response structure in docs:
	// { "status": 200, "msg": "ok", "data": { "total": 45, "status": "ok", "data": [...] } }
	// So Response.Data is SearchData

	// To handle dynamic types in Data, we might need a custom unmarshal or just use result directly
	// Let's try to unmarshal directly into a struct that matches the expected response
	type SearchResponse struct {
		Status  interface{}     `json:"status"`
		Msg     string          `json:"msg"`
		Message json.RawMessage `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	
	var searchResp SearchResponse
	resp, err := req.SetResult(&searchResp).Get("/search/" + keyword)

	if err != nil {
		return nil, nil, err
	}

	// DEBUG: Print raw response
	// fmt.Printf("DEBUG: Raw Response: %s\n", resp.String())

	if resp.IsError() {
		if resp.StatusCode() == 401 || resp.StatusCode() == 403 {
			return nil, nil, fmt.Errorf("API key is invalid or expired. Please check your configuration or purchase a plan at https://rapiddns.io/pricing")
		}
		return nil, nil, fmt.Errorf("API error: %s", resp.Status())
	}
	
	statusOK := false
	if s, ok := searchResp.Status.(float64); ok && s == 200 {
		statusOK = true
	} else if s, ok := searchResp.Status.(string); ok && (s == "200" || s == "ok") {
		statusOK = true
	}

	if !statusOK {
		return nil, nil, fmt.Errorf("API error: %s", searchResp.Msg)
	}

	var searchData SearchData
	
	// First try to parse from 'message' field as it seems to be the current API behavior
	if len(searchResp.Message) > 0 {
		if err := json.Unmarshal(searchResp.Message, &searchData); err == nil {
			// Check if it actually has data or looks valid
			if len(searchData.Data) > 0 || len(searchData.Result) > 0 || searchData.Total > 0 || searchData.Status != "" {
				return &Response{Status: searchResp.Status, Msg: searchResp.Msg, Message: searchData, Data: searchResp.Data}, &searchData, nil
			}
		}
	}

	if err := json.Unmarshal(searchResp.Data, &searchData); err != nil {
		// Try to see if Data is just a string message
		var msg string
		if errStr := json.Unmarshal(searchResp.Data, &msg); errStr == nil {
			return nil, nil, fmt.Errorf("API returned data message: '%s'. Please check your API key or parameters.", msg)
		}
		return nil, nil, fmt.Errorf("failed to parse data: %v", err)
	}

	return &Response{Status: searchResp.Status, Msg: searchResp.Msg, Data: searchData}, &searchData, nil
}

// AdvancedQuery performs an advanced query search
func (c *Client) AdvancedQuery(query string, page, pageSize int) (*Response, *SearchData, error) {
	req := c.restyClient.R().
		SetHeaders(c.getAuthHeader()).
		SetQueryParam("page", strconv.Itoa(page)).
		SetQueryParam("pagesize", strconv.Itoa(pageSize))

	type QueryResponse struct {
		Status  interface{}     `json:"status"`
		Msg     string          `json:"msg"`
		Message json.RawMessage `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	var queryResp QueryResponse
	resp, err := req.SetResult(&queryResp).Get("/search/query/" + query)

	if err != nil {
		return nil, nil, err
	}

	// DEBUG: Print raw response
	// fmt.Printf("DEBUG: Raw Response: %s\n", resp.String())

	if resp.IsError() {
		if resp.StatusCode() == 401 || resp.StatusCode() == 403 {
			return nil, nil, fmt.Errorf("API key is invalid or expired. Please check your configuration or purchase a plan at https://rapiddns.io/pricing")
		}
		return nil, nil, fmt.Errorf("API error: %s", resp.Status())
	}

	statusOK := false
	if s, ok := queryResp.Status.(float64); ok && s == 200 {
		statusOK = true
	} else if s, ok := queryResp.Status.(string); ok && (s == "200" || s == "ok") {
		statusOK = true
	}

	if !statusOK {
		return nil, nil, fmt.Errorf("API error: %s", queryResp.Msg)
	}

	var searchData SearchData

	// First try to parse from 'message' field
	if len(queryResp.Message) > 0 {
		if err := json.Unmarshal(queryResp.Message, &searchData); err == nil {
			if len(searchData.Data) > 0 || len(searchData.Result) > 0 || searchData.Total > 0 || searchData.Status != "" {
				return &Response{Status: queryResp.Status, Msg: queryResp.Msg, Message: searchData, Data: queryResp.Data}, &searchData, nil
			}
		}
	}

	if err := json.Unmarshal(queryResp.Data, &searchData); err != nil {
		var msg string
		if errStr := json.Unmarshal(queryResp.Data, &msg); errStr == nil {
			// Special case: if msg is "ok", it might mean empty results or successful but weird response
			if msg == "ok" {
				// Return empty search data
				return &Response{Status: queryResp.Status, Msg: queryResp.Msg, Data: SearchData{Status: "ok", Data: []Record{}}}, &SearchData{Status: "ok", Data: []Record{}}, nil
			}
			return nil, nil, fmt.Errorf("API returned data message: '%s'. Please check your API key or parameters.", msg)
		}
		return nil, nil, fmt.Errorf("failed to parse data: %v", err)
	}

	return &Response{Status: queryResp.Status, Msg: queryResp.Msg, Data: searchData}, &searchData, nil
}

// ExportData initiates a data export task
func (c *Client) ExportData(queryType, queryInput string, maxResults int, compress bool) (*ExportResponseData, error) {
	req := c.restyClient.R().
		SetHeaders(c.getAuthHeader()).
		SetBody(map[string]interface{}{
			"query_type":  queryType,
			"query_input": queryInput,
			"max_results": maxResults,
			"compress":    compress,
		})

	type ExportResponse struct {
		Status  string          `json:"status"`
		Msg     string          `json:"msg"`
		Message json.RawMessage `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	var exportResp ExportResponse
	// Docs say POST /api/export-data or GET. Usually POST for actions.
	resp, err := req.SetResult(&exportResp).Post("/export-data")

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		if resp.StatusCode() == 401 || resp.StatusCode() == 403 {
			return nil, fmt.Errorf("API key is invalid or expired. Please check your configuration or purchase a plan at https://rapiddns.io/pricing")
		}
		return nil, fmt.Errorf("API error: %s", resp.Status())
	}
	
	if exportResp.Status != "ok" {
		return nil, fmt.Errorf("API error: %s", exportResp.Msg)
	}

	var exportData ExportResponseData

	// First try to parse from 'message' field
	if len(exportResp.Message) > 0 {
		if err := json.Unmarshal(exportResp.Message, &exportData); err == nil {
			if exportData.ExportID != "" {
				return &exportData, nil
			}
		}
	}

	if err := json.Unmarshal(exportResp.Data, &exportData); err != nil {
		var msg string
		if errStr := json.Unmarshal(exportResp.Data, &msg); errStr == nil {
			return nil, fmt.Errorf("API returned data message: '%s'", msg)
		}
		return nil, fmt.Errorf("failed to parse data: %v", err)
	}

	return &exportData, nil
}

// CheckExportStatus checks the status of an export task
func (c *Client) CheckExportStatus(taskID string) (*ExportStatusData, error) {
	req := c.restyClient.R().
		SetHeaders(c.getAuthHeader())

	type ExportStatusResponse struct {
		Status  string          `json:"status"`
		Message json.RawMessage `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	var statusResp ExportStatusResponse
	resp, err := req.SetResult(&statusResp).Get("/export-data/" + taskID)

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		if resp.StatusCode() == 401 || resp.StatusCode() == 403 {
			return nil, fmt.Errorf("API key is invalid or expired. Please check your configuration or purchase a plan at https://rapiddns.io/pricing")
		}
		return nil, fmt.Errorf("API error: %s", resp.Status())
	}

	if statusResp.Status != "ok" {
		return nil, fmt.Errorf("API error: status not ok")
	}

	var statusData ExportStatusData

	// First try to parse from 'message' field
	if len(statusResp.Message) > 0 {
		if err := json.Unmarshal(statusResp.Message, &statusData); err == nil {
			if statusData.ID != "" || statusData.Status != "" {
				return &statusData, nil
			}
		}
	}

	if err := json.Unmarshal(statusResp.Data, &statusData); err != nil {
		var msg string
		if errStr := json.Unmarshal(statusResp.Data, &msg); errStr == nil {
			return nil, fmt.Errorf("API returned data message: '%s'", msg)
		}
		return nil, fmt.Errorf("failed to parse data: %v", err)
	}

	return &statusData, nil
}
