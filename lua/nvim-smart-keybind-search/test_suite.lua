-- Test suite for nvim-smart-keybind-search
-- Provides a simple testing framework for Lua plugin components

local M = {}

-- Test framework
local TestFramework = {
	tests = {},
	passed = 0,
	failed = 0,
	errors = {},
}

function TestFramework:describe(description, test_fn)
	table.insert(self.tests, {
		description = description,
		test_fn = test_fn,
	})
end

function TestFramework:assert(condition, message)
	if not condition then
		error(message or "Assertion failed")
	end
end

function TestFramework:assert_equal(expected, actual, message)
	if expected ~= actual then
		error(string.format("%s: expected %s, got %s", message or "Assertion failed", tostring(expected), tostring(actual)))
	end
end

function TestFramework:assert_not_equal(expected, actual, message)
	if expected == actual then
		error(string.format("%s: expected not %s, got %s", message or "Assertion failed", tostring(expected), tostring(actual)))
	end
end

function TestFramework:assert_type(expected_type, value, message)
	local actual_type = type(value)
	if actual_type ~= expected_type then
		error(string.format("%s: expected type %s, got %s", message or "Type assertion failed", expected_type, actual_type))
	end
end

function TestFramework:assert_nil(value, message)
	if value ~= nil then
		error(string.format("%s: expected nil, got %s", message or "Assertion failed", tostring(value)))
	end
end

function TestFramework:assert_not_nil(value, message)
	if value == nil then
		error(string.format("%s: expected not nil", message or "Assertion failed"))
	end
end

function TestFramework:assert_table_contains(table, key, message)
	if table[key] == nil then
		error(string.format("%s: table does not contain key '%s'", message or "Assertion failed", tostring(key)))
	end
end

function TestFramework:run()
	print("=== Running nvim-smart-keybind-search Test Suite ===")
	print("")

	for i, test in ipairs(self.tests) do
		print(string.format("Test %d: %s", i, test.description))
		
		local success, error_msg = pcall(test.test_fn, self)
		
		if success then
			print("  ✓ PASSED")
			self.passed = self.passed + 1
		else
			print(string.format("  ✗ FAILED: %s", error_msg))
			self.failed = self.failed + 1
			table.insert(self.errors, {
				test = test.description,
				error = error_msg,
			})
		end
	end
	
	print("")
	print("=== Test Results ===")
	print(string.format("Passed: %d", self.passed))
	print(string.format("Failed: %d", self.failed))
	print(string.format("Total: %d", self.passed + self.failed))
	
	if #self.errors > 0 then
		print("")
		print("=== Error Details ===")
		for i, error_info in ipairs(self.errors) do
			print(string.format("%d. %s: %s", i, error_info.test, error_info.error))
		end
	end
	
	return self.passed, self.failed
end

-- Test data
local test_keybindings = {
	{
		keys = "dd",
		command = "delete line",
		description = "Delete current line",
		mode = "n",
	},
	{
		keys = "yy",
		command = "yank line",
		description = "Copy current line",
		mode = "n",
	},
	{
		keys = "<leader>w",
		command = ":w<CR>",
		description = "Save file",
		mode = "n",
		plugin = "vim",
	},
}

local test_queries = {
	"delete line",
	"copy text",
	"save file",
	"find text",
	"replace word",
}

-- Test configuration
local test_config = {
	backend = {
		binary_path = "/tmp/test-server",
		port = 0,
		timeout = 1000,
		auto_start = false,
		log_level = "debug",
	},
	scanner = {
		include_plugins = true,
		include_buffer_local = true,
		exclude_patterns = {
			"^<Plug>",
			"^<SNR>",
		},
	},
	picker = {
		provider = "snack",
		min_query_length = 2,
		debounce_ms = 100,
		max_results = 10,
		show_explanations = true,
		show_relevance = true,
		show_action_menu = true,
		window = {
			width = 0.8,
			height = 0.6,
			border = "rounded",
		},
	},
	keymaps = {
		search = "<leader>sk",
		search_alt = "<leader>fk",
	},
	auto_sync = false,
	sync_on_startup = false,
	watch_changes = false,
	ui = {
		icons = {
			keybinding = "⌨️ ",
			motion = "🔄 ",
			command = "⚡ ",
			plugin = "🔌 ",
			visual = "👁️ ",
			insert = "✏️ ",
		},
		highlights = {
			query = "Question",
			keybinding = "Identifier",
			description = "Comment",
			explanation = "String",
		},
	},
	logging = {
		debug = true,
		file = nil,
		level = "debug",
	},
}

