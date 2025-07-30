@echo off
setlocal enabledelayedexpansion

REM nvim-smart-keybind-search installation script for Windows
REM This script installs the Go backend, sets up the database, and configures the plugin

set "PLUGIN_NAME=nvim-smart-keybind-search"
set "GO_VERSION=1.21"
set "CHROMADB_VERSION=0.4.22"
set "OLLAMA_VERSION=0.1.29"

echo Installing %PLUGIN_NAME%...

REM Check if running in Neovim
if defined NVIM (
    echo Warning: This script should be run outside of Neovim
)

REM Check dependencies
echo Checking dependencies...

REM Check Go
go version >nul 2>&1
if errorlevel 1 (
    echo Go is not installed. Please install Go %GO_VERSION% or later.
    echo Visit: https://golang.org/doc/install
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set "GO_VERSION_CHECK=%%i"
echo ✓ Go %GO_VERSION_CHECK% found

REM Check Git
git --version >nul 2>&1
if errorlevel 1 (
    echo Git is not installed. Please install Git.
    exit /b 1
)
echo ✓ Git found

REM Check if we're in the plugin directory
if not exist "go.mod" (
    echo This script must be run from the plugin root directory
    exit /b 1
)
if not exist "internal\server\rpc.go" (
    echo This script must be run from the plugin root directory
    exit /b 1
)

REM Build the Go binary
echo Building Go backend...

REM Set build flags for better performance
set "CGO_ENABLED=1"
set "GOOS=windows"
set "GOARCH=amd64"

REM Build with optimizations
go build -ldflags="-s -w" -o server.exe cmd/server/main.go

if errorlevel 1 (
    echo Failed to build Go backend
    exit /b 1
)
echo ✓ Go backend built successfully

REM Build the database
echo Building pre-populated database...

REM Check if ChromaDB is available
chroma --version >nul 2>&1
if errorlevel 1 (
    echo ChromaDB CLI not found. Installing ChromaDB...
    
    REM Install ChromaDB using pip
    pip install chromadb==%CHROMADB_VERSION%
    if errorlevel 1 (
        echo pip not found. Please install Python and pip first.
        exit /b 1
    )
)

REM Start ChromaDB server for database building
echo Starting ChromaDB server for database initialization...

REM Start ChromaDB in background
start /B chroma run --path ./data/chroma --host localhost --port 8000

REM Wait for ChromaDB to start
timeout /t 5 /nobreak >nul

REM Build the database
go run scripts/build_database.go

if errorlevel 1 (
    echo Failed to build database
    taskkill /f /im chroma.exe >nul 2>&1
    exit /b 1
)

REM Stop ChromaDB
taskkill /f /im chroma.exe >nul 2>&1

echo ✓ Database built successfully

REM Create installation directories
echo Creating installation directories...

REM Create data directory
if not exist "data\chroma" mkdir "data\chroma"
if not exist "data\ollama" mkdir "data\ollama"

REM Create logs directory
if not exist "logs" mkdir "logs"

echo ✓ Installation directories created

REM Create configuration file
echo Creating configuration...

(
echo {
echo   "server": {
echo     "host": "localhost",
echo     "port": 8080,
echo     "timeout": 30
echo   },
echo   "chromadb": {
echo     "host": "localhost",
echo     "port": 8000,
echo     "path": "./data/chroma"
echo   },
echo   "ollama": {
echo     "host": "localhost",
echo     "port": 11434,
echo     "model": "llama2:7b"
echo   },
echo   "logging": {
echo     "level": "info",
echo     "file": "./logs/server.log"
echo   }
echo }
) > config.json

echo ✓ Configuration created

REM Create startup script
echo Creating startup script...

(
echo @echo off
echo REM Start script for nvim-smart-keybind-search server
echo.
echo setlocal enabledelayedexpansion
echo.
echo set "SCRIPT_DIR=%%~dp0"
echo cd /d "!SCRIPT_DIR!"
echo.
echo REM Load configuration
echo if exist "config.json" ^(
echo   for /f "tokens=2 delims=:," %%i in ^('findstr "port" config.json'^) do set "PORT=%%i"
echo   set "PORT=!PORT: =!"
echo ^) else ^(
echo   set "PORT=8080"
echo ^)
echo.
echo REM Check if server is already running
echo tasklist /fi "imagename eq server.exe" 2^>nul ^| find /i "server.exe" ^>nul
echo if not errorlevel 1 ^(
echo   echo Server is already running
echo   exit /b 0
echo ^)
echo.
echo REM Start ChromaDB
echo echo Starting ChromaDB...
echo start /B chroma run --path ./data/chroma --host localhost --port 8000
echo.
echo REM Wait for ChromaDB to start
echo timeout /t 3 /nobreak ^>nul
echo.
echo REM Start the server
echo echo Starting nvim-smart-keybind-search server...
echo start /B server.exe
echo.
echo echo Server started on port !PORT!
echo echo ChromaDB and server are running in background
) > start_server.bat

REM Create stop script
(
echo @echo off
echo REM Stop script for nvim-smart-keybind-search server
echo.
echo set "SCRIPT_DIR=%%~dp0"
echo cd /d "!SCRIPT_DIR!"
echo.
echo REM Stop server
echo tasklist /fi "imagename eq server.exe" 2^>nul ^| find /i "server.exe" ^>nul
echo if not errorlevel 1 ^(
echo   echo Stopping server...
echo   taskkill /f /im server.exe
echo ^) else ^(
echo   echo Server not running
echo ^)
echo.
echo REM Stop ChromaDB
echo tasklist /fi "imagename eq chroma.exe" 2^>nul ^| find /i "chroma.exe" ^>nul
echo if not errorlevel 1 ^(
echo   echo Stopping ChromaDB...
echo   taskkill /f /im chroma.exe
echo ^) else ^(
echo   echo ChromaDB not running
echo ^)
echo.
echo echo All services stopped
) > stop_server.bat

REM Create health check script
(
echo @echo off
echo REM Health check script for nvim-smart-keybind-search
echo.
echo set "SCRIPT_DIR=%%~dp0"
echo cd /d "!SCRIPT_DIR!"
echo.
echo echo Checking nvim-smart-keybind-search health...
echo.
echo REM Check if server is running
echo tasklist /fi "imagename eq server.exe" 2^>nul ^| find /i "server.exe" ^>nul
echo if not errorlevel 1 ^(
echo   echo ✓ Server is running
echo ^) else ^(
echo   echo ✗ Server is not running
echo ^)
echo.
echo REM Check if ChromaDB is running
echo tasklist /fi "imagename eq chroma.exe" 2^>nul ^| find /i "chroma.exe" ^>nul
echo if not errorlevel 1 ^(
echo   echo ✓ ChromaDB is running
echo ^) else ^(
echo   echo ✗ ChromaDB is not running
echo ^)
echo.
echo REM Check if Ollama is running
echo curl -s http://localhost:11434/api/tags ^>nul 2^>^&1
echo if not errorlevel 1 ^(
echo   echo ✓ Ollama is running
echo ^) else ^(
echo   echo ✗ Ollama is not running
echo ^)
echo.
echo REM Check database files
echo if exist "data\chroma" ^(
echo   echo ✓ Database directory exists
echo ^) else ^(
echo   echo ✗ Database directory missing
echo ^)
echo.
echo REM Check configuration
echo if exist "config.json" ^(
echo   echo ✓ Configuration file exists
echo ^) else ^(
echo   echo ✗ Configuration file missing
echo ^)
) > health_check.bat

echo ✓ Startup scripts created

REM Create plugin manager integration files
echo Creating plugin manager integration...

REM For lazy.nvim
if not exist "extras\lazy.nvim" mkdir "extras\lazy.nvim"
(
echo return {
echo   "nest/nvim-smart-keybind-search",
echo   event = "VeryLazy",
echo   config = function^(^)
echo     require^("nvim-smart-keybind-search"^).setup^({
echo       -- Configuration options
echo       server_host = "localhost",
echo       server_port = 8080,
echo       auto_start = true,
echo       health_check_interval = 30,
echo     }^)
echo   end,
echo   dependencies = {
echo     "nvim-telescope/telescope.nvim",
echo   },
echo }
) > extras\lazy.nvim\init.lua

REM For packer.nvim
if not exist "extras\packer.nvim" mkdir "extras\packer.nvim"
(
echo return require^('packer'^).startup^(function^(use^)
echo   use {
echo     'nest/nvim-smart-keybind-search',
echo     config = function^(^)
echo       require^("nvim-smart-keybind-search"^).setup^({
echo         -- Configuration options
echo         server_host = "localhost",
echo         server_port = 8080,
echo         auto_start = true,
echo         health_check_interval = 30,
echo       }^)
echo     end,
echo     requires = {
echo       'nvim-telescope/telescope.nvim',
echo     }
echo   }
echo end^)
) > extras\packer.nvim\init.lua

REM For vim-plug
if not exist "extras\vim-plug" mkdir "extras\vim-plug"
(
echo call plug#begin^(^)
echo Plug 'nest/nvim-smart-keybind-search'
echo call plug#end^(^)
echo.
echo " Configuration
echo lua require^("nvim-smart-keybind-search"^).setup^({
echo   server_host = "localhost",
echo   server_port = 8080,
echo   auto_start = true,
echo   health_check_interval = 30,
echo }^)
) > extras\vim-plug\init.vim

echo ✓ Plugin manager integration created

REM Create Windows service (using NSSM)
echo Creating Windows service...

REM Check if NSSM is available
nssm version >nul 2>&1
if errorlevel 1 (
    echo NSSM not found. Creating manual service instructions...
    
    (
    echo @echo off
    echo REM Manual service installation instructions
    echo echo To install as a Windows service:
    echo echo 1. Download NSSM from: https://nssm.cc/download
    echo echo 2. Run as administrator:
    echo echo    nssm install nvim-smart-keybind-search "%~dp0start_server.bat"
    echo echo    nssm set nvim-smart-keybind-search AppDirectory "%~dp0"
    echo echo    nssm set nvim-smart-keybind-search Description "nvim-smart-keybind-search Server"
    echo echo    nssm start nvim-smart-keybind-search
    echo pause
    ) > install_service.bat
    
    echo ✓ Manual service installation script created
) else (
    echo NSSM found. Creating automatic service installation...
    
    (
    echo @echo off
    echo REM Automatic service installation using NSSM
    echo echo Installing nvim-smart-keybind-search as Windows service...
    echo.
    echo nssm install nvim-smart-keybind-search "%~dp0start_server.bat"
    echo nssm set nvim-smart-keybind-search AppDirectory "%~dp0"
    echo nssm set nvim-smart-keybind-search Description "nvim-smart-keybind-search Server"
    echo nssm start nvim-smart-keybind-search
    echo.
    echo echo Service installed and started successfully!
    echo pause
    ) > install_service.bat
    
    echo ✓ Automatic service installation script created
)

echo ✓ Installation completed successfully!
echo.
echo Next steps:
echo 1. Install Ollama: https://ollama.ai
echo 2. Download a model: ollama pull llama3.2:3b
echo 3. Start the server: start_server.bat
echo 4. Configure your Neovim plugin manager
echo.
echo Plugin manager examples:
echo • Lazy.nvim: extras\lazy.nvim\init.lua
echo • Packer.nvim: extras\packer.nvim\init.lua
echo • vim-plug: extras\vim-plug\init.vim
echo.
echo Useful commands:
echo • Start server: start_server.bat
echo • Stop server: stop_server.bat
echo • Health check: health_check.bat
echo • Install service: install_service.bat 