# Nvim Smart Keybind Search

A Neovim plugin that provides intelligent, natural language search for keybindings and vim motions using RAG-based search. Find the right keybinding by describing what you want to do in plain English.

## Features

- 🔍 **Natural Language Search**: Search for keybindings using natural language queries
- 🧠 **RAG-Powered**: Uses Retrieval-Augmented Generation for intelligent results
- 📚 **Comprehensive Database**: Pre-populated with extensive vim knowledge base
- ⚡ **Fast Response**: Sub-2-second response times
- 🔄 **Real-time Updates**: Automatically syncs your custom keybindings
- 🏥 **Health Monitoring**: Built-in health checks and monitoring
- 🎯 **Smart Ranking**: Results ranked by relevance and context

## Quick Start

### Prerequisites

1. **Neovim 0.7+**
2. **Go 1.21+** (for backend)
3. **Python 3.8+** (for ChromaDB)
4. **Ollama** (for LLM inference)

### Installation

#### 1. Install Dependencies

**Install Ollama:**
```bash
# macOS/Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download from https://ollama.ai
```

**Download a model:**
```bash
ollama pull llama2:7b
```

#### 2. Install the Plugin

**Using Lazy.nvim:**
```lua
return {
  "nest/nvim-smart-keybind-search",
  event = "VeryLazy",
  config = function()
    require("nvim-smart-keybind-search").setup({
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
```

**Using Packer.nvim:**
```lua
return require('packer').startup(function(use)
  use {
    'nest/nvim-smart-keybind-search',
    config = function()
      require("nvim-smart-keybind-search").setup({
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
```

**Using vim-plug:**
```vim
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
```

#### 3. Setup Backend

**Automatic Setup (Recommended):**
```bash
# Run the installation script
./scripts/install.sh  # Linux/macOS
# or
scripts\install.bat   # Windows
```

**Manual Setup:**
```bash
# Build the Go backend
go build -o server cmd/server/main.go

# Build the database
go run scripts/build_database.go

# Start the server
./server
```

### Usage

#### Basic Search

1. **Open the search interface:**
   ```vim
   :SmartKeybindSearch
   ```

2. **Type your query in natural language:**
   - "delete current line"
   - "move to next word"
   - "copy text to clipboard"
   - "find and replace text"

3. **Select and execute the keybinding**

#### Commands

- `:SmartKeybindSearch [query]` - Search for keybindings
- `:SmartKeybindSync` - Sync your custom keybindings
- `:SmartKeybindHealth` - Check system health

#### Keymaps

The plugin automatically sets up these keymaps:
- `<leader>ks` - Open search interface
- `<leader>kh` - Show health status

## Configuration

### Plugin Configuration

```lua
require("nvim-smart-keybind-search").setup({
  -- Server configuration
  server_host = "localhost",
  server_port = 8080,
  
  -- Auto-start server when Neovim starts
  auto_start = true,
  
  -- Health check interval (seconds)
  health_check_interval = 30,
  
  -- Search configuration
  search = {
    -- Maximum number of results
    max_results = 10,
    
    -- Minimum relevance score (0.0 - 1.0)
    min_relevance = 0.3,
    
    -- Include explanations in results
    show_explanations = true,
  },
  
  -- UI configuration
  ui = {
    -- Use telescope for results display
    use_telescope = true,
    
    -- Custom telescope configuration
    telescope_config = {
      -- Your telescope config here
    },
  },
  
  -- Logging configuration
  logging = {
    level = "info",
    file = "./logs/plugin.log",
  },
})
```

### Server Configuration

The server can be configured via `config.json`:

```json
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
```

## Architecture

### Components

- **Go Backend**: JSON-RPC server handling queries and database operations
- **ChromaDB**: Vector database for semantic search
- **Ollama**: Local LLM for query understanding and response generation
- **Lua Plugin**: Neovim integration and UI

### Data Flow

1. User enters natural language query
2. Query is sent to Go backend via JSON-RPC
3. Backend vectorizes query and searches ChromaDB
4. Results are processed by LLM for relevance ranking
5. Ranked results are returned to Neovim
6. User selects and executes keybinding

