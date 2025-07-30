---@meta

---Configuration management for nvim-smart-keybind-search
---@tag nvim-smart-keybind-search-config
---
---This module handles user configuration merging with sensible defaults.
---It provides a comprehensive configuration system for all plugin features.
---
---@usage
---```lua
---local config = require("nvim-smart-keybind-search.config")
---local merged_config = config.setup({
---  server_host = "localhost",
---  server_port = 8080,
---  auto_start = true,
---})
---```

local M = {}

-- Default configuration
local default_config = {
	-- Backend service configuration
	backend = {
		-- Path to the Go backend binary
		binary_path = nil, -- Auto-detected if nil
		-- Backend service port (0 for auto-assign)
		port = 0,
		-- Timeout for backend operations (milliseconds)
		timeout = 5000,
		-- Auto-start backend if not running
		auto_start = true,
		-- Backend log level (debug, info, warn, error)
		log_level = "info",
	},

	-- Keybinding scanner configuration
	scanner = {
		-- Include plugin keybindings
		include_plugins = true,
		-- Include buffer-local keybindings
		include_buffer_local = true,
		-- Exclude patterns (lua patterns)
		exclude_patterns = {
			"^<Plug>", -- Exclude internal plugin mappings
			"^<SNR>", -- Exclude script-local mappings
		},
		-- Custom description providers
		description_providers = {},
	},

	-- Picker interface configuration
	picker = {
		-- Picker implementation ('snack' or 'telescope' or 'fzf-lua')
		provider = "snack",
		-- Minimum query length before searching
		min_query_length = 2,
		-- Debounce delay for real-time search (milliseconds)
		debounce_ms = 300,
		-- Maximum number of results to display
		max_results = 20,
		-- Show explanations in results
		show_explanations = true,
		-- Show relevance scores in results
		show_relevance = true,
		-- Show action menu for complex keybindings
		show_action_menu = true,
		-- Picker window configuration
		window = {
			width = 0.8,
			height = 0.6,
			border = "rounded",
		},
		-- Snack.nvim specific configuration
		snack_config = {
			-- Additional snack picker options can be added here
		},
	},

	-- Keymapping configuration
	keymaps = {
		-- Main search keymap (set to false to disable)
		search = "<leader>sk",
		-- Alternative search keymap
		search_alt = "<leader>fk",
	},

	-- Auto-sync configuration
	auto_sync = true,
	-- Sync on startup
	sync_on_startup = true,
	-- Watch for keybinding changes
	watch_changes = true,

	-- UI configuration
	ui = {
		-- Icons for different result types
		icons = {
			keybinding = "⌨️ ",
			motion = "🔄 ",
			command = "⚡ ",
			plugin = "🔌 ",
			visual = "👁️ ",
			insert = "✏️ ",
		},
		-- Highlight groups
		highlights = {
			query = "Question",
			keybinding = "Identifier",
			description = "Comment",
			explanation = "String",
		},
	},

	-- Logging configuration
	logging = {
		-- Enable debug logging
		debug = false,
		-- Log file path (nil for no file logging)
		file = nil,
		-- Log level (debug, info, warn, error)
		level = "info",
	},
}

--- Deep merge two tables
--- @param target table Target table to merge into
--- @param source table Source table to merge from
--- @return table Merged table
local function deep_merge(target, source)
	for key, value in pairs(source) do
		if type(value) == "table" and type(target[key]) == "table" then
			target[key] = deep_merge(target[key], value)
		else
			target[key] = value
		end
	end
	return target
end

