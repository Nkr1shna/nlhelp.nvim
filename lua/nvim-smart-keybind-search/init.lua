---@meta

---@class nvim-smart-keybind-search
---@field _initialized boolean
---@field _config table
---@field _change_detection_timer number|nil
local M = {}

---@brief nvim-smart-keybind-search
---@description Intelligent keybinding search plugin using RAG-based search
---@tag nvim-smart-keybind-search
---@tag nvim-smart-keybind-search-introduction
---
---nvim-smart-keybind-search is a Neovim plugin that provides intelligent,
---natural language search for keybindings and vim motions using
---Retrieval-Augmented Generation (RAG) technology.
---
---Features:
---• Natural language search for keybindings
---• RAG-powered intelligent results
---• Comprehensive vim knowledge base
---• Real-time custom keybinding sync
---• Health monitoring and diagnostics
---• Fast sub-2-second response times
---
---@usage
---```lua
---require("nvim-smart-keybind-search").setup({
---  backend = {
---    auto_start = true,
---    timeout = 5000,
---  },
---})
---```
---
---@usage
---```vim
---" Search for keybindings
---:SmartKeybindSearch "delete current line"
---
---" Sync custom keybindings
---:SmartKeybindSync
---
---" Check health status
---:SmartKeybindHealth
---```

-- Import plugin modules
local config = require("nvim-smart-keybind-search.config")
local picker = require("nvim-smart-keybind-search.picker")
local rpc_client = require("nvim-smart-keybind-search.rpc_client")
local keybind_scanner = require("nvim-smart-keybind-search.keybind_scanner")

-- Plugin state
M._initialized = false
M._config = nil
M._change_detection_timer = nil

---Setup the plugin with user configuration
---@tag nvim-smart-keybind-search-setup
---
---Initializes the plugin with the provided configuration options.
---This function should be called before using any plugin features.
---
---@param opts? table User configuration options
---@field opts.backend? table Backend configuration
---@field opts.backend.binary_path? string Path to backend binary (auto-detected if nil)
---@field opts.backend.auto_start? boolean Auto-start backend on setup (default: true)
---@field opts.backend.timeout? number Backend timeout in milliseconds (default: 5000)
---@field opts.auto_sync? boolean Auto-sync keybindings on setup (default: true)
---@field opts.watch_changes? boolean Watch for keybinding changes (default: true)
---@field opts.keymaps? table Keymap configuration
---@field opts.keymaps.search? string Keymap for search command (default: "<leader>sk")
---@field opts.picker? table Picker configuration
---@field opts.ui? table UI configuration
---@field opts.scanner? table Scanner configuration
---@field opts.logging? table Logging configuration
---
---@usage
---```lua
---require("nvim-smart-keybind-search").setup({
---  backend = {
---    auto_start = true,
---    timeout = 5000,
---  },
---  keymaps = {
---    search = "<leader>sk",
---  },
---  picker = {
---    max_results = 20,
---    show_explanations = true,
---  },
---  ui = {
---    show_explanations = true,
---  },
---  logging = {
---    level = "info",
---    debug = false,
---  },
---})
---```
---
---@return nil
function M.setup(opts)
	-- Merge user config with defaults
	M._config = config.setup(opts or {})

	-- Initialize RPC client
	rpc_client.setup(M._config)

	-- Create user commands
	vim.api.nvim_create_user_command("SmartKeybindSearch", function(cmd_opts)
		M.search_keybindings(cmd_opts.args)
	end, {
		nargs = "?",
		desc = "Search for keybindings using natural language",
	})

	vim.api.nvim_create_user_command("SmartKeybindSync", function()
		M.force_sync()
	end, {
		desc = "Force sync all keybindings with backend",
	})

	vim.api.nvim_create_user_command("SmartKeybindStats", function()
		local stats = M.get_scanner_stats()
		if stats then
			print("=== Keybinding Scanner Statistics ===")
			print("Cached keybindings: " .. stats.cached_count)
			print("Last scan hash: " .. (stats.last_scan_hash or "none"))
			print("Change detection: " .. (stats.change_detection_enabled and "enabled" or "disabled"))
		else
			print("Scanner not initialized")
		end
	end, {
		desc = "Show keybinding scanner statistics",
	})

	-- Set up keymapping if configured
	if M._config.keymaps.search then
		vim.keymap.set("n", M._config.keymaps.search, function()
			M.search_keybindings()
		end, { desc = "Smart keybinding search" })
	end

	-- Initialize keybinding scanner if auto-sync is enabled
	if M._config.auto_sync then
		keybind_scanner.setup(M._config.scanner)

		-- Start background database population
		M._populate_database_background()

		-- Set up change detection if enabled
		if M._config.watch_changes then
			M._setup_change_detection()
		end
	end

	M._initialized = true
end

