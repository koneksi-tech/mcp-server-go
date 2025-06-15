package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koneksi/mcp-server/internal/koneksi"
)

func TestNewServer(t *testing.T) {
	client := &koneksi.Client{}
	server := NewServer("test-server", "1.0.0", client)
	
	if server.name != "test-server" {
		t.Errorf("Expected name to be test-server, got %s", server.name)
	}
	if server.version != "1.0.0" {
		t.Errorf("Expected version to be 1.0.0, got %s", server.version)
	}
	if server.client != client {
		t.Error("Expected client to be set correctly")
	}
}

func TestServer_HandleInitialize(t *testing.T) {
	server := NewServer("test-server", "1.0.0", nil)
	
	request := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	if respMap["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc to be 2.0, got %v", respMap["jsonrpc"])
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo to be a map")
	}
	
	if serverInfo["name"] != "test-server" {
		t.Errorf("Expected server name to be test-server, got %v", serverInfo["name"])
	}
	if serverInfo["version"] != "1.0.0" {
		t.Errorf("Expected server version to be 1.0.0, got %v", serverInfo["version"])
	}
}

func TestServer_HandleToolsList(t *testing.T) {
	server := NewServer("test-server", "1.0.0", nil)
	
	request := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tools to be an array")
	}
	
	expectedTools := []string{
		"upload_file", "download_file", "list_directories", 
		"create_directory", "search_files", "backup_file",
	}
	
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
	
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if !ok {
			t.Error("Tool name should be a string")
			continue
		}
		toolNames[name] = true
	}
	
	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found", expectedTool)
		}
	}
}

