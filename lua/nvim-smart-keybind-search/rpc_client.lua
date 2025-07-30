-- JSON-RPC client for communicating with Go backend
-- Handles connection management, request queuing, and automatic reconnection

local M = {}

-- Client state
local client_state = {
	job_id = nil,
	config = nil,
	request_id = 0,
	pending_requests = {},
	request_queue = {},
	is_connected = false,
	reconnect_timer = nil,
	reconnect_attempts = 0,
	max_reconnect_attempts = 5,
	reconnect_delay = 1000, -- milliseconds
}

-- JSON-RPC message types
local JSON_RPC_VERSION = "2.0"

--- Generate next request ID
--- @return number Request ID
local function next_request_id()
	client_state.request_id = client_state.request_id + 1
	return client_state.request_id
end

--- Create JSON-RPC request message
--- @param method string RPC method name
--- @param params any Method parameters
--- @return table JSON-RPC request
local function create_request(method, params)
	return {
		jsonrpc = JSON_RPC_VERSION,
		id = next_request_id(),
		method = method,
		params = params or {},
	}
end

--- Log debug message
--- @param message string Message to log
local function log_debug(message)
	if client_state.config and client_state.config.logging and client_state.config.logging.debug then
		vim.notify("[RPC] " .. message, vim.log.levels.DEBUG)
	end
end

--- Log error message
--- @param message string Error message
local function log_error(message)
	vim.notify("[RPC Error] " .. message, vim.log.levels.ERROR)
end

--- Handle JSON-RPC response from backend
--- @param response table JSON-RPC response
local function handle_response(response)
	if not response.id then
		log_error("Received response without ID")
		return
	end

	local pending = client_state.pending_requests[response.id]
	if not pending then
		log_error("Received response for unknown request ID: " .. response.id)
		return
	end

	-- Clear timeout timer
	if pending.timer then
		vim.fn.timer_stop(pending.timer)
	end

	-- Remove from pending requests
	client_state.pending_requests[response.id] = nil

	-- Call callback with result or error
	if response.error then
		pending.callback(nil, response.error.message or "Unknown error")
	else
		pending.callback(response.result, nil)
	end
end

--- Handle backend output (stdout)
--- @param job_id number Job ID
--- @param data table Output data lines
local function handle_stdout(job_id, data)
	if job_id ~= client_state.job_id then
		return
	end

	for _, line in ipairs(data) do
		if line and line ~= "" then
			log_debug("Received: " .. line)

			local ok, response = pcall(vim.json.decode, line)
			if ok and response then
				handle_response(response)
			else
				log_error("Failed to parse JSON response: " .. line)
			end
		end
	end
end

--- Handle backend error output (stderr)
--- @param job_id number Job ID
--- @param data table Error data lines
local function handle_stderr(job_id, data)
	if job_id ~= client_state.job_id then
		return
	end

	for _, line in ipairs(data) do
		if line and line ~= "" then
			log_error("Backend stderr: " .. line)
		end
	end
end

--- Handle backend exit
--- @param job_id number Job ID
--- @param exit_code number Exit code
local function handle_exit(job_id, exit_code)
	if job_id ~= client_state.job_id then
		return
	end

	log_debug("Backend exited with code: " .. exit_code)
	client_state.is_connected = false
	client_state.job_id = nil

	-- Fail all pending requests
	for id, pending in pairs(client_state.pending_requests) do
		if pending.timer then
			vim.fn.timer_stop(pending.timer)
		end
		pending.callback(nil, "Backend disconnected")
	end
	client_state.pending_requests = {}

	-- Note: Reconnection logic removed for simplicity
end

--- Start the Go backend process
--- @return boolean Success
local function start_backend()
	if not client_state.config or not client_state.config.backend.binary_path then
		log_error("Backend binary path not configured")
		return false
	end

	local binary_path = client_state.config.backend.binary_path
	if vim.fn.executable(binary_path) ~= 1 then
		log_error("Backend binary not found or not executable: " .. binary_path)
		return false
	end

	log_debug("Starting backend: " .. binary_path)

	-- Start the backend process
	local job_id = vim.fn.jobstart({ binary_path }, {
		on_stdout = handle_stdout,
		on_stderr = handle_stderr,
		on_exit = handle_exit,
		stdout_buffered = false,
		stderr_buffered = false,
	})

	if job_id <= 0 then
		log_error("Failed to start backend process")
		return false
	end

	client_state.job_id = job_id
	client_state.is_connected = true
	client_state.reconnect_attempts = 0

	log_debug("Backend started with job ID: " .. job_id)
	return true
end

--- Connect to backend (start if needed)
--- @return boolean Success
local function connect_backend()
	if client_state.is_connected then
		return true
	end

	return start_backend()
end

