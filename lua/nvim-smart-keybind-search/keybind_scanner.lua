-- Keybinding scanner for extracting neovim keybindings
-- Scans and extracts all configured keybindings with metadata

local M = {}

-- Scanner state
local scanner_state = {
	config = nil,
	last_scan_hash = nil,
	keybinding_cache = {},
	change_detection_enabled = true,
}

--- Generate a unique ID for a keybinding
--- @param mode string Vim mode
--- @param lhs string Left-hand side (key sequence)
--- @param plugin string|nil Plugin name
--- @return string Unique keybinding ID
local function generate_keybinding_id(mode, lhs, plugin)
	local base = mode .. ":" .. lhs
	if plugin and plugin ~= "" then
		base = base .. ":" .. plugin
	end
	-- Simple hash function for ID generation
	local hash = 0
	for i = 1, #base do
		hash = (hash * 31 + string.byte(base, i)) % 2147483647
	end
	return tostring(hash)
end

--- Extract plugin name from mapping source
--- @param source string|number|nil Mapping source information
--- @return string|nil Plugin name or nil
local function extract_plugin_name(source)
	if not source then
		return nil
	end

	-- Convert to string if it's a number
	local source_str = tostring(source)
	if source_str == "" then
		return nil
	end

	-- Extract plugin name from various source formats
	-- Format: "~/.local/share/nvim/lazy/plugin-name/..."
	local plugin_match = source_str:match("/([^/]+)/")
	if plugin_match and plugin_match ~= "nvim" and plugin_match ~= "lua" then
		return plugin_match
	end

	-- Format: "plugin-name: ..."
	plugin_match = source_str:match("^([^:]+):")
	if plugin_match then
		return plugin_match
	end

	return nil
end

--- Check if keybinding should be excluded based on patterns
--- @param lhs string Left-hand side key sequence
--- @param exclude_patterns table List of exclusion patterns
--- @return boolean True if should be excluded
local function should_exclude_keybinding(lhs, exclude_patterns)
	if not exclude_patterns then
		return false
	end

	for _, pattern in ipairs(exclude_patterns) do
		if lhs:match(pattern) then
			return true
		end
	end

	return false
end

--- Get description for a keybinding from various sources
--- @param mapping table Vim mapping information
--- @param custom_providers table Custom description providers
--- @return string Description
local function get_keybinding_description(mapping, custom_providers)
	-- Try custom description providers first
	if custom_providers then
		for _, provider in ipairs(custom_providers) do
			if type(provider) == "function" then
				local desc = provider(mapping)
				if desc and desc ~= "" then
					return desc
				end
			end
		end
	end

	-- Use built-in description if available
	if mapping.desc and mapping.desc ~= "" then
		return mapping.desc
	end

	-- Use rhs as fallback description
	if mapping.rhs and mapping.rhs ~= "" then
		-- Clean up the rhs for better readability
		local cleaned_rhs = mapping.rhs:gsub("<[^>]+>", function(match)
			-- Convert key codes to readable format
			local key_map = {
				["<CR>"] = "Enter",
				["<ESC>"] = "Escape",
				["<TAB>"] = "Tab",
				["<SPACE>"] = "Space",
				["<BS>"] = "Backspace",
				["<DEL>"] = "Delete",
			}
			return key_map[match:upper()] or match
		end)
		return "Execute: " .. cleaned_rhs
	end

	return "Custom keybinding"
end

--- Scan keybindings for a specific mode
--- @param mode string Vim mode ('n', 'i', 'v', etc.)
--- @param config table Scanner configuration
--- @return table List of keybindings for the mode
local function scan_mode_keybindings(mode, config)
	local keybindings = {}

	-- Get all mappings for the mode
	local mappings = vim.api.nvim_get_keymap(mode)

	-- Also get buffer-local mappings if enabled
	if config.include_buffer_local then
		local buf_mappings = vim.api.nvim_buf_get_keymap(0, mode)
		for _, mapping in ipairs(buf_mappings) do
			mapping.buffer_local = true
			table.insert(mappings, mapping)
		end
	end

	for _, mapping in ipairs(mappings) do
		local lhs = mapping.lhs or ""

		-- Skip if should be excluded
		if should_exclude_keybinding(lhs, config.exclude_patterns) then
			goto continue
		end

		-- Extract plugin information
		local plugin = nil
		if config.include_plugins then
			plugin = extract_plugin_name(mapping.script)
		end

		-- Generate keybinding data
		local keybinding = {
			id = generate_keybinding_id(mode, lhs, plugin),
			keys = lhs,
			command = mapping.rhs or "",
			description = get_keybinding_description(mapping, config.description_providers),
			mode = mode,
			plugin = plugin,
			metadata = {
				buffer_local = mapping.buffer_local and "true" or "false",
				silent = mapping.silent and "true" or "false",
				noremap = mapping.noremap and "true" or "false",
				nowait = mapping.nowait and "true" or "false",
				expr = mapping.expr and "true" or "false",
				script = mapping.script or "",
			},
		}

		table.insert(keybindings, keybinding)

		::continue::
	end

	return keybindings