--- Validate configuration values
--- @param config table Configuration to validate
--- @return boolean, string True if valid, or false with error message
local function validate_config(config)
	-- Validate backend configuration
	if config.backend.timeout and config.backend.timeout < 1000 then
		return false, "Backend timeout must be at least 1000ms"
	end

	-- Validate picker configuration
	if config.picker.min_query_length and config.picker.min_query_length < 1 then
		return false, "Minimum query length must be at least 1"
	end

	if config.picker.debounce_ms and config.picker.debounce_ms < 0 then
		return false, "Debounce delay cannot be negative"
	end

	if config.picker.max_results and config.picker.max_results < 1 then
		return false, "Maximum results must be at least 1"
	end

	-- Validate window configuration
	local window = config.picker.window
	if window.width and (window.width <= 0 or window.width > 1) then
		return false, "Window width must be between 0 and 1"
	end

	if window.height and (window.height <= 0 or window.height > 1) then
		return false, "Window height must be between 0 and 1"
	end

	-- Validate keymaps
	if config.keymaps.search and type(config.keymaps.search) ~= "string" and config.keymaps.search ~= false then
		return false, "Search keymap must be a string or false"
	end

	return true, nil
end

--- Auto-detect backend binary path
--- @return string|nil Path to backend binary or nil if not found
local function detect_binary_path()
	-- Common locations to check
	local possible_paths = {
		vim.fn.stdpath("data") .. "/nvim-smart-keybind-search/server",
		vim.fn.stdpath("config") .. "/nvim-smart-keybind-search/server",
		"./server", -- Development path
	}

	for _, path in ipairs(possible_paths) do
		if vim.fn.executable(path) == 1 then
			return path
		end
	end

	return nil
end

---Setup configuration with user overrides
---@tag nvim-smart-keybind-search-config-setup
---
---Merges user configuration with defaults and validates the result.
---Auto-detects backend binary path if not provided.
---
---@param user_config? table User configuration overrides
---@field user_config.server_host? string Backend server host (default: "localhost")
---@field user_config.server_port? number Backend server port (default: 8080)
---@field user_config.auto_start? boolean Auto-start server on setup (default: true)
---@field user_config.health_check_interval? number Health check interval in seconds (default: 30)
---@field user_config.auto_sync? boolean Auto-sync keybindings on setup (default: true)
---@field user_config.watch_changes? boolean Watch for keybinding changes (default: true)
---@field user_config.keymaps? table Keymap configuration
---@field user_config.picker? table Picker configuration
---@field user_config.ui? table UI configuration
---@field user_config.scanner? table Scanner configuration
---@field user_config.logging? table Logging configuration
---
---@usage
---```lua
---local config = require("nvim-smart-keybind-search.config")
---local merged_config = config.setup({
---  server_host = "localhost",
---  server_port = 8080,
---  auto_start = true,
---  keymaps = {
---    search = "<leader>ks",
---  },
---  picker = {
---    max_results = 10,
---    show_explanations = true,
---  },
---})
---```
---
---@return table Final merged configuration
function M.setup(user_config)
	-- Start with default configuration
	local config = vim.deepcopy(default_config)

	-- Merge user configuration
	if user_config then
		config = deep_merge(config, user_config)
	end

	-- Auto-detect binary path if not provided
	if not config.backend.binary_path then
		config.backend.binary_path = detect_binary_path()
	end

	-- Validate final configuration
	local valid, error_msg = validate_config(config)
	if not valid then
		error("Invalid configuration: " .. error_msg)
	end

	return config
end

---Get default configuration
---@tag nvim-smart-keybind-search-config-get-defaults
---
---Returns a copy of the default configuration.
---
---@usage
---```lua
---local config = require("nvim-smart-keybind-search.config")
---local defaults = config.get_defaults()
---print("Default server host:", defaults.server_host)
---```
---
---@return table Default configuration
function M.get_defaults()
	return vim.deepcopy(default_config)
end

---Validate a configuration table
---@tag nvim-smart-keybind-search-config-validate
---
---Validates a configuration table and returns whether it's valid.
---
---@param config table Configuration to validate
---@usage
---```lua
---local config = require("nvim-smart-keybind-search.config")
---local is_valid, error_msg = config.validate({
---  server_host = "localhost",
---  server_port = 8080,
---})
---if not is_valid then
---  print("Configuration error:", error_msg)
---end
---```
---
---@return boolean True if valid
---@return string? Error message if invalid
function M.validate(config)
	return validate_config(config)
end

return M
