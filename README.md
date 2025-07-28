# Nvim Smart Keybind Search

A neovim plugin that provides intelligent, natural language search for keybindings and vim motions using RAG-based search.

## Project Structure

```
.
├── cmd/
│   └── server/          # Go backend service entry point
│       └── main.go
├── internal/
│   ├── interfaces/      # Core interface definitions
│   │   ├── rag.go      # RAG agent interface
│   │   ├── vectordb.go # Vector database interface
│   │   └── llm.go      # LLM client interface
│   └── server/         # JSON-RPC server implementation
│       └── rpc.go
├── go.mod
└── README.md
```

## Core Interfaces

- **RAGAgent**: Handles query processing and vector database updates
- **VectorDB**: Manages vector storage and semantic search operations  
- **LLMClient**: Provides language model inference and embedding generation

## JSON-RPC API

The service exposes the following RPC methods:

- `Query`: Process natural language queries for keybinding search
- `UpdateKeybindings`: Update the vector database with new keybindings
- `HealthCheck`: Check the health status of all service components

## Development

To run the server:

```bash
go run cmd/server/main.go
```

The server will start on port 8080 by default, or use the PORT environment variable.