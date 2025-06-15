# Koneksi MCP Bridge API

This REST API acts as a bridge between HTTP clients and the MCP (Model Context Protocol) server, allowing you to send messages to AI assistants through standard HTTP requests. It includes a web UI for easy interaction.

## Architecture

```
HTTP Client → REST API Bridge → MCP Server → AI Assistant
```

## Running the Bridge

```bash
# From the project root
go run ./cmd/koneksi-mcp-bridge

# Or using make
make run-bridge

# The bridge will automatically start the MCP server as a subprocess
# Open http://localhost:8081 in your browser to access the UI
```

## Web UI

The bridge includes a web UI accessible at http://localhost:8081 that provides:

- **Quick Actions**: Buttons for common operations
- **Simple Mode**: Forms for each tool with proper inputs
- **Advanced Mode**: Raw MCP request interface for testing
- **Real-time Response Display**: See MCP responses formatted nicely
- **Connection Status**: Monitor bridge connectivity

## API Endpoints

### File Upload Endpoint
```bash
POST /api/v1/upload
Content-Type: multipart/form-data

Form fields:
- file: The file to upload (required)
- directory_id: Target directory ID (optional)

# Example with curl:
curl -X POST http://localhost:8081/api/v1/upload \
  -F "file=@/path/to/file.txt" \
  -F "directory_id=optional-dir-id"
```

### 1. Send Raw MCP Request
```bash
POST /api/v1/mcp/request
Content-Type: application/json

{
  "method": "initialize",
  "params": {}
}
```

### 2. List Available Tools
```bash
GET /api/v1/mcp/tools/list


# Example response:
{
  "success": true,
  "result": {
    "tools": [
      {
        "name": "upload_file",
        "description": "Upload a file to Koneksi Storage",
        "inputSchema": {...}
      },
      ...
    ]
  }
}
```

### 3. Call a Tool
```bash
POST /api/v1/mcp/tools/call
Content-Type: application/json

{
  "name": "list_directories",
  "arguments": {}
}

# Example: Upload a file
{
  "name": "upload_file",
  "arguments": {
    "filePath": "/path/to/file.txt",
    "directoryId": "optional-dir-id"
  }
}

# Example: Create directory
{
  "name": "create_directory",
  "arguments": {
    "name": "My Directory",
    "description": "A test directory"
  }
}
```

## Example Usage with curl

### List available tools:
```bash
curl http://localhost:8081/api/v1/mcp/tools/list
```

### List directories:
```bash
curl -X POST http://localhost:8081/api/v1/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "list_directories",
    "arguments": {}
  }'
```

### Create a directory:
```bash
curl -X POST http://localhost:8081/api/v1/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "create_directory",
    "arguments": {
      "name": "Test Directory",
      "description": "Created via API"
    }
  }'
```

### Upload a file:
```bash
# First, create a test file
echo "Hello from API" > test.txt

# Then upload it
curl -X POST http://localhost:8081/api/v1/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "upload_file",
    "arguments": {
      "filePath": "./test.txt"
    }
  }'
```

## WebSocket Connection

For real-time bidirectional communication:

```javascript
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onopen = () => {
  // Send MCP request
  ws.send(JSON.stringify({
    method: 'tools/list',
    params: {}
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log('MCP Response:', response);
};
```

## Response Format

All responses follow this format:

```json
{
  "success": true|false,
  "result": {...},  // Present when success=true
  "error": "..."    // Present when success=false
}
```

## Available MCP Tools

1. **upload_file** - Upload a file to Koneksi Storage
2. **download_file** - Download a file from Koneksi Storage
3. **list_directories** - List all directories
4. **create_directory** - Create a new directory
5. **search_files** - List files in a directory
6. **backup_file** - Backup a file with optional compression/encryption

## Environment Variables

The bridge uses the same environment variables as the MCP server:

- `KONEKSI_API_CLIENT_ID`: Your Koneksi API client ID
- `KONEKSI_API_CLIENT_SECRET`: Your Koneksi API client secret
- `KONEKSI_API_BASE_URL`: Koneksi API base URL (optional)
- `PORT`: Bridge server port (default: 8081)

## Use Cases

1. **Web Applications**: Integrate AI-powered file operations into your web app
2. **Automation**: Script complex file management tasks using AI assistance
3. **Testing**: Test MCP server functionality via REST API
4. **Integration**: Bridge between different systems and AI assistants