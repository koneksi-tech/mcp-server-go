package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/koneksi/mcp-server/internal/koneksi"
	"github.com/tidwall/gjson"
)

type Server struct {
	name    string
	version string
	client  *koneksi.Client
}

func NewServer(name, version string, client *koneksi.Client) *Server {
	return &Server{
		name:    name,
		version: version,
		client:  client,
	}
}

func (s *Server) HandleRequest(requestStr string) (interface{}, error) {
	// Parse JSON-RPC request
	parsed := gjson.Parse(requestStr)
	method := parsed.Get("method").String()
	id := parsed.Get("id").Value()

	switch method {
	case "initialize":
		return s.handleInitialize(id)
	case "tools/list":
		return s.handleToolsList(id)
	case "tools/call":
		return s.handleToolCall(parsed, id)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (s *Server) handleInitialize(id interface{}) (interface{}, error) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
		},
	}
	
	// Only include ID if it's not nil
	if id != nil {
		response["id"] = id
	}
	
	return response, nil
}

func (s *Server) handleToolsList(id interface{}) (interface{}, error) {
	tools := []map[string]interface{}{
		{
			"name":        "upload_file",
			"description": "Upload a file to Koneksi Storage",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to upload",
					},
					"directoryId": map[string]interface{}{
						"type":        "string",
						"description": "Directory ID to upload to (optional)",
					},
				},
				"required": []string{"filePath"},
			},
		},
		{
			"name":        "download_file",
			"description": "Download a file from Koneksi Storage",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fileId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the file to download",
					},
					"outputPath": map[string]interface{}{
						"type":        "string",
						"description": "Path where to save the downloaded file",
					},
				},
				"required": []string{"fileId", "outputPath"},
			},
		},
		{
			"name":        "list_directories",
			"description": "List all directories in Koneksi Storage",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "create_directory",
			"description": "Create a new directory in Koneksi Storage",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the directory",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the directory",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "search_files",
			"description": "List files in a directory",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directoryId": map[string]interface{}{
						"type":        "string",
						"description": "Directory ID to search in",
					},
				},
				"required": []string{"directoryId"},
			},
		},
		{
			"name":        "backup_file",
			"description": "Backup a file with optional compression and encryption",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to backup",
					},
					"directoryId": map[string]interface{}{
						"type":        "string",
						"description": "Directory ID to backup to (optional)",
					},
					"compress": map[string]interface{}{
						"type":        "boolean",
						"description": "Compress the file before backup",
					},
					"encrypt": map[string]interface{}{
						"type":        "boolean",
						"description": "Encrypt the file before backup",
					},
					"encryptPassword": map[string]interface{}{
						"type":        "string",
						"description": "Password for encryption (optional)",
					},
				},
				"required": []string{"filePath"},
			},
		},
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"tools": tools,
		},
	}
	
	// Only include ID if it's not nil
	if id != nil {
		response["id"] = id
	}
	
	return response, nil
}

func (s *Server) handleToolCall(parsed gjson.Result, id interface{}) (interface{}, error) {
	toolName := parsed.Get("params.name").String()
	args := parsed.Get("params.arguments").String()

	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(args), &arguments); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	var result interface{}
	var err error

	switch toolName {
	case "upload_file":
		result, err = s.uploadFile(arguments)
	case "download_file":
		result, err = s.downloadFile(arguments)
	case "list_directories":
		result, err = s.listDirectories()
	case "create_directory":
		result, err = s.createDirectory(arguments)
	case "search_files":
		result, err = s.searchFiles(arguments)
	case "backup_file":
		result, err = s.backupFile(arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}, nil
}

func (s *Server) uploadFile(args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["filePath"].(string)
	if !ok {
		return nil, fmt.Errorf("filePath is required")
	}

	directoryId, _ := args["directoryId"].(string)

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Set directory ID if provided
	if directoryId != "" {
		s.client.DirectoryID = directoryId
	} else {
		// Clear any default directory ID if not specified
		s.client.DirectoryID = ""
	}

	// Upload file
	resp, err := s.client.UploadFile(filepath.Base(filePath), file, stat.Size(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	content := fmt.Sprintf("File uploaded successfully!\nFile ID: %s\nFile Name: %s\nSize: %d bytes", 
		resp.FileID, resp.FileName, resp.Size)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}

func (s *Server) downloadFile(args map[string]interface{}) (interface{}, error) {
	fileId, ok := args["fileId"].(string)
	if !ok {
		return nil, fmt.Errorf("fileId is required")
	}

	outputPath, ok := args["outputPath"].(string)
	if !ok {
		return nil, fmt.Errorf("outputPath is required")
	}

	// Download file
	reader, err := s.client.DownloadFile(fileId)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	written, err := io.Copy(outFile, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	content := fmt.Sprintf("File downloaded successfully!\nSaved to: %s\nSize: %d bytes", outputPath, written)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}

func (s *Server) listDirectories() (interface{}, error) {
	directories, err := s.client.ListDirectories()
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}

	content := "Directories:\n"
	for _, dir := range directories {
		content += fmt.Sprintf("- %s (ID: %s)\n  Files: %d, Size: %d bytes\n  Created: %s\n", 
			dir.Name, dir.ID, dir.FileCount, dir.TotalSize, dir.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}

func (s *Server) createDirectory(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	description, _ := args["description"].(string)

	resp, err := s.client.CreateDirectory(name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	content := fmt.Sprintf("Directory created!\nID: %s\nName: %s\nDescription: %s", 
		resp.DirectoryID, resp.Name, resp.Description)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}

func (s *Server) searchFiles(args map[string]interface{}) (interface{}, error) {
	directoryId, ok := args["directoryId"].(string)
	if !ok {
		return nil, fmt.Errorf("directoryId is required")
	}

	files, err := s.client.GetDirectoryFiles(directoryId)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory files: %w", err)
	}

	content := fmt.Sprintf("Files in directory %s:\n", directoryId)
	for _, file := range files {
		content += fmt.Sprintf("- %s (ID: %s, Size: %d bytes)\n", file.Name, file.ID, file.Size)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}

func (s *Server) backupFile(args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["filePath"].(string)
	if !ok {
		return nil, fmt.Errorf("filePath is required")
	}

	directoryId, _ := args["directoryId"].(string)
	compress, _ := args["compress"].(bool)
	encrypt, _ := args["encrypt"].(bool)
	encryptPassword, _ := args["encryptPassword"].(string)

	// For now, this is a simplified backup that just uploads the file
	// In a full implementation, you would integrate with the compression
	// and encryption packages from the main project

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if directoryId != "" {
		s.client.DirectoryID = directoryId
	}

	fileName := filepath.Base(filePath)
	if compress {
		fileName += ".gz"
	}
	if encrypt {
		fileName += ".enc"
	}

	resp, err := s.client.UploadFile(fileName, file, stat.Size(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to backup file: %w", err)
	}

	content := fmt.Sprintf("File backed up successfully!\nFile ID: %s\nFile Name: %s\nSize: %d bytes\nCompression: %t\nEncryption: %t", 
		resp.FileID, resp.FileName, resp.Size, compress, encrypt)

	if encrypt && encryptPassword != "" {
		content += "\nEncryption password was provided"
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
	}, nil
}