# Troubleshooting Guide

This guide covers common issues and their solutions for nvim-smart-keybind-search.

## Quick Health Check

First, run a comprehensive health check:

```bash
# From terminal
./health_check.sh

# From Neovim
:SmartKeybindHealth
```

## Common Issues

### 1. Server Won't Start

#### Symptoms
- Error: "Failed to start server"
- Error: "Connection refused"
- Plugin shows "Server unavailable"

#### Solutions

**Check if port is in use:**
```bash
# Linux/macOS
lsof -i :8080

# Windows
netstat -an | findstr :8080
```

**Kill existing processes:**
```bash
# Linux/macOS
pkill -f server
pkill -f chroma

# Windows
taskkill /f /im server.exe
taskkill /f /im chroma.exe
```

**Check server logs:**
```bash
tail -f logs/server.log
```

**Manual server start:**
```bash
# Start ChromaDB first
chroma run --path ./data/chroma --host localhost --port 8000 &

# Start server
./server
```

### 2. Ollama Not Running

#### Symptoms
- Error: "LLM client unavailable"
- Slow or no responses
- Health check shows "Ollama not running"

#### Solutions

**Install Ollama:**
```bash
# macOS/Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download from https://ollama.ai
```

**Start Ollama:**
```bash
# Start Ollama daemon
ollama serve

# In another terminal, pull a model
ollama pull llama2:7b
```

**Check Ollama status:**
```bash
curl http://localhost:11434/api/tags
```

**Verify model is loaded:**
```bash
ollama list
```

### 3. ChromaDB Issues

#### Symptoms
- Error: "Vector database unavailable"
- Database not found errors
- Empty search results

#### Solutions

**Install ChromaDB:**
```bash
pip install chromadb==0.4.22
```

**Start ChromaDB:**
```bash
chroma run --path ./data/chroma --host localhost --port 8000
```

**Rebuild database:**
```bash
# Stop ChromaDB
pkill -f chroma

# Rebuild database
go run scripts/build_database.go
```

**Check database files:**
```bash
ls -la data/chroma/
```

### 4. No Search Results

#### Symptoms
- Empty results for valid queries
- "No keybindings found" message
- Low relevance scores

#### Solutions

**Check database population:**
```bash
# Check if database has data
curl -X GET "http://localhost:8000/api/v1/collections/keybindings/count"
```

**Re-sync keybindings:**
```vim
:SmartKeybindSync
```

**Lower relevance threshold:**
```lua
require("nvim-smart-keybind-search").setup({
  search = {
    min_relevance = 0.1,  -- Lower from default 0.3
  },
})
```

**Check query format:**
- Use natural language: "delete line" not "dd"
- Be specific: "move to next word" not "move"
- Try synonyms: "copy" instead of "yank"

### 5. Slow Response Times

#### Symptoms
- Queries take >2 seconds
- Timeout errors
- Laggy interface

#### Solutions

**Check system resources:**
```bash
# Check CPU and memory usage
top
htop  # if available
```

**Optimize Ollama model:**
```bash
# Use smaller model
ollama pull llama2:3b

# Update config.json
{
  "ollama": {
    "model": "llama2:3b"
  }
}
```

**Check network connectivity:**
```bash
ping localhost
curl -w "@-" -o /dev/null -s "http://localhost:8080/health"
```

**Increase timeout:**
```lua
require("nvim-smart-keybind-search").setup({
  server_timeout = 60,  -- Increase from default 30
})
```

### 6. Plugin Not Loading

#### Symptoms
- Commands not available
- Error: "module not found"
- Plugin manager issues

#### Solutions

**Check Neovim version:**
```vim
:version
```
Requires Neovim 0.7+

**Verify plugin installation:**
```vim
:checkhealth nvim-smart-keybind-search
```

**Check plugin manager:**
```lua
-- For Lazy.nvim
:Lazy sync

-- For Packer.nvim
:PackerSync
```

**Manual plugin load:**
```vim
:lua require("nvim-smart-keybind-search").setup()
```

### 7. Permission Issues

#### Symptoms
- "Permission denied" errors
- Can't create files/directories
- Service won't start

#### Solutions

**Fix file permissions:**
```bash
chmod +x server
chmod +x scripts/*.sh
chmod 755 data/
chmod 755 logs/
```

**Check user permissions:**
```bash
# Ensure user owns the directory
sudo chown -R $USER:$USER .
```

**Run as correct user:**
```bash
# Don't run as root
sudo -u $USER ./server
```

### 8. Memory Issues

#### Symptoms
- Out of memory errors
- Slow performance
- Crashes

#### Solutions

**Check memory usage:**
```bash
free -h
ps aux | grep -E "(server|chroma|ollama)"
```

