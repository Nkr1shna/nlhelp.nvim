-- Health check module for nvim-smart-keybind-search
-- Integrates with neovim's :checkhealth system

local M = {}

--- Check if a command is available
--- @param cmd string Command to check
--- @return boolean True if command is available
local function has_command(cmd)
	return vim.fn.executable(cmd) == 1
end

--- Check if a Lua module is available
--- @param module string Module name to check
--- @return boolean True if module is available
local function has_module(module)
	local ok, _ = pcall(require, module)
	return ok
end

--- Main health check function
function M.check()
	vim.health.start("nvim-smart-keybind-search")

	-- Check plugin initialization
	local plugin = require("nvim-smart-keybind-search")
	if plugin.is_initialized() then
		vim.health.ok("Plugin is initialized")
	else
		vim.health.warn('Plugin not initialized. Run require("nvim-smart-keybind-search").setup()')
	end

	-- Check configuration
	local config = plugin.get_config()
	if config then
		vim.health.ok("Configuration loaded")

		-- Check backend binary
		if config.backend.binary_path then
			if has_command(config.backend.binary_path) then
				vim.health.ok("Backend binary found: " .. config.backend.binary_path)
			else
				vim.health.error("Backend binary not found: " .. config.backend.binary_path)
			end
		else
			vim.health.warn("Backend binary path not configured")
		end
	else
		vim.health.error("Configuration not loaded")
	end

	-- Check dependencies
	vim.health.start("Dependencies")

	-- Check primary picker provider
	if has_module("snack") then
		vim.health.ok("snack.nvim is available")
	else
		vim.health.warn("snack.nvim not found (primary picker provider)")
	end

	-- Check alternative picker providers
	if has_module("telescope") then
		vim.health.ok("telescope.nvim is available (alternative picker)")
	end

	if has_module("fzf-lua") then
		vim.health.ok("fzf-lua is available (alternative picker)")
	end

	-- Check external dependencies
	vim.health.start("External Dependencies")

	-- Check for Go (needed for backend)
	if has_command("go") then
		vim.health.ok("Go is installed")
	else
		vim.health.warn("Go not found (needed for building backend)")
	end

	-- Check for curl (needed for model downloads)
	if has_command("curl") then
		vim.health.ok("curl is available")
	else
		vim.health.error("curl not found (needed for downloading models)")
	end

	-- Perform backend health check if plugin is initialized
	if plugin.is_initialized() then
		vim.health.start("Backend Health")
		plugin.health_check()
	end
end

return M
