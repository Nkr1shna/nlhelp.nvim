" nvim-smart-keybind-search plugin initialization
" This file is loaded automatically by Vim/Neovim

" Prevent loading if already loaded or if not in Neovim
if exists('g:loaded_nvim_smart_keybind_search') || !has('nvim')
  finish
endif

" Set loaded flag
let g:loaded_nvim_smart_keybind_search = 1

" Define commands that can be used before setup() is called
command! -nargs=? SmartKeybindSearch lua require('nvim-smart-keybind-search').search_keybindings(<q-args>)
command! SmartKeybindSync lua require('nvim-smart-keybind-search').sync_keybindings()
command! SmartKeybindHealth lua require('nvim-smart-keybind-search.health').check()

" Health check is handled automatically by the health.lua module
" No manual registration needed in modern Neovim