## API Reference

### RPC Methods

#### Query
```json
{
  "method": "Query",
  "params": {
    "query": "delete current line",
    "limit": 10
  }
}
```

Response:
```json
{
  "results": [
    {
      "keybinding": {
        "keys": "dd",
        "command": "delete line",
        "description": "Delete current line",
        "mode": "n"
      },
      "relevance": 0.95,
      "explanation": "This is the standard vim command to delete the current line."
    }
  ],
  "reasoning": "The query matches the delete line command."
}
```

#### SyncKeybindings
```json
{
  "method": "SyncKeybindings",
  "params": {
    "keybindings": [
      {
        "keys": "<leader>w",
        "command": "save file",
        "description": "Save current file",
        "mode": "n"
      }
    ],
    "clear_existing": false
  }
}
```

#### HealthCheck
```json
{
  "method": "HealthCheck",
  "params": {}
}
```

## Troubleshooting

### Common Issues

#### Server Won't Start

1. **Check if port is in use:**
   ```bash
   lsof -i :8080  # Linux/macOS
   netstat -an | findstr :8080  # Windows
   ```

2. **Check Ollama is running:**
   ```bash
   curl http://localhost:11434/api/tags
   ```

3. **Check ChromaDB is running:**
   ```bash
   curl http://localhost:8000/api/v1/heartbeat
   ```

#### No Results Found

1. **Check database is populated:**
   ```bash
   ./health_check.sh
   ```

2. **Re-sync keybindings:**
   ```vim
   :SmartKeybindSync
   ```

3. **Check query relevance threshold:**
   ```lua
   -- Lower the threshold in config
   min_relevance = 0.1
   ```

#### Slow Response Times

1. **Check server resources:**
   ```bash
   ./health_check.sh
   ```

2. **Optimize Ollama model:**
   ```bash
   # Use a smaller model
   ollama pull llama2:3b
   ```

3. **Check network connectivity:**
   ```bash
   ping localhost
   ```

### Health Checks

Run comprehensive health checks:

```bash
# Check all components
./health_check.sh

# Check from Neovim
:SmartKeybindHealth
```

### Logs

Check logs for detailed error information:

```bash
# Server logs
tail -f logs/server.log

# Plugin logs
tail -f logs/plugin.log
```

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/nest/nvim-smart-keybind-search.git
cd nvim-smart-keybind-search

# Install dependencies
go mod download

# Build the server
go build -o server cmd/server/main.go

# Build the database
go run scripts/build_database.go

# Run tests
go test ./...
```

### Project Structure

```
.
├── cmd/
│   └── server/          # Go backend service entry point
├── internal/
│   ├── chromadb/        # ChromaDB client and operations
│   ├── interfaces/      # Core interface definitions
│   ├── keybindings/     # Keybinding scanning and vectorization
│   ├── ollama/          # Ollama client and model management
│   ├── rag/             # RAG agent implementation
│   └── server/          # JSON-RPC server implementation
├── lua/
│   └── nvim-smart-keybind-search/  # Lua plugin code
├── scripts/
│   ├── build_database.go # Database initialization
│   ├── install.sh       # Installation script (Linux/macOS)
│   └── install.bat      # Installation script (Windows)
└── plugin/
    └── nvim-smart-keybind-search.vim  # Vim plugin file
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test ./internal/server -v

# Run benchmarks
go test ./internal/server -bench=.

# Run Lua tests
busted lua/
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [ChromaDB](https://www.trychroma.com/) for vector database
- [Ollama](https://ollama.ai/) for local LLM inference
- [Telescope.nvim](https://github.com/nvim-telescope/telescope.nvim) for UI
- [snack.nvim](https://github.com/nvim-snack/snack.nvim) for picker interface

## Support

- **Issues**: [GitHub Issues](https://github.com/nest/nvim-smart-keybind-search/issues)
- **Discussions**: [GitHub Discussions](https://github.com/nest/nvim-smart-keybind-search/discussions)
- **Documentation**: [Wiki](https://github.com/nest/nvim-smart-keybind-search/wiki)