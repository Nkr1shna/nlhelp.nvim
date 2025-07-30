-- Utility functions for nvim-smart-keybind-search
-- Common helper functions used across the plugin

local M = {}

--- Log a message with the plugin prefix
--- @param message string Message to log
--- @param level number Log level (vim.log.levels)
function M.log(message, level)
	level = level or vim.log.levels.INFO
	vim.notify("[nvim-smart-keybind-search] " .. message, level)
end

--- Debug log (only shown if debug is enabled)
--- @param message string Debug message
--- @param config table Plugin configuration
function M.debug(message, config)
	if config and config.logging and config.logging.debug then
		M.log("[DEBUG] " .. message, vim.log.levels.DEBUG)
	end
end

--- Check if a table is empty
--- @param t table Table to check
--- @return boolean True if table is empty
function M.is_empty(t)
	return next(t) == nil
end

--- Get the length of a table (works for both arrays and maps)
--- @param t table Table to measure
--- @return number Length of table
function M.table_length(t)
	local count = 0
	for _ in pairs(t) do
		count = count + 1
	end
	return count
end

--- Escape special characters in a string for pattern matching
--- @param str string String to escape
--- @return string Escaped string
function M.escape_pattern(str)
	return str:gsub("[%^%$%(%)%%%.%[%]%*%+%-%?]", "%%%1")
end

--- Split a string by delimiter
--- @param str string String to split
--- @param delimiter string Delimiter to split by
--- @return table Array of split parts
function M.split(str, delimiter)
	local result = {}
	local pattern = "([^" .. M.escape_pattern(delimiter) .. "]+)"

	for match in str:gmatch(pattern) do
		table.insert(result, match)
	end

	return result
end

--- Trim whitespace from both ends of a string
--- @param str string String to trim
--- @return string Trimmed string
function M.trim(str)
	return str:match("^%s*(.-)%s*$")
end

--- Check if a string starts with a prefix
--- @param str string String to check
--- @param prefix string Prefix to look for
--- @return boolean True if string starts with prefix
function M.starts_with(str, prefix)
	return str:sub(1, #prefix) == prefix
end

--- Check if a string ends with a suffix
--- @param str string String to check
--- @param suffix string Suffix to look for
--- @return boolean True if string ends with suffix
function M.ends_with(str, suffix)
	return str:sub(-#suffix) == suffix
end

--- Convert a keymap mode character to full name
--- @param mode string Mode character (n, i, v, etc.)
--- @return string Full mode name
function M.mode_to_name(mode)
	local mode_names = {
		n = "normal",
		i = "insert",
		v = "visual",
		x = "visual-block",
		s = "select",
		o = "operator-pending",
		c = "command",
		t = "terminal",
		[""] = "visual-block", -- Ctrl-V visual mode
	}

	return mode_names[mode] or mode
end

--- Generate a unique ID for a keybinding
--- @param keys string Key sequence
--- @param mode string Mode
--- @param buffer number|nil Buffer number (nil for global)
--- @return string Unique ID
function M.generate_keybind_id(keys, mode, buffer)
	local id_parts = { mode, keys }

	if buffer then
		table.insert(id_parts, "buf:" .. buffer)
	end

	return table.concat(id_parts, ":")
end

--- Debounce a function call
--- @param func function Function to debounce
--- @param delay number Delay in milliseconds
--- @return function Debounced function
function M.debounce(func, delay)
	local timer = nil

	return function(...)
		local args = { ... }

		if timer then
			timer:stop()
		end

		timer = vim.defer_fn(function()
			func(unpack(args))
			timer = nil
		end, delay)
	end
end

--- Create a simple cache with TTL
--- @param ttl_ms number Time to live in milliseconds
--- @return table Cache object with get/set/clear methods
function M.create_cache(ttl_ms)
	local cache = {}
	local timestamps = {}

	return {
		get = function(key)
			local now = vim.loop.now()
			local timestamp = timestamps[key]

			if timestamp and (now - timestamp) < ttl_ms then
				return cache[key]
			else
				-- Expired or doesn't exist
				cache[key] = nil
				timestamps[key] = nil
				return nil
			end
		end,

		set = function(key, value)
			cache[key] = value
			timestamps[key] = vim.loop.now()
		end,

		clear = function()
			cache = {}
			timestamps = {}
		end,
	}
end

return M
