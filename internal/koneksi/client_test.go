package koneksi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "test-id", "test-secret", "test-dir")
	
	if client.BaseURL != "https://api.example.com" {
		t.Errorf("Expected BaseURL to be https://api.example.com, got %s", client.BaseURL)
	}
	if client.ClientID != "test-id" {
		t.Errorf("Expected ClientID to be test-id, got %s", client.ClientID)
	}
	if client.ClientSecret != "test-secret" {
		t.Errorf("Expected ClientSecret to be test-secret, got %s", client.ClientSecret)
	}
	if client.DirectoryID != "test-dir" {
		t.Errorf("Expected DirectoryID to be test-dir, got %s", client.DirectoryID)
	}
	if client.HttpClient == nil {
		t.Error("Expected HttpClient to be initialized")
	}
}

func TestClient_UploadFile(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		expectedError  bool
		expectedFileID string
	}{
		{
			name:       "successful upload",
			statusCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"file_id": "test-file-id",
					"hash":    "test-hash",
					"name":    "test.txt",
					"size":    100,
				},
			},
			expectedError:  false,
			expectedFileID: "test-file-id",
		},
		{
			name:          "upload failure",
			statusCode:    http.StatusBadRequest,
			responseBody:  "Bad Request",
			expectedError: true,
		},
		{
			name:       "file_id fallback to hash",
			statusCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"file_id": "",
					"hash":    "test-hash",
					"name":    "test.txt",
					"size":    100,
				},
			},
			expectedError:  false,
			expectedFileID: "test-hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				if r.Header.Get("Client-ID") != "test-id" {
					t.Errorf("Expected Client-ID header to be test-id, got %s", r.Header.Get("Client-ID"))
				}
				if r.Header.Get("Client-Secret") != "test-secret" {
					t.Errorf("Expected Client-Secret header to be test-secret, got %s", r.Header.Get("Client-Secret"))
				}
				
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.responseBody)
				} else {
					w.Write([]byte(tt.responseBody.(string)))
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-id", "test-secret", "")
			fileData := strings.NewReader("test file content")
			
			resp, err := client.UploadFile("test.txt", fileData, 17, "checksum")
			
			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectedError && resp.FileID != tt.expectedFileID {
				t.Errorf("Expected FileID %s, got %s", tt.expectedFileID, resp.FileID)
			}
		})
	}
}

func TestClient_DownloadFile(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError bool
	}{
		{
			name:          "successful download",
			statusCode:    http.StatusOK,
			responseBody:  "file content",
			expectedError: false,
		},
		{
			name:          "download not found",
			statusCode:    http.StatusNotFound,
			responseBody:  "File not found",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET method, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/files/test-file-id/download") {
					t.Errorf("Expected path to contain /files/test-file-id/download, got %s", r.URL.Path)
				}
				
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-id", "test-secret", "")
			
			reader, err := client.DownloadFile("test-file-id")
			
			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectedError {
				defer reader.Close()
				content, _ := io.ReadAll(reader)
				if string(content) != tt.responseBody {
					t.Errorf("Expected content %s, got %s", tt.responseBody, string(content))
				}
			}
		})
	}
}

func TestClient_ListDirectories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		
		// Check that it's requesting the root directory
		if !strings.HasSuffix(r.URL.Path, "/directories/root") {
			t.Errorf("Expected path to end with /directories/root, got %s", r.URL.Path)
		}
		
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":          "dir1",
					"name":        "Directory 1",
					"description": "Test directory 1",
					"created_at":  "2023-01-01T00:00:00Z",
					"file_count":  5,
					"total_size":  1024,
				},
				{
					"id":          "dir2",
					"name":        "Directory 2",
					"description": "Test directory 2",
					"created_at":  "2023-01-02T00:00:00Z",
					"file_count":  10,
					"total_size":  2048,
				},
			},
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-id", "test-secret", "")
	
	directories, err := client.ListDirectories()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(directories) != 2 {
		t.Errorf("Expected 2 directories, got %d", len(directories))
	}
	
	if directories[0].ID != "dir1" {
		t.Errorf("Expected first directory ID to be dir1, got %s", directories[0].ID)
	}
	if directories[1].FileCount != 10 {
		t.Errorf("Expected second directory file count to be 10, got %d", directories[1].FileCount)
	}
}

func TestClient_CreateDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		
		if reqBody["name"] != "New Directory" {
			t.Errorf("Expected name to be 'New Directory', got %s", reqBody["name"])
		}
		
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":          "new-dir-id",
				"name":        "New Directory",
				"description": "A new test directory",
				"created_at":  "2023-01-01T00:00:00Z",
			},
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-id", "test-secret", "")
	
	dir, err := client.CreateDirectory("New Directory", "A new test directory")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if dir.DirectoryID != "new-dir-id" {
		t.Errorf("Expected directory ID to be new-dir-id, got %s", dir.DirectoryID)
	}
	if dir.Name != "New Directory" {
		t.Errorf("Expected directory name to be 'New Directory', got %s", dir.Name)
	}
}

func TestClient_GetDirectoryFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/directories/test-dir-id") {
			t.Errorf("Expected path to contain /directories/test-dir-id, got %s", r.URL.Path)
		}
		
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"id":           "file1",
						"name":         "file1.txt",
						"size":         100,
						"content_type": "text/plain",
						"hash":         "hash1",
					},
					{
						"id":           "file2",
						"name":         "file2.txt",
						"size":         200,
						"content_type": "text/plain",
						"hash":         "hash2",
					},
				},
			},
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-id", "test-secret", "")
	
	files, err := client.GetDirectoryFiles("test-dir-id")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
	
	if files[0].ID != "file1" {
		t.Errorf("Expected first file ID to be file1, got %s", files[0].ID)
	}
	if files[1].Size != 200 {
		t.Errorf("Expected second file size to be 200, got %d", files[1].Size)
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	client := NewClient("http://invalid-url", "test-id", "test-secret", "")
	client.HttpClient.Timeout = 1 * time.Second
	
	t.Run("network error", func(t *testing.T) {
		_, err := client.ListDirectories()
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})
	
	t.Run("invalid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()
		
		client := NewClient(server.URL, "test-id", "test-secret", "")
		_, err := client.ListDirectories()
		if err == nil {
			t.Error("Expected error for invalid JSON response")
		}
	})
}