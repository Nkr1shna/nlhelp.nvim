-- panvimdoc configuration for nvim-smart-keybind-search
-- This file configures how panvimdoc generates the vim help documentation

return {
  -- Plugin name
  name = "nvim-smart-keybind-search",
  
  -- Plugin description
  description = "Intelligent keybinding search plugin using RAG-based search",
  
  -- Author information
  author = "nest",
  license = "MIT",
  repository = "https://github.com/nest/nvim-smart-keybind-search",
  
  -- Documentation files to process
  files = {
    "lua/nvim-smart-keybind-search/init.lua",
    "lua/nvim-smart-keybind-search/config.lua",
    "lua/nvim-smart-keybind-search/picker.lua",
    "lua/nvim-smart-keybind-search/rpc_client.lua",
    "lua/nvim-smart-keybind-search/keybind_scanner.lua",
    "lua/nvim-smart-keybind-search/health.lua",
    "lua/nvim-smart-keybind-search/utils.lua",
  },
  
  -- Output file
  output_file = "doc/nvim-smart-keybind-search.txt",
  
  -- Tags to include
  tags = {
    "nvim-smart-keybind-search",
    "nvim-smart-keybind-search-introduction",
    "nvim-smart-keybind-search-installation",
    "nvim-smart-keybind-search-quick-start",
    "nvim-smart-keybind-search-configuration",
    "nvim-smart-keybind-search-commands",
    "nvim-smart-keybind-search-api",
    "nvim-smart-keybind-search-troubleshooting",
    "nvim-smart-keybind-search-setup",
    "nvim-smart-keybind-search-search",
    "nvim-smart-keybind-search-sync",
    "nvim-smart-keybind-search-force-sync",
    "nvim-smart-keybind-search-health",
    "nvim-smart-keybind-search-get-config",
    "nvim-smart-keybind-search-is-initialized",
    "nvim-smart-keybind-search-get-scanner-stats",
  },
  
  -- Sections to include
  sections = {
    "introduction",
    "installation", 
    "quick-start",
    "configuration",
    "commands",
    "keymaps",
    "api-reference",
    "troubleshooting",
    "credits",
  },
  
  -- Custom sections
  custom_sections = {
    ["introduction"] = {
      title = "INTRODUCTION",
      tag = "nvim-smart-keybind-search-introduction",
      content = [[
nvim-smart-keybind-search is a Neovim plugin that provides intelligent,
natural language search for keybindings and vim motions using
Retrieval-Augmented Generation (RAG) technology.

Features:
• Natural language search for keybindings
• RAG-powered intelligent results
• Comprehensive vim knowledge base
• Real-time custom keybinding sync
• Health monitoring and diagnostics
• Fast sub-2-second response times
      ]],
    },
    
    ["installation"] = {
      title = "INSTALLATION",
      tag = "nvim-smart-keybind-search-installation",
      content = [[
Prerequisites:
• Neovim 0.7+
• Go 1.21+ (for backend)
• Python 3.8+ (for ChromaDB)
• Ollama (for LLM inference)

1. Install Ollama:
   https://ollama.ai

2. Download a model:
   >ollama pull llama3.2:3b

3. Install the plugin using your plugin manager:

   Lazy.nvim:
   >lua
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
   <

   Packer.nvim:
   >lua
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
   <

   vim-plug:
   >vim
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
   <

4. Setup backend:
   >bash
   ./scripts/install.sh  # Linux/macOS
   # or
   scripts\install.bat   # Windows
   <
      ]],
    },
    
    ["quick-start"] = {
      title = "QUICK START",
      tag = "nvim-smart-keybind-search-quick-start",
      content = [[
1. Start the server:
   >bash
   ./start_server.sh
   <

2. Open the search interface:
   >vim
   :SmartKeybindSearch
   <

3. Type your query in natural language:
   • "delete current line"
   • "move to next word"
   • "copy text to clipboard"
   • "find and replace text"

4. Select and execute the keybinding
      ]],
    },
    
    ["configuration"] = {
      title = "CONFIGURATION",
      tag = "nvim-smart-keybind-search-configuration",
      content = [[
Basic Configuration:
>lua
require("nvim-smart-keybind-search").setup({
  -- Server connection
  server_host = "localhost",
  server_port = 8080,
  
  -- Auto-start server when Neovim starts
  auto_start = true,
  
  -- Health check interval (seconds)
  health_check_interval = 30,
})
<

Advanced Configuration:
>lua
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
    -- Maximum number of results to show
    max_results = 10,
    
    -- Minimum relevance score (0.0 - 1.0)
    min_relevance = 0.3,
    
    -- Show explanations in results
    show_explanations = true,
    
    -- Include your custom keybindings in search
    include_custom = true,
    
    -- Prioritize custom keybindings
    prioritize_custom = true,
  },
  
  -- UI configuration
  ui = {
    -- Use telescope for results display
    use_telescope = true,
    
    -- Custom telescope configuration
    telescope_config = {
      layout_config = {
        width = 0.8,
        height = 0.6,
      },
      sorting_strategy = "ascending",
    },
    
    -- Custom keymaps for results
    keymaps = {
      execute = "<CR>",
      copy = "c",
      explain = "e",
      close = "q",
    },
  },
  
  -- Logging configuration
  logging = {
    level = "info",  -- debug, info, warn, error
    file = "./logs/plugin.log",
    max_size = "10MB",
    max_files = 5,
  },
})
<
      ]],
    },
    
    ["commands"] = {
      title = "COMMANDS",
      tag = "nvim-smart-keybind-search-commands",
      content = [[
:SmartKeybindSearch [query]		*:SmartKeybindSearch*
	Search for keybindings using natural language.
	If [query] is provided, it will be used as the initial search term.

:SmartKeybindSync			*:SmartKeybindSync*
	Force sync all keybindings with the backend.

:SmartKeybindHealth			*:SmartKeybindHealth*
	Check the health status of the plugin and backend services.
      ]],
    },
    
    ["keymaps"] = {
      title = "KEYMAPS",
      tag = "nvim-smart-keybind-search-keymaps",
      content = [[
The plugin automatically sets up these keymaps:

<leader>ks				*<leader>ks*
	Open the search interface.

<leader>kh				*<leader>kh*
	Show health status.
      ]],
    },
    
    ["troubleshooting"] = {
      title = "TROUBLESHOOTING",
      tag = "nvim-smart-keybind-search-troubleshooting",
      content = [[
Common Issues:

1. Server Won't Start:
   • Check if port is in use: lsof -i :8080
   • Kill existing processes: pkill -f server
   • Check server logs: tail -f logs/server.log

2. Ollama Not Running:
   • Install Ollama: https://ollama.ai
   • Start Ollama: ollama serve
   • Download model: ollama pull llama3.2:3b
   • Check status: curl http://localhost:11434/api/tags

3. ChromaDB Issues:
   • Install ChromaDB: pip install chromadb==0.4.22
   • Start ChromaDB: chroma run --path ./data/chroma --host localhost --port 8000
   • Rebuild database: go run scripts/build_database.go

4. No Search Results:
   • Check database population: curl -X GET "http://localhost:8000/api/v1/collections/keybindings/count"
   • Re-sync keybindings: :SmartKeybindSync
   • Lower relevance threshold in config

5. Slow Response Times:
   • Check system resources: top
   • Use smaller model: ollama pull llama3.2:3b
   • Check network connectivity: ping localhost
   • Increase timeout in config

Health Checks:
>bash
# Check all components
./health_check.sh

# Check from Neovim
:SmartKeybindHealth
<

Logs:
>bash
# Server logs
tail -f logs/server.log

# Plugin logs
tail -f logs/plugin.log
<
      ]],
    },
    
    ["credits"] = {
      title = "CREDITS",
      tag = "nvim-smart-keybind-search-credits",
      content = [[
Author: nest
License: MIT
Repository: https://github.com/nest/nvim-smart-keybind-search

Acknowledgments:
• ChromaDB for vector database
• Ollama for local LLM inference
• Telescope.nvim for UI
• snack.nvim for picker interface
      ]],
    },
  },
  
  -- Options for panvimdoc
  options = {
    -- Include function signatures
    include_signatures = true,
    
    -- Include parameter descriptions
    include_parameters = true,
    
    -- Include return value descriptions
    include_returns = true,
    
    -- Include usage examples
    include_examples = true,
    
    -- Include tags
    include_tags = true,
    
    -- Format code blocks
    format_code_blocks = true,
    
    -- Include table of contents
    include_toc = true,
  },
} 