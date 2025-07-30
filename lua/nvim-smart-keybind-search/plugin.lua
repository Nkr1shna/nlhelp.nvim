-- Plugin metadata and lazy loading configuration
-- This file is used by plugin managers for proper loading

-- Prevent loading if already loaded
if vim.g.loaded_nvim_smart_keybind_search then
	return
end

-- Set loaded flag
vim.g.loaded_nvim_smart_keybind_search = 1

-- Plugin information
local plugin_info = {
	name = "nvim-smart-keybind-search",
	version = "0.1.0",
	description = "Intelligent keybinding search using natural language",
	author = "nvim-smart-keybind-search",
	license = "MIT",
	dependencies = {
		"snack.nvim", -- Primary picker provider
	},
	optional_dependencies = {
		"telescope.nvim", -- Alternative picker provider
		"fzf-lua", -- Alternative picker provider
	},
}

-- Export plugin info for plugin managers
return plugin_info
