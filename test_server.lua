-- Simple test to check if the server binary works
-- This will help us understand what's failing

local function test_server_binary()
	local binary_path = "./server"

	-- Check if binary exists and is executable
	if vim.fn.executable(binary_path) ~= 1 then
		print("❌ Server binary not found or not executable: " .. binary_path)
		return false
	end

	print("✅ Server binary found: " .. binary_path)

	-- Try to start the server and send a simple health check
	local job_id = vim.fn.jobstart({ binary_path }, {
		on_stdout = function(job_id, data)
			for _, line in ipairs(data) do
				if line and line ~= "" then
					print("📥 Server output: " .. line)
				end
			end
		end,
		on_stderr = function(job_id, data)
			for _, line in ipairs(data) do
				if line and line ~= "" then
					print("❌ Server error: " .. line)
				end
			end
		end,
		on_exit = function(job_id, exit_code)
			print("🔚 Server exited with code: " .. exit_code)
		end,
		stdout_buffered = false,
		stderr_buffered = false,
	})

	if job_id <= 0 then
		print("❌ Failed to start server process")
		return false
	end

	print("✅ Server started with job ID: " .. job_id)

	-- Wait a moment for initialization
	vim.defer_fn(function()
		-- Send a simple health check request
		local health_request = {
			jsonrpc = "2.0",
			method = "HealthCheck",
			params = {},
			id = 1,
		}

		local json_data = vim.json.encode(health_request) .. "\n"
		print("📤 Sending health check: " .. json_data:sub(1, -2))

		local success = vim.fn.chansend(job_id, json_data)
		if success == 0 then
			print("❌ Failed to send health check request")
		else
			print("✅ Health check request sent")
		end

		-- Stop the server after a few seconds
		vim.defer_fn(function()
			vim.fn.jobstop(job_id)
			print("🛑 Server stopped")
		end, 5000)
	end, 2000)

	return true
end

-- Run the test
test_server_binary()