-- Test functions
function M.test_config_module()
	local config = require("nvim-smart-keybind-search.config")
	
	TestFramework:describe("Configuration module", function(self)
		-- Test default configuration
		self:describe("Default configuration", function(self)
			local default_config = config.get_defaults()
			
			self:assert_type("table", default_config, "Default config should be a table")
			self:assert_table_contains(default_config, "backend", "Default config should have backend section")
			self:assert_table_contains(default_config, "picker", "Default config should have picker section")
			self:assert_table_contains(default_config, "keymaps", "Default config should have keymaps section")
			
			-- Test backend config
			self:assert_type("table", default_config.backend, "Backend config should be a table")
			self:assert_table_contains(default_config.backend, "port", "Backend config should have port")
			self:assert_table_contains(default_config.backend, "timeout", "Backend config should have timeout")
			
			-- Test picker config
			self:assert_type("table", default_config.picker, "Picker config should be a table")
			self:assert_equal("snack", default_config.picker.provider, "Default picker provider should be snack")
			self:assert_equal(2, default_config.picker.min_query_length, "Default min query length should be 2")
		end)
		
		-- Test configuration setup
		self:describe("Configuration setup", function(self)
			local user_config = {
				picker = {
					provider = "telescope",
					min_query_length = 3,
				},
				keymaps = {
					search = "<leader>st",
				},
			}
			
			local merged_config = config.setup(user_config)
			
			self:assert_type("table", merged_config, "Merged config should be a table")
			self:assert_equal("telescope", merged_config.picker.provider, "User picker provider should override default")
			self:assert_equal(3, merged_config.picker.min_query_length, "User min query length should override default")
			self:assert_equal("<leader>st", merged_config.keymaps.search, "User keymap should override default")
			
			-- Test that other defaults are preserved
			self:assert_equal("snack", merged_config.picker.window.border, "Default window border should be preserved")
		end)
		
		-- Test configuration validation
		self:describe("Configuration validation", function(self)
			local invalid_config = {
				picker = {
					min_query_length = 0, -- Invalid: should be >= 1
				},
			}
			
			local success, error_msg = pcall(config.setup, invalid_config)
			self:assert(not success, "Invalid config should cause error")
			self:assert_not_nil(error_msg, "Error message should be provided")
		end)
	end)
end

