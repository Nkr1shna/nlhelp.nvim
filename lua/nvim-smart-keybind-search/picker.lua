-- Picker interface for nvim-smart-keybind-search
-- Integrates with snack.nvim and other picker providers

local M = {}

-- Import required modules
local rpc_client = require("nvim-smart-keybind-search.rpc_client")

-- Picker state
local picker_state = {
	current_picker = nil,
	debounce_timer = nil,
	last_query = "",
	search_results = {},
	is_searching = false,
	loading_timer = nil,
	query_suggestions = {},
}

-- Common query suggestions for auto-completion
local common_queries = {
	"delete word",
	"delete line", 
	"copy line",
	"paste",
	"save file",
	"quit",
	"jump to end",
	"jump to beginning",
	"find text",
	"replace text",
	"undo",
	"redo",
	"visual mode",
	"insert mode",
	"normal mode",
	"split window",
	"close window",
	"next buffer",
	"previous buffer",
	"search forward",
	"search backward",
}

--- Format a search result for display with enhanced customization
--- @param result table Search result from backend
--- @param config table Picker configuration
--- @param ui_config table UI configuration
--- @return table Formatted item for picker
local function format_result(result, config, ui_config)
	local keybinding = result.keybinding
	local icons = ui_config.icons or {}

	-- Determine icon based on keybinding type with enhanced logic
	local icon = icons.keybinding or "⌨️ "
	if keybinding.plugin and keybinding.plugin ~= "" then
		icon = icons.plugin or "🔌 "
	elseif keybinding.command and keybinding.command:match("^[gG]") then
		icon = icons.motion or "🔄 "
	elseif keybinding.command and keybinding.command:match("^:") then
		icon = icons.command or "⚡ "
	elseif keybinding.mode and keybinding.mode == "v" then
		icon = icons.visual or "👁️ "
	elseif keybinding.mode and keybinding.mode == "i" then
		icon = icons.insert or "✏️ "
	end

	-- Build display text with enhanced formatting
	local display_parts = {}
	
	-- Add relevance indicator if enabled
	if config.show_relevance and result.relevance then
		local relevance_icon = result.relevance > 0.8 and "⭐" or result.relevance > 0.6 and "✨" or "•"
		table.insert(display_parts, relevance_icon .. " ")
	end
	
	table.insert(display_parts, icon .. keybinding.keys)

	if keybinding.description and keybinding.description ~= "" then
		table.insert(display_parts, " - " .. keybinding.description)
	elseif keybinding.command and keybinding.command ~= "" then
		table.insert(display_parts, " - " .. keybinding.command)
	end

	-- Add plugin info if available
	if keybinding.plugin and keybinding.plugin ~= "" then
		table.insert(display_parts, " [" .. keybinding.plugin .. "]")
	end

	-- Add mode indicator if different from normal
	if keybinding.mode and keybinding.mode ~= "n" then
		table.insert(display_parts, " (" .. keybinding.mode .. ")")
	end

	-- Add explanation if enabled and available
	if config.show_explanations and result.explanation and result.explanation ~= "" then
		table.insert(display_parts, "\n  " .. result.explanation)
	end

	return {
		text = table.concat(display_parts),
		keybinding = keybinding,
		relevance = result.relevance,
		explanation = result.explanation,
		display_parts = display_parts, -- Store for detailed preview
	}
end

