# User Guide

This guide covers how to use nvim-smart-keybind-search effectively.

## Getting Started

### First Time Setup

1. **Install the plugin** (see README.md for installation instructions)
2. **Start the backend server:**
   ```bash
   ./start_server.sh
   ```
3. **Test the installation:**
   ```vim
   :SmartKeybindHealth
   ```

### Basic Usage

#### Opening the Search Interface

```vim
:SmartKeybindSearch
```

Or use the default keymap:
```vim
<leader>ks
```

#### Making Queries

Type your query in natural language:

**Movement queries:**
- "move to next word"
- "go to beginning of line"
- "jump to end of file"
- "move up 5 lines"

**Editing queries:**
- "delete current line"
- "copy this word"
- "paste after cursor"
- "change inner word"

**Search queries:**
- "find text forward"
- "search and replace"
- "go to next match"

**Visual mode queries:**
- "select this word"
- "visual block mode"
- "select inner quotes"

#### Understanding Results

Results show:
- **Keys**: The actual keybinding (e.g., "dd")
- **Command**: What it does (e.g., "delete line")
- **Description**: Detailed explanation
- **Mode**: Which vim mode it works in (n, v, i, c)
- **Relevance**: How well it matches your query (0.0-1.0)

#### Executing Keybindings

1. **Select a result** from the list
2. **Press Enter** to execute the keybinding
3. **Press 'c'** to copy the keybinding to clipboard
4. **Press 'e'** to see detailed explanation

## Advanced Usage

### Query Techniques

#### Be Specific
```
❌ "move"
✅ "move to next word"
✅ "move cursor left"
```

#### Use Synonyms
```
"copy" = "yank"
"delete" = "remove"
"find" = "search"
"paste" = "put"
```

#### Combine Actions
```
"delete and paste"
"copy and move"
"select and change"
```

#### Specify Context
```
"delete in visual mode"
"move in insert mode"
"search in command mode"
```

### Custom Keybindings

#### Syncing Your Keybindings

The plugin automatically detects and syncs your custom keybindings:

```vim
" Your custom mappings
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>
vnoremap <leader>y "+y
```

Run sync manually:
```vim
:SmartKeybindSync
```

#### Adding Custom Descriptions

For better search results, add descriptions to your mappings:

```lua
-- In your init.lua or config
vim.keymap.set('n', '<leader>w', ':w<CR>', { 
  desc = 'save current file' 
})
vim.keymap.set('n', '<leader>q', ':q<CR>', { 
  desc = 'quit current buffer' 
})
```

### Configuration Options

#### Basic Configuration

```lua
require("nvim-smart-keybind-search").setup({
  -- Server connection
  server_host = "localhost",
  server_port = 8080,
  
  -- Auto-start server when Neovim starts
  auto_start = true,
  
  -- Health check interval (seconds)
  health_check_interval = 30,
})
```

#### Search Configuration

```lua
require("nvim-smart-keybind-search").setup({
  search = {
    -- Maximum number of results to show
    max_results = 10,
    
    -- Minimum relevance score (0.0 - 1.0)
    min_relevance = 0.3,
    
    -- Show explanations in results
    show_explanations = true,
    
    -- Include your custom keybindings in search
    include_custom = true,
    
    -- Prioritize custom keybindings
    prioritize_custom = true,
  },
})
```

#### UI Configuration

```lua
require("nvim-smart-keybind-search").setup({
  ui = {
    -- Use telescope for results display
    use_telescope = true,
    
    -- Custom telescope configuration
    telescope_config = {
      layout_config = {
        width = 0.8,
        height = 0.6,
      },
      sorting_strategy = "ascending",
    },
    
    -- Custom keymaps for results
    keymaps = {
      execute = "<CR>",
      copy = "c",
      explain = "e",
      close = "q",
    },
  },
})
```

#### Logging Configuration

```lua
require("nvim-smart-keybind-search").setup({
  logging = {
    level = "info",  -- debug, info, warn, error
    file = "./logs/plugin.log",
    max_size = "10MB",
    max_files = 5,
  },
})
```