**Limit Ollama memory:**
```bash
# Set environment variable
export OLLAMA_HOST=127.0.0.1:11434
export OLLAMA_ORIGINS=*
export OLLAMA_MODELS=/path/to/models
```

**Use smaller model:**
```bash
ollama pull llama2:3b
```

**Restart services:**
```bash
./stop_server.sh
sleep 5
./start_server.sh
```

### 9. Network Issues

#### Symptoms
- Connection refused errors
- Timeout errors
- Can't reach localhost

#### Solutions

**Check firewall:**
```bash
# Linux
sudo ufw status
sudo iptables -L

# macOS
sudo pfctl -s rules
```

**Check port availability:**
```bash
# Test ports
nc -zv localhost 8080
nc -zv localhost 8000
nc -zv localhost 11434
```

**Use different ports:**
```json
{
  "server": {
    "port": 8081
  },
  "chromadb": {
    "port": 8001
  }
}
```

### 10. Database Corruption

#### Symptoms
- ChromaDB errors
- Missing data
- Inconsistent results

#### Solutions

**Backup and rebuild:**
```bash
# Backup existing data
cp -r data/chroma data/chroma.backup

# Remove corrupted database
rm -rf data/chroma

# Rebuild database
go run scripts/build_database.go
```

**Check database integrity:**
```bash
# Check ChromaDB collections
curl -X GET "http://localhost:8000/api/v1/collections"
```

**Verify data files:**
```bash
ls -la data/chroma/
file data/chroma/*
```

## Performance Optimization

### 1. Faster Response Times

**Use smaller LLM model:**
```bash
ollama pull llama2:3b
```

**Optimize ChromaDB:**
```bash
# Increase memory limit
export CHROMA_SERVER_HOST=localhost
export CHROMA_SERVER_HTTP_PORT=8000
export CHROMA_SERVER_CORS_ALLOW_ORIGINS=["*"]
```

**Cache results:**
```lua
require("nvim-smart-keybind-search").setup({
  cache = {
    enabled = true,
    ttl = 300,  -- 5 minutes
  },
})
```

### 2. Memory Optimization

**Limit concurrent requests:**
```lua
require("nvim-smart-keybind-search").setup({
  max_concurrent_requests = 5,
})
```

**Use streaming responses:**
```lua
require("nvim-smart-keybind-search").setup({
  streaming = true,
})
```

### 3. Storage Optimization

**Compress database:**
```bash
# Archive old logs
tar -czf logs-$(date +%Y%m%d).tar.gz logs/
rm logs/*.log
```

**Clean temporary files:**
```bash
find . -name "*.tmp" -delete
find . -name "*.cache" -delete
```

## Debug Mode

Enable debug logging for detailed troubleshooting:

```lua
require("nvim-smart-keybind-search").setup({
  logging = {
    level = "debug",
    file = "./logs/debug.log",
  },
})
```

```json
{
  "logging": {
    "level": "debug",
    "file": "./logs/server-debug.log"
  }
}
```

## Getting Help

### 1. Collect Information

Before asking for help, collect:

```bash
# System information
uname -a
go version
python --version
nvim --version

# Plugin health
:SmartKeybindHealth

# Logs
tail -n 50 logs/server.log
tail -n 50 logs/plugin.log

# Configuration
cat config.json
```

### 2. Report Issues

When reporting issues, include:

1. **Environment**: OS, Neovim version, Go version
2. **Steps to reproduce**: Exact commands and queries
3. **Expected vs actual behavior**: What you expected vs what happened
4. **Logs**: Relevant log entries
5. **Configuration**: Your setup configuration

### 3. Community Support

- **GitHub Issues**: [Create an issue](https://github.com/nest/nvim-smart-keybind-search/issues)
- **GitHub Discussions**: [Ask questions](https://github.com/nest/nvim-smart-keybind-search/discussions)
- **Wiki**: [Check documentation](https://github.com/nest/nvim-smart-keybind-search/wiki)

## Recovery Procedures

### Complete Reset

If all else fails, perform a complete reset:

```bash
# Stop all services
./stop_server.sh

# Remove all data
rm -rf data/
rm -rf logs/
rm -f server
rm -f config.json

# Reinstall
./scripts/install.sh

# Start fresh
./start_server.sh
```

### Backup and Restore

**Create backup:**
```bash
tar -czf nvim-smart-keybind-search-backup-$(date +%Y%m%d).tar.gz \
  data/ logs/ config.json server
```

**Restore backup:**
```bash
tar -xzf nvim-smart-keybind-search-backup-YYYYMMDD.tar.gz
./start_server.sh
``` 