--- Show loading indicator
--- @param picker table Snack picker instance
local function show_loading_indicator(picker)
	if picker_state.loading_timer then
		vim.fn.timer_stop(picker_state.loading_timer)
	end
	
	-- Show loading message in picker
	if picker and picker.set_prompt then
		picker:set_prompt("Searching... ")
	end
	
	-- Set up loading animation
	local dots = {".", "..", "..."}
	local dot_index = 1
	
	picker_state.loading_timer = vim.fn.timer_start(500, function()
		if picker and picker.set_prompt then
			picker:set_prompt("Searching" .. dots[dot_index] .. " ")
			dot_index = (dot_index % #dots) + 1
		end
	end, { repeat = -1 })
end

--- Hide loading indicator
--- @param picker table Snack picker instance
local function hide_loading_indicator(picker)
	if picker_state.loading_timer then
		vim.fn.timer_stop(picker_state.loading_timer)
		picker_state.loading_timer = nil
	end
	
	if picker and picker.set_prompt then
		picker:set_prompt("Search keybindings: ")
	end
end

--- Get query suggestions based on input
--- @param query string Current query
--- @return table List of suggestions
local function get_query_suggestions(query)
	if #query < 2 then
		return {}
	end
	
	local suggestions = {}
	local lower_query = query:lower()
	
	for _, suggestion in ipairs(common_queries) do
		if suggestion:lower():find(lower_query, 1, true) then
			table.insert(suggestions, suggestion)
		end
	end
	
	-- Limit suggestions
	if #suggestions > 5 then
		suggestions = vim.list_slice(suggestions, 1, 5)
	end
	
	return suggestions
end

--- Perform search with enhanced debouncing and loading indicators
--- @param query string Search query
--- @param config table Picker configuration
--- @param ui_config table UI configuration
--- @param callback function Callback to update picker results
--- @param picker table Snack picker instance (optional)
local function debounced_search(query, config, ui_config, callback, picker)
	-- Clear existing timer
	if picker_state.debounce_timer then
		vim.fn.timer_stop(picker_state.debounce_timer)
		picker_state.debounce_timer = nil
	end

	-- Skip search if query is too short
	if #query < (config.min_query_length or 2) then
		hide_loading_indicator(picker)
		callback({})
		return
	end

	-- Skip search if query hasn't changed
	if query == picker_state.last_query then
		hide_loading_indicator(picker)
		callback(picker_state.search_results)
		return
	end

	-- Show loading indicator
	show_loading_indicator(picker)

	-- Set up debounced search
	picker_state.debounce_timer = vim.fn.timer_start(config.debounce_ms or 300, function()
		picker_state.debounce_timer = nil
		picker_state.last_query = query
		picker_state.is_searching = true

		-- Perform RPC query
		rpc_client.query(query, function(results, error_msg)
			picker_state.is_searching = false
			hide_loading_indicator(picker)

			if error_msg then
				-- Enhanced error display
				local error_lines = {
					"Search Error:",
					error_msg,
					"",
					"Try:",
					"- Check if backend is running",
					"- Verify your query",
					"- Check network connection"
				}
				
				-- Show error in picker if possible
				if picker and picker.set_prompt then
					picker:set_prompt("Error - Press Esc to close")
				end
				
				vim.notify("Search error: " .. error_msg, vim.log.levels.WARN)
				callback({})
				return
			end

			if not results or not results.results then
				callback({})
				return
			end

			-- Format results for picker with enhanced customization
			local formatted_results = {}
			local max_results = config.max_results or 20

			for i, result in ipairs(results.results) do
				if i > max_results then
					break
				end
				table.insert(formatted_results, format_result(result, config, ui_config))
			end

			picker_state.search_results = formatted_results
			callback(formatted_results)
		end)
	end)
end

--- Handle result selection with enhanced actions
--- @param item table Selected item
--- @param config table Picker configuration
--- @param picker table Snack picker instance
local function handle_selection(item, config, picker)
	if not item or not item.keybinding then
		return
	end

	local keybinding = item.keybinding

	-- Show action menu for complex keybindings
	if config.show_action_menu and (keybinding.command and keybinding.command:match("^:") or keybinding.plugin) then
		show_action_menu(item, config, picker)
		return
	end

	-- Try to execute the keybinding
	if keybinding.keys and keybinding.keys ~= "" then
		-- Close picker first
		if picker_state.current_picker then
			picker_state.current_picker = nil
		end

		-- Execute the keybinding
		vim.schedule(function()
			-- Convert keys to executable format
			local keys = keybinding.keys

			-- Handle special key sequences
			keys = keys:gsub("<leader>", vim.g.mapleader or "\\")
			keys = keys:gsub("<localleader>", vim.g.maplocalleader or "\\")

			-- Execute the keys
			vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(keys, true, false, true), "n", false)
		end)
	else
		vim.notify("No executable keybinding found", vim.log.levels.WARN)
	end
end

--- Show action menu for complex keybindings
--- @param item table Selected item
--- @param config table Picker configuration
--- @param picker table Snack picker instance
local function show_action_menu(item, config, picker)
	local keybinding = item.keybinding
	local actions = {
		{ text = "Execute", action = "execute" },
		{ text = "Copy to clipboard", action = "copy" },
		{ text = "Show details", action = "details" },
		{ text = "Explain", action = "explain" },
	}
	
	-- Create action picker
	local snack_ok, snack = pcall(require, "snack")
	if not snack_ok then
		-- Fallback to simple execution
		handle_selection(item, config, picker)
		return
	end
	
	local action_picker = snack.picker({
		prompt = "Choose action for: " .. keybinding.keys,
		source = {
			name = "action_menu",
			get = function(ctx, callback)
				callback(actions)
			end,
			format = function(action)
				return action.text
			end,
		},
		win = {
			width = 0.4,
			height = 0.3,
			border = "rounded",
		},
		actions = {
			["<CR>"] = function(action_picker, action_item)
				action_picker:close()
				execute_action(action_item.action, item, config, picker)
			end,
			["<Esc>"] = function(action_picker)
				action_picker:close()
			end,
		},
	})
	
	action_picker:open()
