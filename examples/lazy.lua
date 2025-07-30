-- lazy.nvim configuration for nvim-smart-keybind-search
-- Add this to your lazy.nvim plugin configuration

return {
	"your-username/nvim-smart-keybind-search",
	dependencies = {
		"snacks.nvim", -- Required for picker interface
	},
	config = function()
		require("nvim-smart-keybind-search").setup({
			-- Server configuration
			server_host = "localhost",
			server_port = 8080,
			auto_start = true,

			-- Database configuration
			auto_sync = true, -- Automatically sync keybindings on startup
			watch_changes = true, -- Watch for keybinding changes

			-- Keymaps
			keymaps = {
				search = "<leader>ks", -- Search keybindings
				health = "<leader>kh", -- Health check
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
	end,

	-- Lazy load on command or keymap
	cmd = {
		"SmartKeybindSearch",
		"SmartKeybindSync",
		"SmartKeybindHealth",
		"SmartKeybindStats",
	},

	keys = {
		{ "<leader>ks", "<cmd>SmartKeybindSearch<cr>", desc = "Smart keybinding search" },
		{ "<leader>kh", "<cmd>SmartKeybindHealth<cr>", desc = "Smart keybinding health check" },
	},
}
