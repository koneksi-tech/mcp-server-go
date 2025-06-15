# Koneksi MCP Server (Go)

A Go-based MCP (Model Context Protocol) server that provides AI assistants with access to Koneksi Storage for secure file storage and backup operations.

## Features

- Upload files to Koneksi Storage
- Download files from Koneksi Storage
- Create and manage directories
- List and search files
- Backup files with optional compression and encryption
- Secure authentication using API keys
- Lightweight Go implementation

## Installation

1. Navigate to the mcp-server directory:
```bash
cd mcp-server
```

2. Install dependencies:
```bash
go mod download
```

3. Build the project:
```bash
go build -o koneksi-mcp main.go
```

4. Configure your Koneksi API credentials:
```bash
cp .env.example .env
# Edit .env with your API credentials
```

## Configuration

Set the following environment variables in your `.env` file:

- `KONEKSI_API_CLIENT_ID`: Your Koneksi API client ID
- `KONEKSI_API_CLIENT_SECRET`: Your Koneksi API client secret
- `KONEKSI_API_BASE_URL`: (Optional) Koneksi API base URL

## Usage

### With Claude Desktop

Add the server to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "koneksi": {
      "command": "/path/to/koneksi-backup-cli/mcp-server/koneksi-mcp",
      "env": {
        "KONEKSI_API_CLIENT_ID": "your_client_id",
        "KONEKSI_API_CLIENT_SECRET": "your_client_secret"
      }
    }
  }
}
```

### Available Tools

1. **upload_file**: Upload a file to Koneksi Storage
   - `filePath`: Path to the file to upload
   - `directoryId`: (Optional) Directory ID to upload to

2. **download_file**: Download a file from Koneksi Storage
   - `fileId`: ID of the file to download
   - `outputPath`: Path where to save the file

3. **list_directories**: List all directories

4. **create_directory**: Create a new directory
   - `name`: Name of the directory
   - `description`: (Optional) Description

5. **search_files**: List files in a directory
   - `directoryId`: Directory ID to search in

6. **backup_file**: Backup a file with optional compression and encryption
   - `filePath`: Path to the file to backup
   - `directoryId`: (Optional) Directory ID to backup to
   - `compress`: (Optional) Compress the file before backup
   - `encrypt`: (Optional) Encrypt the file before backup
   - `encryptPassword`: (Optional) Password for encryption

## Development

Run the server:
```bash
go run main.go
```

Build for different platforms:
```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o koneksi-mcp-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o koneksi-mcp-darwin-arm64

# Linux
GOOS=linux GOARCH=amd64 go build -o koneksi-mcp-linux-amd64

# Windows
GOOS=windows GOARCH=amd64 go build -o koneksi-mcp-windows-amd64.exe
```

## Testing

Test the MCP server:
```bash
# Run the server
./koneksi-mcp

# In another terminal, send a test request
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | ./koneksi-mcp
```

## Example Usage with Claude

Once configured, you can use commands like:

- "Upload the file /path/to/document.pdf to Koneksi Storage"
- "Download file with ID abc123 to my desktop"
- "List all my directories in Koneksi Storage"
- "Create a new directory called 'Project Files' for my project backups"
- "Show me all files in directory xyz789"
- "Backup the file /path/to/important.doc with compression and encryption"

## License

MIT