### Performance Optimization

#### Caching

Enable result caching for faster repeated queries:

```lua
require("nvim-smart-keybind-search").setup({
  cache = {
    enabled = true,
    ttl = 300,  -- Cache for 5 minutes
    max_size = 1000,  -- Maximum cached results
  },
})
```

#### Streaming Responses

For large result sets, enable streaming:

```lua
require("nvim-smart-keybind-search").setup({
  streaming = true,
  stream_chunk_size = 5,
})
```

#### Connection Pooling

Optimize server connections:

```lua
require("nvim-smart-keybind-search").setup({
  connection = {
    pool_size = 5,
    timeout = 30,
    retry_attempts = 3,
  },
})
```

## Best Practices

### Writing Effective Queries

#### 1. Use Natural Language
```
❌ "dd"
✅ "delete current line"

❌ "w"
✅ "move to next word"
```

#### 2. Be Descriptive
```
❌ "move"
✅ "move cursor to next word"

❌ "copy"
✅ "copy selected text to clipboard"
```

#### 3. Specify Mode When Needed
```
"delete in visual mode"
"paste in insert mode"
"search in command mode"
```

#### 4. Use Common Terms
```
"save file" instead of "write buffer"
"quit" instead of "exit"
"find" instead of "locate"
```

### Organizing Your Keybindings

#### 1. Use Descriptive Names

```lua
-- Good
vim.keymap.set('n', '<leader>fs', ':w<CR>', { desc = 'save file' })
vim.keymap.set('n', '<leader>fq', ':q<CR>', { desc = 'quit buffer' })

-- Avoid
vim.keymap.set('n', '<leader>a', ':w<CR>')
vim.keymap.set('n', '<leader>b', ':q<CR>')
```

#### 2. Group Related Mappings

```lua
-- File operations
vim.keymap.set('n', '<leader>fs', ':w<CR>', { desc = 'save file' })
vim.keymap.set('n', '<leader>fq', ':q<CR>', { desc = 'quit buffer' })
vim.keymap.set('n', '<leader>fn', ':e<CR>', { desc = 'new file' })

-- Buffer operations
vim.keymap.set('n', '<leader>bn', ':bn<CR>', { desc = 'next buffer' })
vim.keymap.set('n', '<leader>bp', ':bp<CR>', { desc = 'previous buffer' })
vim.keymap.set('n', '<leader>bd', ':bd<CR>', { desc = 'delete buffer' })
```

#### 3. Use Consistent Naming

```lua
-- Use consistent prefixes
vim.keymap.set('n', '<leader>w', ':w<CR>', { desc = 'save file' })
vim.keymap.set('n', '<leader>wq', ':wq<CR>', { desc = 'save and quit' })
vim.keymap.set('n', '<leader>w!', ':w!<CR>', { desc = 'force save' })
```

### Troubleshooting Queries

#### When No Results Found

1. **Try different terms:**
   ```
   "copy" → "yank"
   "delete" → "remove"
   "find" → "search"
   ```

2. **Be more specific:**
   ```
   "move" → "move to next word"
   "edit" → "change inner word"
   ```

3. **Check mode:**
   ```
   "delete in visual mode"
   "paste in insert mode"
   ```

4. **Lower relevance threshold:**
   ```lua
   require("nvim-smart-keybind-search").setup({
     search = {
       min_relevance = 0.1,  -- Lower from 0.3
     },
   })
   ```

#### When Too Many Results

1. **Be more specific:**
   ```
   "move" → "move to next word"
   "copy" → "copy current line"
   ```

2. **Limit results:**
   ```lua
   require("nvim-smart-keybind-search").setup({
     search = {
       max_results = 5,  -- Reduce from 10
     },
   })
   ```

3. **Increase relevance threshold:**
   ```lua
   require("nvim-smart-keybind-search").setup({
     search = {
       min_relevance = 0.5,  -- Increase from 0.3
     },
   })
   ```

## Examples

### Common Workflows

#### 1. Text Editing Workflow

