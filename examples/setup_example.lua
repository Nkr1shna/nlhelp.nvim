-- Example setup for nvim-smart-keybind-search
-- Add this to your Neovim configuration

-- Basic setup with auto-detected binary path
require("nvim-smart-keybind-search").setup({
	-- Backend configuration (stdin/stdout JSON-RPC)
	backend = {
		-- The plugin will auto-detect the binary path from:
		-- 1. Plugin directory (./server or ./main)
		-- 2. Data directory (~/.local/share/nvim/nvim-smart-keybind-search/server)
		-- You can also specify it manually:
		-- binary_path = "/path/to/your/server/binary",
		auto_start = true,
		timeout = 5000, -- 5 second timeout for RPC calls
		log_level = "info",
	},

	-- Keymaps
	keymaps = {
		search = "<leader>sk", -- Search keybindings
		search_alt = "<leader>fk", -- Alternative search
	},

	-- Picker configuration
	picker = {
		provider = "snack", -- or "telescope" or "fzf-lua"
		max_results = 20,
		show_explanations = true,
		show_relevance = true,
		debounce_ms = 300, -- Debounce search input
	},

	-- Scanner configuration
	scanner = {
		include_plugins = true,
		include_buffer_local = true,
	},

	-- Auto-sync settings
	auto_sync = true,
	sync_on_startup = true,
	watch_changes = true,

	-- Logging
	logging = {
		debug = false,
		level = "info",
	},
})

-- The plugin communicates with the Go backend via stdin/stdout JSON-RPC
-- No HTTP server or network ports are used!
