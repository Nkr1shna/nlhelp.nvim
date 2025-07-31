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

## Prerequisites

- **Neovim 0.8+**
- **Go 1.21+** (for the backend server)
- **Python 3.8+** (for ChromaDB)
- **Internet connection** (for initial HuggingFace dataset download)

**Note:** The Go backend automatically handles Ollama installation and model management. It will install Ollama if not present and download the required model (llama3.2:3b) automatically on first run.

## Installation

### lazy.nvim (Recommended)

```lua
{
  "Nkr1shna/nlhelp.nvim",
  event = "VeryLazy",
  dependencies = {
    "snacks.nvim", -- Required for picker interface
    "nvim-telescope/telescope.nvim", -- Alternative picker
  },
  config = function()
    require("nvim-smart-keybind-search").setup({
      server_host = "localhost",
      server_port = 8080,
      auto_start = true,
      health_check_interval = 30,
      keymaps = {
        search = "<leader>ks",  -- Search keybindings
        health = "<leader>kh",  -- Health check
      },
    })
  end,
  cmd = { "SmartKeybindSearch", "SmartKeybindSync", "SmartKeybindHealth" },
  keys = {
    { "<leader>ks", "<cmd>SmartKeybindSearch<cr>", desc = "Smart keybinding search" },
  },
}
```

### packer.nvim

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
        keymaps = {
          search = "<leader>ks",
          health = "<leader>kh",
        },
      })
    end,
    requires = {
      'snacks.nvim',
      'nvim-telescope/telescope.nvim',
    }
  }
end)
```

### vim-plug

```vim
call plug#begin()
Plug 'Nkr1shna/nlhelp.nvim'
Plug 'snacks.nvim'
Plug 'nvim-telescope/telescope.nvim'
call plug#end()

" Configuration
lua require("nvim-smart-keybind-search").setup({
  server_host = "localhost",
  server_port = 8080,
  auto_start = true,
  health_check_interval = 30,
})
```

## What Happens on First Run

When you call `setup()`, the plugin automatically:

1. **🔧 Installs dependencies** - Ollama, ChromaDB, and required Python packages
2. **📚 Populates built-in knowledge** - From Neovim quick reference
3. **🌐 Downloads general knowledge** - From HuggingFace dataset (aegis-nvim/neovim-help)
4. **⌨️  Scans your keybindings** - Extracts and vectorizes your current keybindings
5. **🚀 Starts the backend server** - Ready to search!

This happens in the background, so you can continue using Neovim while it sets up.

## Usage

Search for keybindings using natural language:

```vim
:SmartKeybindSearch
" or use <leader>ks (default keybinding)
```

Example queries:
- "delete a word"
- "jump to beginning of line"
- "split window vertically"
- "search and replace"

### Health Check

Verify everything is working:

```vim
:SmartKeybindHealth
```

## Configuration Options

```lua
require("nvim-smart-keybind-search").setup({
  -- Server settings
  server_host = "localhost",
  server_port = 8080,
  auto_start = true,
  health_check_interval = 30,
  
  -- Database settings
  auto_sync = true,        -- Auto-sync keybindings on startup
  watch_changes = true,    -- Watch for keybinding changes
  
  -- Keymaps
  keymaps = {
    search = "<leader>ks",
    health = "<leader>kh",
  },
  
  -- Search settings
  picker = {
    max_results = 10,
    min_relevance = 0.3,
  },
  
  -- UI settings
  ui = {
    show_explanations = true,
  },
})
```

## Troubleshooting

### Database Issues
```vim
:SmartKeybindHealth  " Check status
:SmartKeybindSync    " Force re-sync keybindings
```

### Server Issues
The plugin automatically manages the Go backend server. If you have issues:

1. Check that Go 1.21+ is installed: `go version`
2. Check the health status: `:SmartKeybindHealth`
3. Check logs in `~/.local/share/nvim-smart-keybind-search/logs/`

### Network Issues
If the HuggingFace dataset download fails, the plugin will use fallback data and retry later when you have internet.

### No Results Found

1. **Check database is populated:**
   ```bash
   ./health_check.sh
   ```

2. **Check Ollama is running:**
   ```bash
   curl http://localhost:11434/api/tags
   ```

3. **Check query relevance threshold:**
   ```lua
   -- Lower the threshold in config
   min_relevance = 0.1
   ```

## Project Structure

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
│   └── build_database.go # Database initialization
└── plugin/
    └── nvim-smart-keybind-search.vim  # Vim plugin file
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