--- Send JSON-RPC request to backend
--- @param method string RPC method name
--- @param params any Method parameters
--- @param callback function Callback function(result, error)
--- @param timeout? number Optional timeout in milliseconds
local function send_request(method, params, callback, timeout)
	-- Queue request if not connected
	if not client_state.is_connected then
		table.insert(client_state.request_queue, {
			method = method,
			params = params,
			callback = callback,
			timeout = timeout,
		})

		-- Try to connect
		if not connect_backend() then
			callback(nil, "Failed to connect to backend")
		end
		return
	end

	local request = create_request(method, params)
	local request_timeout = timeout or (client_state.config and client_state.config.backend.timeout) or 5000

	-- Store pending request
	client_state.pending_requests[request.id] = {
		callback = callback,
		timer = nil,
	}

	-- Set timeout timer
	client_state.pending_requests[request.id].timer = vim.fn.timer_start(request_timeout, function()
		local pending = client_state.pending_requests[request.id]
		if pending then
			client_state.pending_requests[request.id] = nil
			pending.callback(nil, "Request timeout")
		end
	end)

	-- Send request
	local json_data = vim.json.encode(request) .. "\n"
	log_debug("Sending: " .. json_data:sub(1, -2)) -- Remove newline for logging

	local success = vim.fn.chansend(client_state.job_id, json_data)
	if success == 0 then
		-- Failed to send, clean up
		if client_state.pending_requests[request.id] then
			if client_state.pending_requests[request.id].timer then
				vim.fn.timer_stop(client_state.pending_requests[request.id].timer)
			end
			client_state.pending_requests[request.id] = nil
		end
		callback(nil, "Failed to send request to backend")
	end
end

--- Process queued requests
local function process_queue()
	if not client_state.is_connected or #client_state.request_queue == 0 then
		return
	end

	local queue = client_state.request_queue
	client_state.request_queue = {}

	for _, queued_request in ipairs(queue) do
		send_request(queued_request.method, queued_request.params, queued_request.callback, queued_request.timeout)
	end
end

--- Setup the RPC client with backend configuration
--- @param backend_config table Backend configuration
function M.setup(backend_config)
	client_state.config = backend_config

	-- Connect to backend if auto_start is enabled
	if backend_config.backend.auto_start then
		connect_backend()

		-- Process any queued requests after connection
		vim.defer_fn(process_queue, 100)
	end

	log_debug("RPC client setup completed")
end

--- Query the backend for keybinding search
--- @param query string Search query
--- @param callback function Callback function(results, error)
function M.query(query, callback)
	if not query or query == "" then
		callback(nil, "Query cannot be empty")
		return
	end

	send_request("Query", { query = query }, callback)
end

--- Sync all keybindings with the backend
--- @param keybindings table List of keybindings
--- @param callback function Callback function(success, error)
function M.sync_keybindings(keybindings, callback)
	send_request("SyncKeybindings", { keybindings = keybindings or {} }, function(result, error)
		if error then
			callback(false, error)
		else
			callback(true, nil)
		end
	end)
end

--- Update specific keybindings
--- @param keybindings table List of changed keybindings
--- @param callback function Callback function(success, error)
function M.update_keybindings(keybindings, callback)
	send_request("UpdateKeybindings", { keybindings = keybindings or {} }, function(result, error)
		if error then
			callback(false, error)
		else
			callback(true, nil)
		end
	end)
end

--- Check backend health
--- @param callback function Callback function(health_status, error)
function M.health_check(callback)
	send_request("HealthCheck", {}, callback, 2000) -- Shorter timeout for health checks
end

--- Disconnect from backend
function M.disconnect()
	if client_state.reconnect_timer then
		vim.fn.timer_stop(client_state.reconnect_timer)
		client_state.reconnect_timer = nil
	end

	if client_state.job_id then
		vim.fn.jobstop(client_state.job_id)
		client_state.job_id = nil
	end

	client_state.is_connected = false
	client_state.reconnect_attempts = 0

	-- Clear pending requests
	for id, pending in pairs(client_state.pending_requests) do
		if pending.timer then
			vim.fn.timer_stop(pending.timer)
		end
		pending.callback(nil, "Client disconnected")
	end
	client_state.pending_requests = {}
	client_state.request_queue = {}

	log_debug("RPC client disconnected")
end

--- Check if client is connected
--- @return boolean Connection status
function M.is_connected()
	return client_state.is_connected
end

--- Get connection statistics
--- @return table Connection stats
function M.get_stats()
	return {
		connected = client_state.is_connected,
		job_id = client_state.job_id,
		pending_requests = vim.tbl_count(client_state.pending_requests),
		queued_requests = #client_state.request_queue,
		reconnect_attempts = client_state.reconnect_attempts,
	}
end

return M