---Search for keybindings using natural language query
---@tag nvim-smart-keybind-search-search
---
---Opens the search interface to find keybindings using natural language.
---The query can describe what you want to do in plain English.
---
---@param query? string Optional initial query to pre-fill the search
---
---@usage
---```lua
----- Search with initial query
---require("nvim-smart-keybind-search").search_keybindings("delete current line")
---
----- Open search interface without initial query
---require("nvim-smart-keybind-search").search_keybindings()
---```
---
---@usage
---```vim
---" Search with query
---:SmartKeybindSearch "delete current line"
---
---" Open search interface
---:SmartKeybindSearch
---```
---
---@return nil
function M.search_keybindings(query)
	if not M._initialized then
		vim.notify(
			'nvim-smart-keybind-search not initialized. Run :lua require("nvim-smart-keybind-search").setup()',
			vim.log.levels.ERROR
		)
		return
	end

	picker.open(query, M._config.picker, M._config.ui)
end

---Sync all keybindings with the backend
---@tag nvim-smart-keybind-search-sync
---
---Scans all current keybindings and syncs them with the backend server.
---This includes both built-in and custom keybindings.
---
---@usage
---```lua
----- Force sync all keybindings
---require("nvim-smart-keybind-search").sync_keybindings()
---```
---
---@usage
---```vim
---" Sync keybindings
---:SmartKeybindSync
---```
---
---@return nil
function M.sync_keybindings()
	if not M._initialized then
		vim.notify("Plugin not initialized", vim.log.levels.WARN)
		return
	end

	local keybindings = keybind_scanner.scan_all()
	rpc_client.sync_keybindings(keybindings, function(success, error_msg)
		if success then
			vim.notify("Keybindings synced successfully", vim.log.levels.INFO)
		else
			vim.notify("Failed to sync keybindings: " .. (error_msg or "Unknown error"), vim.log.levels.ERROR)
		end
	end)
end

--- Update specific keybindings (for incremental updates)
--- @param changed_keybindings table List of changed keybindings
function M.update_keybindings(changed_keybindings)
	if not M._initialized then
		return
	end

	rpc_client.update_keybindings(changed_keybindings, function(success, error_msg)
		if not success then
			vim.notify("Failed to update keybindings: " .. (error_msg or "Unknown error"), vim.log.levels.WARN)
		end
	end)
end

---Check plugin and backend health status
---@tag nvim-smart-keybind-search-health
---
---Performs a comprehensive health check of the plugin and backend services.
---Returns detailed status information about all components.
---
---@usage
---```lua
----- Get health status
---local health = require("nvim-smart-keybind-search").health_check()
---print("Plugin status:", health.plugin.status)
---print("Backend status:", health.backend.status)
---```
---
---@usage
---```vim
---" Check health status
---:SmartKeybindHealth
---```
---
---@return table Health status information
---@field return.plugin table Plugin health information
---@field return.plugin.status string Plugin status ("healthy", "unhealthy", "unknown")
---@field return.plugin.initialized boolean Whether plugin is initialized
---@field return.plugin.config_valid boolean Whether configuration is valid
---@field return.backend table Backend health information
---@field return.backend.status string Backend status ("healthy", "unhealthy", "unknown")
---@field return.backend.chromadb_available boolean Whether ChromaDB is available
---@field return.backend.ollama_available boolean Whether Ollama is available
---@field return.backend.model_loaded boolean Whether LLM model is loaded
function M.health_check()
	local health = {}

	-- Check plugin initialization
	health.plugin_initialized = M._initialized
	health.config_valid = M._config ~= nil

	-- Check backend health
	rpc_client.health_check(function(backend_health)
		health.backend = backend_health

		-- Display health status
		print("=== nvim-smart-keybind-search Health Check ===")
		print("Plugin initialized: " .. tostring(health.plugin_initialized))
		print("Config valid: " .. tostring(health.config_valid))

		if health.backend then
			print("Backend status: " .. (health.backend.status or "unknown"))
			print("ChromaDB: " .. tostring(health.backend.chromadb_available or false))
			print("Ollama: " .. tostring(health.backend.ollama_available or false))
			print("Model loaded: " .. tostring(health.backend.model_loaded or false))
		else
			print("Backend: Not available")
		end
	end)

	return health
end

---Get current plugin configuration
---@tag nvim-smart-keybind-search-get-config
---
---Returns the current plugin configuration that was set during setup.
---
---@usage
---```lua
---local config = require("nvim-smart-keybind-search").get_config()
---print("Server host:", config.server_host)
---print("Server port:", config.server_port)
---```
---
---@return table Current configuration
function M.get_config()
	return M._config
end

---Check if plugin is initialized
---@tag nvim-smart-keybind-search-is-initialized
---
---Returns whether the plugin has been properly initialized.
---
---@usage
---```lua
---local is_init = require("nvim-smart-keybind-search").is_initialized()
---if is_init then
---  print("Plugin is initialized")
---else
---  print("Plugin is not initialized")
---end
---```
---
---@return boolean True if initialized
function M.is_initialized()
	return M._initialized
end

