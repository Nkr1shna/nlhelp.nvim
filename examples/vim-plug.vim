" vim-plug configuration for nvim-smart-keybind-search
" Add this to your vim-plug configuration

Plug 'your-username/nvim-smart-keybind-search'
Plug 'snacks.nvim' " Required for picker interface

" Configuration (add to your init.lua or after plug#end())
lua << EOF
require("nvim-smart-keybind-search").setup({
  -- Server configuration
  server_host = "localhost",
  server_port = 8080,
  auto_start = true,
  
  -- Database configuration
  auto_sync = true,        -- Automatically sync keybindings on startup
  watch_changes = true,    -- Watch for keybinding changes
  
  -- Keymaps
  keymaps = {
    search = "<leader>ks",  -- Search keybindings
    health = "<leader>kh",  -- Health check
  },
  
  -- Picker configuration
  picker = {
    max_results = 10,
    min_relevance = 0.3,
  },
  
  -- UI configuration
  ui = {
    show_explanations = true,
  },
})
EOF