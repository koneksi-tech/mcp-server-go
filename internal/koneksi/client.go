package koneksi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	DirectoryID  string
	HttpClient   *http.Client
}

type FileUploadResponse struct {
	FileID     string    `json:"file_id"`
	FileName   string    `json:"file_name"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	Status     string    `json:"status"`
}

type DirectoryInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	FileCount   int       `json:"file_count"`
	TotalSize   int64     `json:"total_size"`
}

type DirectoryResponse struct {
	DirectoryID string    `json:"directory_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type FileInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Hash        string `json:"hash"`
}

func NewClient(baseURL, clientID, clientSecret, directoryID string) *Client {
	return &Client{
		BaseURL:      baseURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		DirectoryID:  directoryID,
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) UploadFile(fileName string, fileData io.Reader, size int64, checksum string) (*FileUploadResponse, error) {
	endpoint := "/api/clients/v1/files"

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file data
	if _, err := io.Copy(part, fileData); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	url := c.BaseURL + endpoint
	if c.DirectoryID != "" {
		url += fmt.Sprintf("?directory_id=%s", c.DirectoryID)
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Data struct {
			FileID      string `json:"file_id"`
			Hash        string `json:"hash"`
			Name        string `json:"name"`
			Size        int    `json:"size"`
		} `json:"data"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	fileID := apiResp.Data.FileID
	if fileID == "" {
		fileID = apiResp.Data.Hash
	}

	return &FileUploadResponse{
		FileID:     fileID,
		FileName:   apiResp.Data.Name,
		Size:       int64(apiResp.Data.Size),
		UploadedAt: time.Now(),
		Status:     apiResp.Status,
	}, nil
}

func (c *Client) UploadFileFromBytes(fileName string, fileContent []byte, directoryID string) (*FileUploadResponse, error) {
	// Create a reader from the byte array
	reader := bytes.NewReader(fileContent)
	
	// Save the current directory ID and restore it after
	originalDirID := c.DirectoryID
	if directoryID != "" {
		c.DirectoryID = directoryID
	}
	defer func() {
		c.DirectoryID = originalDirID
	}()
	
	// Use the existing UploadFile method
	return c.UploadFile(fileName, reader, int64(len(fileContent)), "")
}

func (c *Client) DownloadFile(fileID string) (io.ReadCloser, error) {
	endpoint := fmt.Sprintf("/api/clients/v1/files/%s/download", fileID)
	
	req, err := http.NewRequest("GET", c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func (c *Client) ListDirectories() ([]DirectoryInfo, error) {
	// Default to root directory
	endpoint := "/api/clients/v1/directories/root"

	req, err := http.NewRequest("GET", c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data struct {
			Directory struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Size      int64  `json:"size"`
				CreatedAt string `json:"createdAt"`
			} `json:"directory"`
			Subdirectories []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Size      int64  `json:"size"`
				CreatedAt string `json:"createdAt"`
				UpdatedAt string `json:"updatedAt"`
			} `json:"subdirectories"`
			Files []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Size int64  `json:"size"`
			} `json:"files"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Include the root directory itself
	rootCreatedAt, _ := time.Parse(time.RFC3339, apiResp.Data.Directory.CreatedAt)
	directories := []DirectoryInfo{
		{
			ID:          apiResp.Data.Directory.ID,
			Name:        apiResp.Data.Directory.Name,
			Description: "Root directory",
			CreatedAt:   rootCreatedAt,
			FileCount:   len(apiResp.Data.Files),
			TotalSize:   apiResp.Data.Directory.Size,
		},
	}

	// Add all subdirectories
	for _, dir := range apiResp.Data.Subdirectories {
		createdAt, _ := time.Parse(time.RFC3339, dir.CreatedAt)
		directories = append(directories, DirectoryInfo{
			ID:          dir.ID,
			Name:        dir.Name,
			Description: "", // Description not provided in this endpoint
			CreatedAt:   createdAt,
			FileCount:   0,  // File count not provided for subdirectories
			TotalSize:   dir.Size,
		})
	}

	return directories, nil
}

func (c *Client) CreateDirectory(name, description string) (*DirectoryResponse, error) {
	endpoint := "/api/clients/v1/directories"

	reqBody := map[string]string{
		"name":        name,
		"description": description,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, apiResp.Data.CreatedAt)

	return &DirectoryResponse{
		DirectoryID: apiResp.Data.ID,
		Name:        apiResp.Data.Name,
		Description: apiResp.Data.Description,
		CreatedAt:   createdAt,
	}, nil
}

func (c *Client) GetDirectoryFiles(directoryID string) ([]FileInfo, error) {
	endpoint := fmt.Sprintf("/api/clients/v1/directories/%s", directoryID)

	req, err := http.NewRequest("GET", c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data struct {
			Files []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Size        int64  `json:"size"`
				ContentType string `json:"content_type"`
				Hash        string `json:"hash"`
			} `json:"files"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	files := make([]FileInfo, 0, len(apiResp.Data.Files))
	for _, file := range apiResp.Data.Files {
		files = append(files, FileInfo{
			ID:          file.ID,
			Name:        file.Name,
			Size:        file.Size,
			ContentType: file.ContentType,
			Hash:        file.Hash,
		})
	}

	return files, nil
}