--- Setup automatic change detection for keybindings
function M._setup_change_detection()
	-- Create autocmd group for keybinding change detection
	local group = vim.api.nvim_create_augroup("SmartKeybindSearch", { clear = true })

	-- Watch for events that might change keybindings
	vim.api.nvim_create_autocmd({
		"VimEnter", -- After vim starts and plugins load
		"BufEnter", -- When entering buffers (for buffer-local mappings)
		"FileType", -- When filetype changes (filetype-specific mappings)
		"SourcePre", -- Before sourcing files (config changes)
	}, {
		group = group,
		callback = function()
			-- Debounce the change detection to avoid excessive calls
			if M._change_detection_timer then
				vim.fn.timer_stop(M._change_detection_timer)
			end

			M._change_detection_timer = vim.fn.timer_start(1000, function()
				M._change_detection_timer = nil
				M._check_for_changes()
			end)
		end,
	})
end

--- Check for keybinding changes and update backend if needed
function M._check_for_changes()
	if not M._initialized or not M._config.watch_changes then
		return
	end

	local changes = keybind_scanner.detect_changes()
	if changes then
		-- Update backend with changed keybindings
		M.update_keybindings(changes)
	end
end

---Force a manual sync of all keybindings
---@tag nvim-smart-keybind-search-force-sync
---
---Forces a complete sync of all keybindings, bypassing change detection.
---Useful when you want to ensure all keybindings are up to date.
---
---@usage
---```lua
----- Force sync all keybindings
---require("nvim-smart-keybind-search").force_sync()
---```
---
---@usage
---```vim
---" Force sync keybindings
---:SmartKeybindSync
---```
---
---@return nil
function M.force_sync()
	if not M._initialized then
		vim.notify("Plugin not initialized", vim.log.levels.WARN)
		return
	end

	-- Force a fresh scan
	local keybindings = keybind_scanner.scan_all()
	rpc_client.sync_keybindings(keybindings, function(success, error_msg)
		if success then
			vim.notify(
				"Keybindings force-synced successfully (" .. #keybindings .. " keybindings)",
				vim.log.levels.INFO
			)
		else
			vim.notify("Failed to force-sync keybindings: " .. (error_msg or "Unknown error"), vim.log.levels.ERROR)
		end
	end)
end

---Get scanner statistics
---@tag nvim-smart-keybind-search-get-scanner-stats
---
---Returns statistics about the keybinding scanner.
---
---@usage
---```lua
---local stats = require("nvim-smart-keybind-search").get_scanner_stats()
---if stats then
---  print("Cached keybindings:", stats.cached_count)
---  print("Last scan hash:", stats.last_scan_hash)
---end
---```
---
---@return table? Scanner statistics or nil if not initialized
---@field return.cached_count number Number of cached keybindings
---@field return.last_scan_hash string Hash of last scan
---@field return.change_detection_enabled boolean Whether change detection is enabled
function M.get_scanner_stats()
	if not M._initialized then
		return nil
	end

	return keybind_scanner.get_stats()
end

--- Populate database in background during setup
function M._populate_database_background()
	vim.notify("nvim-smart-keybind-search: Initializing database in background...", vim.log.levels.INFO)

	-- Run database population asynchronously
	vim.schedule(function()
		local data_dir = vim.fn.stdpath("data") .. "/nvim-smart-keybind-search"

		-- Ensure data directory exists
		vim.fn.mkdir(data_dir, "p")
		vim.fn.mkdir(data_dir .. "/chroma", "p")
		vim.fn.mkdir(data_dir .. "/logs", "p")

		-- Get project root directory
		local script_dir = vim.fn.fnamemodify(debug.getinfo(1).source:sub(2), ":h:h:h")

		-- Build the Go server first
		local build_cmd = string.format("cd %s && go build -ldflags='-s -w' -o server cmd/server/main.go", script_dir)
		vim.fn.system(build_cmd)

		-- Population steps
		local steps = {
			{
				name = "Built-in knowledge (Neovim quick reference)",
				cmd = string.format("cd %s && go run scripts/populate-builtin/main.go", script_dir),
			},
			{
				name = "General knowledge (HuggingFace dataset)",
				cmd = string.format("cd %s && go run scripts/populate-general/main.go", script_dir),
			},
			{
				name = "User keybindings",
				cmd = string.format("cd %s && go run scripts/populate-user/main.go", script_dir),
			},
		}

		local completed_steps = 0
		local total_steps = #steps

		for i, step in ipairs(steps) do
			vim.notify(string.format("Populating %s... (%d/%d)", step.name, i, total_steps), vim.log.levels.INFO)

			local success = false
			if step.cmd then
				local result = vim.fn.system(step.cmd)
				success = vim.v.shell_error == 0
				if not success then
					vim.notify(string.format("Failed to populate %s: %s", step.name, result), vim.log.levels.WARN)
				end
			elseif step.func then
				success = step.func()
			end

			if success then
				completed_steps = completed_steps + 1
			end
		end

		-- Final notification
		if completed_steps == total_steps then
			vim.notify(
				"🎉 nvim-smart-keybind-search is ready! Use <leader>ks to start searching.",
				vim.log.levels.INFO
			)
		else
			vim.notify(
				string.format(
					"Database initialization completed with %d/%d steps successful. Plugin is ready to use.",
					completed_steps,
					total_steps
				),
				vim.log.levels.WARN
			)
		end

		-- Perform initial keybinding sync after database is ready
		M.sync_keybindings()
	end)
end

return M
