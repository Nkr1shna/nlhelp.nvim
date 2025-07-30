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
4. **Ollama** (automatically installed by backend)

**Note:** The Go backend automatically handles Ollama installation and model management. It will install Ollama if not present and download the required model (llama3.2:3b) automatically on first run.

### Installation

```

#### 1. Install the Plugin

**Using Lazy.nvim:**
```lua
return {
  "Nkr1shna/nlhelp.nvim",
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
    'Nkr1shna/nlhelp.nvim',
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
Plug 'Nkr1shna/nlhelp.nvim'
call plug#end()

" Configuration
lua require("nvim-smart-keybind-search").setup({
  server_host = "localhost",
  server_port = 8080,
  auto_start = true,
  health_check_interval = 30,
})
```

#### 2. Setup Backend

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

2. **Check Ollama is running:**
   ```bash
   curl http://localhost:11434/api/tags

#### No Results Found

1. **Check database is populated:**
   ```bash
   ./health_check.sh

3. **Check query relevance threshold:**
   ```lua
   -- Lower the threshold in config
   min_relevance = 0.1


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
git clone https://github.com/Nkr1shna/nlhelp.nvim.git
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

- **Issues**: [GitHub Issues](https://github.com/Nkr1shna/nlhelp.nvim/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Nkr1shna/nlhelp.nvim/discussions)
- **Documentation**: [Wiki](https://github.com/Nkr1shna/nlhelp.nvim/wiki)