end

--- Calculate hash of keybindings for change detection
--- @param keybindings table List of keybindings
--- @return string Hash string
local function calculate_keybindings_hash(keybindings)
	-- Create a deterministic string representation
	local hash_parts = {}

	-- Sort keybindings by ID for consistent hashing
	table.sort(keybindings, function(a, b)
		return a.id < b.id
	end)

	for _, kb in ipairs(keybindings) do
		local kb_str = string.format(
			"%s|%s|%s|%s|%s|%s",
			kb.id or "",
			kb.keys or "",
			kb.command or "",
			kb.description or "",
			kb.mode or "",
			kb.plugin or ""
		)
		table.insert(hash_parts, kb_str)
	end

	local combined = table.concat(hash_parts, "\n")

	-- Simple hash function
	local hash = 0
	for i = 1, #combined do
		hash = (hash * 31 + string.byte(combined, i)) % 2147483647
	end

	return tostring(hash)
end

--- Setup the keybinding scanner with configuration
--- @param scanner_config table Scanner configuration
function M.setup(scanner_config)
	scanner_state.config = scanner_config or {}

	-- Set default values
	if scanner_state.config.include_plugins == nil then
		scanner_state.config.include_plugins = true
	end
	if scanner_state.config.include_buffer_local == nil then
		scanner_state.config.include_buffer_local = true
	end
	if not scanner_state.config.exclude_patterns then
		scanner_state.config.exclude_patterns = {
			"^<Plug>", -- Exclude internal plugin mappings
			"^<SNR>", -- Exclude script-local mappings
		}
	end
	if not scanner_state.config.description_providers then
		scanner_state.config.description_providers = {}
	end

	scanner_state.change_detection_enabled = true
end

--- Scan all configured keybindings
--- @return table List of keybindings with metadata
function M.scan_all()
	if not scanner_state.config then
		error("Keybinding scanner not initialized. Call setup() first.")
	end

	local all_keybindings = {}

	-- Define modes to scan
	local modes = { "n", "i", "v", "x", "s", "o", "c", "t" }

	for _, mode in ipairs(modes) do
		local mode_keybindings = scan_mode_keybindings(mode, scanner_state.config)
		for _, kb in ipairs(mode_keybindings) do
			table.insert(all_keybindings, kb)
		end
	end

	-- Update cache and hash for change detection
	scanner_state.keybinding_cache = all_keybindings
	scanner_state.last_scan_hash = calculate_keybindings_hash(all_keybindings)

	return all_keybindings
end

--- Detect changes in keybindings since last scan
--- @return table List of changed keybindings, or nil if no changes
function M.detect_changes()
	if not scanner_state.change_detection_enabled then
		return nil
	end

	-- Perform a new scan
	local current_keybindings = M.scan_all()
	local current_hash = calculate_keybindings_hash(current_keybindings)

	-- Compare with last scan
	if scanner_state.last_scan_hash == current_hash then
		return nil -- No changes detected
	end

	-- Changes detected, return all current keybindings
	-- In a more sophisticated implementation, we could return only the diff
	return current_keybindings
end

--- Get cached keybindings without rescanning
--- @return table Cached keybindings
function M.get_cached()
	return scanner_state.keybinding_cache or {}
end

--- Enable or disable change detection
--- @param enabled boolean Whether to enable change detection
function M.set_change_detection(enabled)
	scanner_state.change_detection_enabled = enabled
end

--- Get scanner statistics
--- @return table Scanner statistics
function M.get_stats()
	return {
		cached_count = #(scanner_state.keybinding_cache or {}),
		last_scan_hash = scanner_state.last_scan_hash,
		change_detection_enabled = scanner_state.change_detection_enabled,
		config = scanner_state.config,
	}
end

--- Serialize keybindings for JSON-RPC transmission
--- @param keybindings table List of keybindings
--- @return string JSON string
function M.serialize_keybindings(keybindings)
	-- Ensure all required fields are present and properly typed
	local serializable = {}

	for _, kb in ipairs(keybindings or {}) do
		local serialized_kb = {
			id = tostring(kb.id or ""),
			keys = tostring(kb.keys or ""),
			command = tostring(kb.command or ""),
			description = tostring(kb.description or ""),
			mode = tostring(kb.mode or ""),
			plugin = kb.plugin and tostring(kb.plugin) or nil,
			metadata = {},
		}

		-- Ensure metadata is properly serialized
		if kb.metadata then
			for key, value in pairs(kb.metadata) do
				serialized_kb.metadata[tostring(key)] = tostring(value)
			end
		end

		table.insert(serializable, serialized_kb)
	end

	return vim.json.encode(serializable)
end

--- Deserialize keybindings from JSON string
--- @param json_string string JSON string
--- @return table List of keybindings
function M.deserialize_keybindings(json_string)
	local ok, keybindings = pcall(vim.json.decode, json_string)
	if not ok then
		error("Failed to deserialize keybindings: " .. tostring(keybindings))
	end

	return keybindings or {}
end

return M
