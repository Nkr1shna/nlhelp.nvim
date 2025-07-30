#!/bin/bash

# nvim-smart-keybind-search installation script
# This script installs the Go backend, sets up the database, and configures the plugin

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PLUGIN_NAME="nvim-smart-keybind-search"
GO_VERSION="1.21"
CHROMADB_VERSION="0.4.22"
OLLAMA_VERSION="0.1.29"

# Platform detection
PLATFORM=$(uname -s)
ARCH=$(uname -m)

echo -e "${BLUE}Installing ${PLUGIN_NAME}...${NC}"

# Check if running in Neovim
if [ -n "$NVIM" ]; then
    echo -e "${YELLOW}Warning: This script should be run outside of Neovim${NC}"
fi

# Check dependencies
echo -e "${BLUE}Checking dependencies...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Please install Go ${GO_VERSION} or later.${NC}"
    echo -e "${YELLOW}Visit: https://golang.org/doc/install${NC}"
    exit 1
fi

GO_VERSION_CHECK=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | cut -c3-)
if [ "$(printf '%s\n' "$GO_VERSION" "$GO_VERSION_CHECK" | sort -V | head -n1)" != "$GO_VERSION" ]; then
    echo -e "${RED}Go version ${GO_VERSION} or later is required. Current version: ${GO_VERSION_CHECK}${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Go ${GO_VERSION_CHECK} found${NC}"

# Check Git
if ! command -v git &> /dev/null; then
    echo -e "${RED}Git is not installed. Please install Git.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Git found${NC}"

# Check if we're in the plugin directory
if [ ! -f "go.mod" ] || [ ! -f "internal/server/rpc.go" ]; then
    echo -e "${RED}This script must be run from the plugin root directory${NC}"
    exit 1
fi

# Build the Go binary
echo -e "${BLUE}Building Go backend...${NC}"

# Set build flags for better performance
export CGO_ENABLED=1
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)

# Build with optimizations
go build -ldflags="-s -w" -o server cmd/server/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Go backend built successfully${NC}"
else
    echo -e "${RED}Failed to build Go backend${NC}"
    exit 1
fi

# Build the database
echo -e "${BLUE}Building pre-populated database...${NC}"

# Check if ChromaDB is available
if ! command -v chroma &> /dev/null; then
    echo -e "${YELLOW}ChromaDB CLI not found. Installing ChromaDB...${NC}"
    
    # Install ChromaDB using pip
    if command -v pip3 &> /dev/null; then
        pip3 install chromadb==${CHROMADB_VERSION}
    elif command -v pip &> /dev/null; then
        pip install chromadb==${CHROMADB_VERSION}
    else
        echo -e "${RED}pip not found. Please install Python and pip first.${NC}"
        exit 1
    fi
fi

# Start ChromaDB server for database building
echo -e "${BLUE}Starting ChromaDB server for database initialization...${NC}"

# Start ChromaDB in background
chroma run --path ./data/chroma --host localhost --port 8000 &
CHROMADB_PID=$!

# Wait for ChromaDB to start
sleep 5

# Build the database
go run scripts/build_database.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Database built successfully${NC}"
else
    echo -e "${RED}Failed to build database${NC}"
    kill $CHROMADB_PID 2>/dev/null || true
    exit 1
fi

# Stop ChromaDB
kill $CHROMADB_PID 2>/dev/null || true

# Create installation directories
echo -e "${BLUE}Creating installation directories...${NC}"

# Create data directory
mkdir -p data/chroma
mkdir -p data/ollama

# Create logs directory
mkdir -p logs

# Set permissions
chmod +x server

echo -e "${GREEN}✓ Installation directories created${NC}"

# Create configuration file
echo -e "${BLUE}Creating configuration...${NC}"

cat > config.json << EOF
{
  "server": {
    "host": "localhost",
    "port": 8080,
    "timeout": 30
  },
  "chromadb": {
    "host": "localhost",
    "port": 8000,
    "path": "./data/chroma"
  },
  "ollama": {
    "host": "localhost",
    "port": 11434,
    "model": "llama2:7b"
  },
  "logging": {
    "level": "info",
    "file": "./logs/server.log"
  }
}
EOF

echo -e "${GREEN}✓ Configuration created${NC}"