func TestServer_HandleToolCall_ListDirectories(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":          "dir1",
					"name":        "Test Directory",
					"description": "A test directory",
					"created_at":  "2023-01-01T00:00:00Z",
					"file_count":  5,
					"total_size":  1024,
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()
	
	client := koneksi.NewClient(mockServer.URL, "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	request := `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{
			"name":"list_directories",
			"arguments":"{}"
		}
	}`
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be an array")
	}
	
	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}
	
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Expected text to be a string")
	}
	
	if !strings.Contains(text, "Test Directory") {
		t.Error("Expected response to contain 'Test Directory'")
	}
}

func TestServer_HandleToolCall_CreateDirectory(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer mockServer.Close()
	
	client := koneksi.NewClient(mockServer.URL, "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	request := `{
		"jsonrpc":"2.0",
		"id":4,
		"method":"tools/call",
		"params":{
			"name":"create_directory",
			"arguments":"{\"name\":\"New Directory\",\"description\":\"A new test directory\"}"
		}
	}`
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be an array")
	}
	
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Expected text to be a string")
	}
	
	if !strings.Contains(text, "Directory created!") {
		t.Error("Expected response to contain 'Directory created!'")
	}
	if !strings.Contains(text, "new-dir-id") {
		t.Error("Expected response to contain directory ID")
	}
}

func TestServer_HandleToolCall_UploadFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"file_id": "test-file-id",
				"hash":    "test-hash",
				"name":    "test.txt",
				"size":    12,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()
	
	client := koneksi.NewClient(mockServer.URL, "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	request := fmt.Sprintf(`{
		"jsonrpc":"2.0",
		"id":5,
		"method":"tools/call",
		"params":{
			"name":"upload_file",
			"arguments":"{\"filePath\":\"%s\"}"
		}
	}`, strings.ReplaceAll(testFile, "\\", "\\\\"))
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be an array")
	}
	
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Expected text to be a string")
	}
	
	if !strings.Contains(text, "File uploaded successfully!") {
		t.Error("Expected response to contain 'File uploaded successfully!'")
	}
	if !strings.Contains(text, "test-file-id") {
		t.Error("Expected response to contain file ID")
	}
}

func TestServer_HandleToolCall_DownloadFile(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("downloaded content"))
	}))
	defer mockServer.Close()
	
	client := koneksi.NewClient(mockServer.URL, "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "downloaded.txt")
	
	request := fmt.Sprintf(`{
		"jsonrpc":"2.0",
		"id":6,
		"method":"tools/call",
		"params":{
			"name":"download_file",
			"arguments":"{\"fileId\":\"test-file-id\",\"outputPath\":\"%s\"}"
		}
	}`, strings.ReplaceAll(outputPath, "\\", "\\\\"))
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be an array")
	}
	
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Expected text to be a string")
	}
	
	if !strings.Contains(text, "File downloaded successfully!") {
		t.Error("Expected response to contain 'File downloaded successfully!'")
	}
	
	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected downloaded file to exist")
	}
	
	// Verify content
	content2, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content2) != "downloaded content" {
		t.Errorf("Expected content to be 'downloaded content', got %s", string(content2))
	}
}

func TestServer_HandleToolCall_SearchFiles(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer mockServer.Close()
	
	client := koneksi.NewClient(mockServer.URL, "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	request := `{
		"jsonrpc":"2.0",
		"id":7,
		"method":"tools/call",
		"params":{
			"name":"search_files",
			"arguments":"{\"directoryId\":\"test-dir-id\"}"
		}
	}`
	
	response, err := server.HandleRequest(request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatal("Expected response to be a map")
	}
	
	result, ok := respMap["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be an array")
	}
	
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Expected text to be a string")
	}
	
	if !strings.Contains(text, "Files in directory test-dir-id:") {
		t.Error("Expected response to contain directory ID")
	}
	if !strings.Contains(text, "file1.txt") {
		t.Error("Expected response to contain file1.txt")
	}
	if !strings.Contains(text, "file2.txt") {
		t.Error("Expected response to contain file2.txt")
	}
}

func TestServer_ErrorHandling(t *testing.T) {
	server := NewServer("test-server", "1.0.0", nil)
	
	tests := []struct {
		name    string
		request string
		errMsg  string
	}{
		{
			name:    "unknown method",
			request: `{"jsonrpc":"2.0","id":1,"method":"unknown","params":{}}`,
			errMsg:  "unknown method",
		},
		{
			name:    "invalid JSON",
			request: `{invalid json}`,
			errMsg:  "unknown method",
		},
		{
			name:    "unknown tool",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"unknown_tool","arguments":"{}"}}`,
			errMsg:  "unknown tool",
		},
		{
			name:    "invalid arguments",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"upload_file","arguments":"invalid json"}}`,
			errMsg:  "failed to parse arguments",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.HandleRequest(tt.request)
			if err == nil {
				t.Error("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got %v", tt.errMsg, err)
			}
		})
	}
}

func TestServer_HandleToolCall_RequiredParameters(t *testing.T) {
	client := koneksi.NewClient("http://localhost", "test-id", "test-secret", "")
	server := NewServer("test-server", "1.0.0", client)
	
	tests := []struct {
		name      string
		toolName  string
		arguments string
		errMsg    string
	}{
		{
			name:      "upload_file missing filePath",
			toolName:  "upload_file",
			arguments: "{}",
			errMsg:    "filePath is required",
		},
		{
			name:      "download_file missing fileId",
			toolName:  "download_file",
			arguments: "{\"outputPath\":\"/tmp/test.txt\"}",
			errMsg:    "fileId is required",
		},
		{
			name:      "download_file missing outputPath",
			toolName:  "download_file",
			arguments: "{\"fileId\":\"test-id\"}",
			errMsg:    "outputPath is required",
		},
		{
			name:      "create_directory missing name",
			toolName:  "create_directory",
			arguments: "{}",
			errMsg:    "name is required",
		},
		{
			name:      "search_files missing directoryId",
			toolName:  "search_files",
			arguments: "{}",
			errMsg:    "directoryId is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := fmt.Sprintf(`{
				"jsonrpc":"2.0",
				"id":1,
				"method":"tools/call",
				"params":{
					"name":"%s",
					"arguments":"%s"
				}
			}`, tt.toolName, strings.ReplaceAll(tt.arguments, "\"", "\\\""))
			
			_, err := server.HandleRequest(request)
			if err == nil {
				t.Error("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got %v", tt.errMsg, err)
			}
		})
	}
}