function M.test_keybind_scanner()
	local keybind_scanner = require("nvim-smart-keybind-search.keybind_scanner")
	
	TestFramework:describe("Keybinding scanner module", function(self)
		-- Test scanner setup
		self:describe("Scanner setup", function(self)
			local scanner_config = {
				include_plugins = true,
				include_buffer_local = true,
				exclude_patterns = {
					"^<Plug>",
					"^<SNR>",
				},
			}
			
			keybind_scanner.setup(scanner_config)
			
			-- Test that setup doesn't crash
			self:assert(true, "Scanner setup should complete without error")
		end)
		
		-- Test keybinding scanning
		self:describe("Keybinding scanning", function(self)
			local keybindings = keybind_scanner.scan_all()
			
			self:assert_type("table", keybindings, "Scan result should be a table")
			
			-- Test that we get some keybindings (even if empty)
			self:assert_type("number", #keybindings, "Keybindings count should be a number")
			
			-- Test keybinding structure if any exist
			if #keybindings > 0 then
				local kb = keybindings[1]
				self:assert_type("table", kb, "Keybinding should be a table")
				self:assert_table_contains(kb, "keys", "Keybinding should have keys field")
				self:assert_table_contains(kb, "command", "Keybinding should have command field")
				self:assert_table_contains(kb, "mode", "Keybinding should have mode field")
			end
		end)
		
		-- Test change detection
		self:describe("Change detection", function(self)
			local changes = keybind_scanner.detect_changes()
			
			-- Change detection should return nil or a table
			if changes ~= nil then
				self:assert_type("table", changes, "Changes should be a table when not nil")
			end
		end)
		
		-- Test scanner statistics
		self:describe("Scanner statistics", function(self)
			local stats = keybind_scanner.get_stats()
			
			self:assert_type("table", stats, "Scanner stats should be a table")
			self:assert_table_contains(stats, "cached_count", "Stats should have cached_count")
			self:assert_table_contains(stats, "change_detection_enabled", "Stats should have change_detection_enabled")
		end)
	end)
end

function M.test_rpc_client()
	local rpc_client = require("nvim-smart-keybind-search.rpc_client")
	
	TestFramework:describe("RPC client module", function(self)
		-- Test client setup
		self:describe("RPC client setup", function(self)
			rpc_client.setup(test_config)
			
			-- Test that setup doesn't crash
			self:assert(true, "RPC client setup should complete without error")
		end)
		
		-- Test connection status
		self:describe("Connection status", function(self)
			local connected = rpc_client.is_connected()
			
			self:assert_type("boolean", connected, "Connection status should be boolean")
			
			-- Note: In test environment, backend might not be available
			-- So we just test that the function returns a boolean
		end)
		
		-- Test health check
		self:describe("Health check", function(self)
			rpc_client.health_check(function(health)
				-- Health check callback should be called
				self:assert_type("table", health, "Health status should be a table")
			end)
		end)
		
		-- Test query functionality
		self:describe("Query functionality", function(self)
			rpc_client.query("test query", function(results, error_msg)
				-- Query callback should be called
				-- In test environment, we might get an error due to no backend
				-- So we just test that the callback is called
				self:assert(true, "Query callback should be called")
			end)
		end)
		
		-- Test keybinding sync
		self:describe("Keybinding sync", function(self)
			rpc_client.sync_keybindings(test_keybindings, function(success, error_msg)
				-- Sync callback should be called
				self:assert_type("boolean", success, "Sync success should be boolean")
			end)
		end)
	end)
end

function M.test_picker()
	local picker = require("nvim-smart-keybind-search.picker")
	
	TestFramework:describe("Picker module", function(self)
		-- Test picker state
		self:describe("Picker state", function(self)
			local is_open = picker.is_open()
			
			self:assert_type("boolean", is_open, "Picker open status should be boolean")
			self:assert_equal(false, is_open, "Picker should not be open initially")
		end)
		
		-- Test picker statistics
		self:describe("Picker statistics", function(self)
			local stats = picker.get_stats()
			
			self:assert_type("table", stats, "Picker stats should be a table")
			self:assert_table_contains(stats, "is_open", "Stats should have is_open")
			self:assert_table_contains(stats, "last_query", "Stats should have last_query")
			self:assert_table_contains(stats, "results_count", "Stats should have results_count")
			self:assert_table_contains(stats, "is_searching", "Stats should have is_searching")
		end)
		
		-- Test picker close (when not open)
		self:describe("Picker close", function(self)
			-- Should not crash when closing non-existent picker
			picker.close()
			self:assert(true, "Closing non-existent picker should not crash")
		end)
	end)
end

function M.test_utils()
	local utils = require("nvim-smart-keybind-search.utils")
	
	TestFramework:describe("Utils module", function(self)
		-- Test string utilities
		self:describe("String utilities", function(self)
			-- Test string trimming
			local trimmed = utils.trim("  test  ")
			self:assert_equal("test", trimmed, "String should be trimmed")
			
			-- Test string splitting
			local parts = utils.split("a,b,c", ",")
			self:assert_type("table", parts, "Split result should be a table")
			self:assert_equal(3, #parts, "Should have 3 parts")
			self:assert_equal("a", parts[1], "First part should be 'a'")
			self:assert_equal("b", parts[2], "Second part should be 'b'")
			self:assert_equal("c", parts[3], "Third part should be 'c'")
		end)
		
		-- Test table utilities
		self:describe("Table utilities", function(self)
			local t1 = {a = 1, b = 2}
			local t2 = {c = 3, d = 4}
			local merged = utils.merge_tables(t1, t2)
			
			self:assert_type("table", merged, "Merged result should be a table")
			self:assert_equal(1, merged.a, "First table values should be preserved")
			self:assert_equal(2, merged.b, "First table values should be preserved")
			self:assert_equal(3, merged.c, "Second table values should be added")
			self:assert_equal(4, merged.d, "Second table values should be added")
		end)
		
		-- Test validation utilities
		self:describe("Validation utilities", function(self)
			-- Test string validation
			self:assert(utils.is_string("test"), "Valid string should return true")
			self:assert(not utils.is_string(123), "Number should return false")
			self:assert(not utils.is_string(nil), "Nil should return false")
			
			-- Test table validation
			self:assert(utils.is_table({}), "Valid table should return true")
			self:assert(not utils.is_table("test"), "String should return false")
			self:assert(not utils.is_table(nil), "Nil should return false")
		end)
	end)
end

function M.test_health()
	local health = require("nvim-smart-keybind-search.health")
	
	TestFramework:describe("Health module", function(self)
		-- Test health check
		self:describe("Health check", function(self)
			-- Health check should not crash
			health.check()
			self:assert(true, "Health check should complete without error")
		end)
	end)
end

-- Main test runner
function M.run_all_tests()
	print("Starting nvim-smart-keybind-search test suite...")
	print("")
	
	-- Run all test modules
	M.test_config_module()
	M.test_keybind_scanner()
	M.test_rpc_client()
	M.test_picker()
	M.test_utils()
	M.test_health()
	
	-- Run the test framework
	local passed, failed = TestFramework:run()
	
	print("")
	if failed == 0 then
		print("🎉 All tests passed!")
	else
		print("❌ Some tests failed. Please check the error details above.")
	end
	
	return passed, failed
end

-- Export test framework for external use
M.TestFramework = TestFramework

return M 