package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"

	"nvim-smart-keybind-search/internal/server"
)

func main() {
	// Create RPC server
	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(json.NewCodec(), "application/json")

	// TODO: Initialize actual implementations of interfaces in future tasks
	// For now, create service with nil dependencies for basic structure
	service := server.NewRPCService(nil, nil, nil)
	
	err := rpcServer.RegisterService(service, "")
	if err != nil {
		log.Fatal("Error registering RPC service:", err)
	}

	// Set up HTTP handler
	http.Handle("/rpc", rpcServer)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting JSON-RPC server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}