# Create startup script
echo -e "${BLUE}Creating startup script...${NC}"

cat > start_server.sh << 'EOF'
#!/bin/bash

# Start script for nvim-smart-keybind-search server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load configuration
if [ -f "config.json" ]; then
    PORT=$(grep -o '"port":[[:space:]]*[0-9]*' config.json | grep -o '[0-9]*' | head -1)
else
    PORT=8080
fi

# Check if server is already running
if pgrep -f "server" > /dev/null; then
    echo "Server is already running"
    exit 0
fi

# Start ChromaDB
echo "Starting ChromaDB..."
chroma run --path ./data/chroma --host localhost --port 8000 &
CHROMADB_PID=$!

# Wait for ChromaDB to start
sleep 3

# Start the server
echo "Starting nvim-smart-keybind-search server..."
./server &
SERVER_PID=$!

# Save PIDs for cleanup
echo $CHROMADB_PID > .chromadb.pid
echo $SERVER_PID > .server.pid

echo "Server started on port $PORT"
echo "ChromaDB PID: $CHROMADB_PID"
echo "Server PID: $SERVER_PID"
EOF

chmod +x start_server.sh

# Create stop script
cat > stop_server.sh << 'EOF'
#!/bin/bash

# Stop script for nvim-smart-keybind-search server

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Stop server
if [ -f ".server.pid" ]; then
    SERVER_PID=$(cat .server.pid)
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo "Stopping server (PID: $SERVER_PID)..."
        kill $SERVER_PID
        rm .server.pid
    else
        echo "Server not running"
        rm -f .server.pid
    fi
fi

# Stop ChromaDB
if [ -f ".chromadb.pid" ]; then
    CHROMADB_PID=$(cat .chromadb.pid)
    if kill -0 $CHROMADB_PID 2>/dev/null; then
        echo "Stopping ChromaDB (PID: $CHROMADB_PID)..."
        kill $CHROMADB_PID
        rm .chromadb.pid
    else
        echo "ChromaDB not running"
        rm -f .chromadb.pid
    fi
fi

echo "All services stopped"
EOF

chmod +x stop_server.sh

# Create health check script
cat > health_check.sh << 'EOF'
#!/bin/bash

# Health check script for nvim-smart-keybind-search

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Checking nvim-smart-keybind-search health..."

# Check if server is running
if [ -f ".server.pid" ]; then
    SERVER_PID=$(cat .server.pid)
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo "✓ Server is running (PID: $SERVER_PID)"
    else
        echo "✗ Server is not running"
        rm -f .server.pid
    fi
else
    echo "✗ Server PID file not found"
fi

# Check if ChromaDB is running
if [ -f ".chromadb.pid" ]; then
    CHROMADB_PID=$(cat .chromadb.pid)
    if kill -0 $CHROMADB_PID 2>/dev/null; then
        echo "✓ ChromaDB is running (PID: $CHROMADB_PID)"
    else
        echo "✗ ChromaDB is not running"
        rm -f .chromadb.pid
    fi
else
    echo "✗ ChromaDB PID file not found"
fi

# Check if Ollama is running
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✓ Ollama is running"
else
    echo "✗ Ollama is not running"
fi

# Check database files
if [ -d "data/chroma" ]; then
    echo "✓ Database directory exists"
else
    echo "✗ Database directory missing"
fi

# Check configuration
if [ -f "config.json" ]; then
    echo "✓ Configuration file exists"
else
    echo "✗ Configuration file missing"
fi
EOF

chmod +x health_check.sh

echo -e "${GREEN}✓ Startup scripts created${NC}"

# Create plugin manager integration files
echo -e "${BLUE}Creating plugin manager integration...${NC}"

# For lazy.nvim
mkdir -p extras/lazy.nvim
cat > extras/lazy.nvim/init.lua << 'EOF'
return {
  "nest/nvim-smart-keybind-search",
  event = "VeryLazy",
  config = function()
    require("nvim-smart-keybind-search").setup({
      -- Configuration options
      server_host = "localhost",
      server_port = 8080,
      auto_start = true,
      health_check_interval = 30,
    })
  end,
  dependencies = {
    "nvim-telescope/telescope.nvim",
  },
}
EOF

