package main

import (
	"flag"
	"log/slog"
	"meowstore/handlers"
	"meowstore/loggers"
	"meowstore/storages"
	"net/http"
	"os"
)

func main() {
	logFlag := flag.String("log", "", "The log file path.")
	jwtKeyFlag := flag.String("key", "", "The JWT key file path.")
	dbFlag := flag.String("db", "", "The database file path.")
	portFlag := flag.String("port", "80", "The server port. Default to 80.")
	flag.Parse()

	// Logger.
	logger := loggers.InitializeGlobalLogger(*logFlag)
	defer logger.Close()

	// SQL storage.
	storage := storages.NewSQLiteStorage(*dbFlag)
	if storage == nil {
		return
	}
	defer storage.Close()

	// JWT key.
	jwtKey, err := os.ReadFile(*jwtKeyFlag)
	if err != nil {
		slog.Error("failed to read jwt key", "error", err)
		return
	}
	service := handlers.NewServiceHandler(storage, jwtKey)

	// Start the server.
	mux := http.NewServeMux()
	mux.HandleFunc("/playlist/meta", service.GetPlaylistsMeta)
	mux.HandleFunc("/playlist/get", service.GetPlaylist)
	mux.HandleFunc("/playlist/put", service.PutPlaylist)

	addr := ":" + *portFlag
	slog.Info("Server is starting", "port", *portFlag)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server crashed or failed to start", "error", err)
	}
}