end

--- Execute selected action
--- @param action string Action to execute
--- @param item table Selected item
--- @param config table Picker configuration
--- @param picker table Snack picker instance
local function execute_action(action, item, config, picker)
	local keybinding = item.keybinding
	
	if action == "execute" then
		handle_selection(item, config, picker)
	elseif action == "copy" then
		-- Copy keybinding to clipboard
		vim.fn.setreg("+", keybinding.keys)
		vim.notify("Copied '" .. keybinding.keys .. "' to clipboard", vim.log.levels.INFO)
	elseif action == "details" then
		-- Show detailed information
		show_detailed_info(item, config)
	elseif action == "explain" then
		-- Show explanation
		if item.explanation then
			vim.notify("Explanation: " .. item.explanation, vim.log.levels.INFO)
		else
			vim.notify("No explanation available", vim.log.levels.WARN)
		end
	end
end

--- Show detailed information about a keybinding
--- @param item table Selected item
--- @param config table Picker configuration
local function show_detailed_info(item, config)
	local keybinding = item.keybinding
	local lines = {
		"=== Keybinding Details ===",
		"Keys: " .. (keybinding.keys or ""),
		"Mode: " .. (keybinding.mode or "normal"),
	}
	
	if keybinding.command and keybinding.command ~= "" then
		table.insert(lines, "Command: " .. keybinding.command)
	end
	
	if keybinding.description and keybinding.description ~= "" then
		table.insert(lines, "Description: " .. keybinding.description)
	end
	
	if keybinding.plugin and keybinding.plugin ~= "" then
		table.insert(lines, "Plugin: " .. keybinding.plugin)
	end
	
	if item.relevance then
		table.insert(lines, "Relevance: " .. string.format("%.2f", item.relevance))
	end
	
	if item.explanation and item.explanation ~= "" then
		table.insert(lines, "")
		table.insert(lines, "Explanation:")
		table.insert(lines, item.explanation)
	end
	
	-- Create a floating window to show details
	local width = math.min(80, vim.o.columns - 4)
	local height = math.min(20, #lines + 2)
	local row = math.floor((vim.o.lines - height) / 2)
	local col = math.floor((vim.o.columns - width) / 2)
	
	local buf = vim.api.nvim_create_buf(false, true)
	local win = vim.api.nvim_open_win(buf, true, {
		relative = "editor",
		width = width,
		height = height,
		row = row,
		col = col,
		style = "minimal",
		border = "rounded",
	})
	
	vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
	vim.api.nvim_buf_set_option(buf, "modifiable", false)
	vim.api.nvim_buf_set_option(buf, "buftype", "nofile")
	vim.api.nvim_buf_set_option(buf, "filetype", "text")
	
	-- Add close action
	vim.keymap.set("n", "q", function()
		vim.api.nvim_win_close(win, true)
	end, { buffer = buf, noremap = true })
	vim.keymap.set("n", "<Esc>", function()
		vim.api.nvim_win_close(win, true)
	end, { buffer = buf, noremap = true })
	
	vim.api.nvim_command("startinsert")
end

--- Create enhanced snack.nvim picker source
--- @param config table Picker configuration
--- @param ui_config table UI configuration
--- @param picker table Snack picker instance
--- @return table Snack picker source
local function create_snack_source(config, ui_config, picker)
	return {
		name = "keybinding_search",
		get = function(ctx, callback)
			local query = ctx.query or ""
			
			-- Get query suggestions
			picker_state.query_suggestions = get_query_suggestions(query)
			
			debounced_search(query, config, ui_config, callback, picker)
		end,
		format = function(item)
			return item.text
		end,
		preview = function(item)
			if not item or not item.keybinding then
				return nil
			end

			local lines = {}
			local kb = item.keybinding

			-- Enhanced preview with better formatting
			table.insert(lines, "=== Keybinding Preview ===")
			table.insert(lines, "")
			table.insert(lines, "Keys: " .. (kb.keys or ""))
			table.insert(lines, "Mode: " .. (kb.mode or "normal"))

			if kb.command and kb.command ~= "" then
				table.insert(lines, "Command: " .. kb.command)
			end

			if kb.description and kb.description ~= "" then
				table.insert(lines, "Description: " .. kb.description)
			end

			if kb.plugin and kb.plugin ~= "" then
				table.insert(lines, "Plugin: " .. kb.plugin)
			end

			-- Relevance score with visual indicator
			if item.relevance then
				local relevance_bar = ""
				local score = math.floor(item.relevance * 10)
				for i = 1, 10 do
					relevance_bar = relevance_bar .. (i <= score and "█" or "░")
				end
				table.insert(lines, "Relevance: " .. string.format("%.2f", item.relevance))
				table.insert(lines, "Match: " .. relevance_bar)
			end

			-- Explanation with better formatting
			if item.explanation and item.explanation ~= "" then
				table.insert(lines, "")
				table.insert(lines, "Explanation:")
				-- Wrap long explanations
				local wrapped = vim.fn.split(item.explanation, "\\n")
				for _, line in ipairs(wrapped) do
					table.insert(lines, "  " .. line)
				end
			end

			return {
				lines = lines,
				filetype = "text",
			}
		end,
	}
end

--- Open the enhanced keybinding search picker
--- @param initial_query string|nil Initial query to populate
--- @param picker_config table Picker configuration
--- @param ui_config table UI configuration (optional)
function M.open(initial_query, picker_config, ui_config)
	-- Check if snack.nvim is available
	local snack_ok, snack = pcall(require, "snack")
	if not snack_ok or not snack.picker then
		vim.notify("snack.nvim picker not available. Please install snack.nvim.", vim.log.levels.ERROR)
		return
	end

	-- Check if RPC client is connected
	if not rpc_client.is_connected() then
		vim.notify("Backend not connected. Please check your configuration.", vim.log.levels.ERROR)
		return
	end

	-- Reset picker state
	picker_state.last_query = ""
	picker_state.search_results = {}
	picker_state.is_searching = false
	picker_state.query_suggestions = {}

	-- Create enhanced picker configuration
	local snack_picker_config = vim.tbl_deep_extend("force", {
		prompt = "Search keybindings: ",
		source = create_snack_source(picker_config, ui_config or {}),
		win = {
			width = picker_config.window.width or 0.8,
			height = picker_config.window.height or 0.6,
			border = picker_config.window.border or "rounded",
		},
		preview = {
			enabled = true,
		},
		actions = {
			["<CR>"] = function(picker, item)
				handle_selection(item, picker_config, picker)
				picker:close()
			end,
			["<C-c>"] = function(picker)
				picker:close()
			end,
			["<Esc>"] = function(picker)
				picker:close()
			end,
			-- Enhanced keyboard shortcuts
			["<Tab>"] = function(picker)
				-- Auto-complete with first suggestion
				local suggestions = picker_state.query_suggestions
				if #suggestions > 0 then
					picker:set_query(suggestions[1])
				end
			end,
			["<C-d>"] = function(picker, item)
				-- Show details without closing picker
				if item then
					show_detailed_info(item, picker_config)
				end
			end,
			["<C-y>"] = function(picker, item)
				-- Copy keybinding to clipboard
				if item and item.keybinding then
					vim.fn.setreg("+", item.keybinding.keys)
					vim.notify("Copied '" .. item.keybinding.keys .. "' to clipboard", vim.log.levels.INFO)
				end
			end,
		},
	}, picker_config.snack_config or {})

	-- Set initial query if provided
	if initial_query and initial_query ~= "" then
		snack_picker_config.query = initial_query
	end

	-- Open the picker
	local picker = snack.picker(snack_picker_config)
	picker_state.current_picker = picker

	-- Set up cleanup on picker close
	picker:on("close", function()
		picker_state.current_picker = nil
		if picker_state.debounce_timer then
			vim.fn.timer_stop(picker_state.debounce_timer)
			picker_state.debounce_timer = nil
		end
		if picker_state.loading_timer then
			vim.fn.timer_stop(picker_state.loading_timer)
			picker_state.loading_timer = nil
		end
	end)

	picker:open()
end

--- Check if picker is currently open
--- @return boolean True if picker is open
function M.is_open()
	return picker_state.current_picker ~= nil
end

--- Close the current picker if open
function M.close()
	if picker_state.current_picker then
		picker_state.current_picker:close()
	end
end

--- Get picker statistics
--- @return table Picker statistics
function M.get_stats()
	return {
		is_open = M.is_open(),
		last_query = picker_state.last_query,
		results_count = #picker_state.search_results,
		is_searching = picker_state.is_searching,
		has_debounce_timer = picker_state.debounce_timer ~= nil,
		has_loading_timer = picker_state.loading_timer ~= nil,
		query_suggestions_count = #picker_state.query_suggestions,
	}
end

return M