# For packer.nvim
mkdir -p extras/packer.nvim
cat > extras/packer.nvim/init.lua << 'EOF'
return require('packer').startup(function(use)
  use {
    'nest/nvim-smart-keybind-search',
    config = function()
      require("nvim-smart-keybind-search").setup({
        -- Configuration options
        server_host = "localhost",
        server_port = 8080,
        auto_start = true,
        health_check_interval = 30,
      })
    end,
    requires = {
      'nvim-telescope/telescope.nvim',
    }
  }
end)
EOF

# For vim-plug
mkdir -p extras/vim-plug
cat > extras/vim-plug/init.vim << 'EOF'
call plug#begin()
Plug 'nest/nvim-smart-keybind-search'
call plug#end()

" Configuration
lua require("nvim-smart-keybind-search").setup({
  server_host = "localhost",
  server_port = 8080,
  auto_start = true,
  health_check_interval = 30,
})
EOF

echo -e "${GREEN}✓ Plugin manager integration created${NC}"

# Create systemd service file (for Linux)
if [ "$PLATFORM" = "Linux" ]; then
    echo -e "${BLUE}Creating systemd service...${NC}"
    
    SERVICE_USER=$(whoami)
    SERVICE_PATH=$(pwd)
    
    cat > nvim-smart-keybind-search.service << EOF
[Unit]
Description=nvim-smart-keybind-search Server
After=network.target

[Service]
Type=forking
User=$SERVICE_USER
WorkingDirectory=$SERVICE_PATH
ExecStart=$SERVICE_PATH/start_server.sh
ExecStop=$SERVICE_PATH/stop_server.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    echo -e "${GREEN}✓ Systemd service file created${NC}"
    echo -e "${YELLOW}To install the service, run:${NC}"
    echo -e "${BLUE}sudo cp nvim-smart-keybind-search.service /etc/systemd/system/${NC}"
    echo -e "${BLUE}sudo systemctl enable nvim-smart-keybind-search${NC}"
    echo -e "${BLUE}sudo systemctl start nvim-smart-keybind-search${NC}"
fi

# Create launchd plist (for macOS)
if [ "$PLATFORM" = "Darwin" ]; then
    echo -e "${BLUE}Creating launchd service...${NC}"
    
    SERVICE_USER=$(whoami)
    SERVICE_PATH=$(pwd)
    
    cat > com.nvim-smart-keybind-search.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.nvim-smart-keybind-search</string>
    <key>ProgramArguments</key>
    <array>
        <string>$SERVICE_PATH/start_server.sh</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>$SERVICE_PATH</string>
    <key>StandardOutPath</key>
    <string>$SERVICE_PATH/logs/launchd.log</string>
    <key>StandardErrorPath</key>
    <string>$SERVICE_PATH/logs/launchd.log</string>
</dict>
</plist>
EOF

    echo -e "${GREEN}✓ Launchd service file created${NC}"
    echo -e "${YELLOW}To install the service, run:${NC}"
    echo -e "${BLUE}cp com.nvim-smart-keybind-search.plist ~/Library/LaunchAgents/${NC}"
    echo -e "${BLUE}launchctl load ~/Library/LaunchAgents/com.nvim-smart-keybind-search.plist${NC}"
fi

echo -e "${GREEN}✓ Installation completed successfully!${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo -e "1. Install Ollama: ${YELLOW}https://ollama.ai${NC}"
echo -e "2. Download a model: ${YELLOW}ollama pull llama2:7b${NC}"
echo -e "3. Start the server: ${YELLOW}./start_server.sh${NC}"
echo -e "4. Configure your Neovim plugin manager"
echo ""
echo -e "${BLUE}Plugin manager examples:${NC}"
echo -e "• Lazy.nvim: ${YELLOW}extras/lazy.nvim/init.lua${NC}"
echo -e "• Packer.nvim: ${YELLOW}extras/packer.nvim/init.lua${NC}"
echo -e "• vim-plug: ${YELLOW}extras/vim-plug/init.vim${NC}"
echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo -e "• Start server: ${YELLOW}./start_server.sh${NC}"
echo -e "• Stop server: ${YELLOW}./stop_server.sh${NC}"
echo -e "• Health check: ${YELLOW}./health_check.sh${NC}" 