```vim
" Find and select text
:SmartKeybindSearch "select this word"
" Result: viw (visual inner word)

" Delete the selected text
:SmartKeybindSearch "delete selected text"
" Result: d (delete in visual mode)

" Paste the deleted text
:SmartKeybindSearch "paste after cursor"
" Result: p (paste after cursor)
```

#### 2. File Navigation Workflow

```vim
" Open file explorer
:SmartKeybindSearch "open file explorer"
" Result: :Ex (netrw)

" Navigate to next file
:SmartKeybindSearch "next file"
" Result: :next

" Save current file
:SmartKeybindSearch "save file"
" Result: :w
```

#### 3. Search and Replace Workflow

```vim
" Find text
:SmartKeybindSearch "find text forward"
" Result: / (search forward)

" Replace text
:SmartKeybindSearch "replace text"
" Result: :s/old/new/g

" Find and replace all
:SmartKeybindSearch "find and replace all"
" Result: :%s/old/new/g
```

### Advanced Examples

#### 1. Complex Text Operations

```vim
" Change inner quotes
:SmartKeybindSearch "change inner quotes"
" Result: ci" (change inner quotes)

" Delete around parentheses
:SmartKeybindSearch "delete around parentheses"
" Result: da( (delete around parentheses)

" Yank inner word
:SmartKeybindSearch "yank inner word"
" Result: yiw (yank inner word)
```

#### 2. Window Management

```vim
" Split window horizontally
:SmartKeybindSearch "split window horizontally"
" Result: :sp

" Split window vertically
:SmartKeybindSearch "split window vertically"
" Result: :vsp

" Move to window above
:SmartKeybindSearch "move to window above"
" Result: <C-w>k
```

#### 3. Buffer Management

```vim
" List buffers
:SmartKeybindSearch "list buffers"
" Result: :ls

" Next buffer
:SmartKeybindSearch "next buffer"
" Result: :bn

" Delete buffer
:SmartKeybindSearch "delete buffer"
" Result: :bd
```

## Tips and Tricks

### 1. Use Aliases

Create command aliases for common queries:

```vim
" In your init.vim or init.lua
command! -nargs=? Save :SmartKeybindSearch "save file"
command! -nargs=? Quit :SmartKeybindSearch "quit"
command! -nargs=? Find :SmartKeybindSearch "find text"
```

### 2. Custom Keymaps

Add custom keymaps for quick access:

```lua
-- Quick search
vim.keymap.set('n', '<leader>ks', ':SmartKeybindSearch<CR>')

-- Quick sync
vim.keymap.set('n', '<leader>ky', ':SmartKeybindSync<CR>')

-- Quick health check
vim.keymap.set('n', '<leader>kh', ':SmartKeybindHealth<CR>')
```

### 3. Integration with Other Plugins

#### Telescope Integration

```lua
-- Add to telescope extensions
require('telescope').setup({
  extensions = {
    ['nvim-smart-keybind-search'] = {
      -- Custom telescope config
    },
  },
})
```

#### Which-Key Integration

```lua
-- Add to which-key
require("which-key").register({
  ["<leader>k"] = {
    name = "Smart Keybind Search",
    s = { ":SmartKeybindSearch<CR>", "Search Keybindings" },
    y = { ":SmartKeybindSync<CR>", "Sync Keybindings" },
    h = { ":SmartKeybindHealth<CR>", "Health Check" },
  },
})
```

### 4. Performance Tips

1. **Use caching** for repeated queries
2. **Limit results** for faster response
3. **Be specific** in your queries
4. **Sync keybindings** regularly
5. **Monitor health** periodically

### 5. Debugging

Enable debug mode for troubleshooting:

```lua
require("nvim-smart-keybind-search").setup({
  logging = {
    level = "debug",
    file = "./logs/debug.log",
  },
})
```

Check logs for detailed information:
```bash
tail -f logs/debug.log
```

## Getting Help

- **Documentation**: Check the README.md and docs/
- **Health Check**: Run `:SmartKeybindHealth`
- **Logs**: Check `logs/plugin.log` and `logs/server.log`
- **Issues**: Report on GitHub with logs and steps to reproduce 