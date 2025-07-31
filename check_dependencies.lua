-- Check what dependencies are available for the plugin

local function check_command(cmd)
	local result = vim.fn.executable(cmd) == 1
	local status = result and "✅" or "❌"
	print(status .. " " .. cmd .. ": " .. (result and "available" or "not found"))
	return result
end

local function check_python_module(module)
	local cmd = string.format("python3 -c 'import %s; print(\"OK\")'", module)
	local result = vim.fn.system(cmd)
	local success = vim.v.shell_error == 0 and result:match("OK")
	local status = success and "✅" or "❌"
	print(status .. " Python module '" .. module .. "': " .. (success and "available" or "not found"))
	return success
end

print("=== Dependency Check for nvim-smart-keybind-search ===")
print()

print("📦 System Commands:")
check_command("python3")
check_command("pip3")
check_command("curl")
check_command("wget")
check_command("uv")
check_command("go")

print()
print("🐍 Python Modules:")
check_python_module("chromadb")

print()
print("🔧 Plugin Files:")
local plugin_files = {
	"./server",
	"./main",
	"lua/nvim-smart-keybind-search/init.lua",
	"cmd/server/main.go",
}

for _, file in ipairs(plugin_files) do
	local exists = vim.fn.filereadable(file) == 1
	local status = exists and "✅" or "❌"
	print(status .. " " .. file .. ": " .. (exists and "exists" or "not found"))
end

print()
print("💡 Recommendations:")
print("1. If ChromaDB is not installed, the server will try to auto-install it")
print("2. If you don't have 'uv', install it with: curl -LsSf https://astral.sh/uv/install.sh | sh")
print("3. The server needs time to initialize ChromaDB and Ollama on first run")
print("4. Check server logs by running: ./server (it will show initialization progress)")
