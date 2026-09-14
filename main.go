package main

import (
	"flag"
	"log/slog"
	"meowstore/handlers"
	"meowstore/loggers"
	"meowstore/storages"
	"net/http"
	"os"
	"runtime/debug"
	"time"
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

	// Panic handler.
	defer func() {
		if e := recover(); e != nil {
			slog.Error("server crashed",
				"error", e,
				"stack", string(debug.Stack()),
			)
		}
	}()

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
	mux.HandleFunc("POST /api/getPlaylist", service.GetPlaylist)
	mux.HandleFunc("POST /api/getPlaylistContent", service.GetPlaylistContent)
	mux.HandleFunc("POST /api/putPlaylist", service.PutPlaylist)
	mux.HandleFunc("POST /api/deletePlaylist", service.DeletePlaylist)
	mux.HandleFunc("POST /api/getMusic", service.GetMusic)
	mux.HandleFunc("POST /api/putMusic", service.PutMusic)
	mux.HandleFunc("POST /api/putMusicBulk", service.PutMusicBulk)
	mux.HandleFunc("POST /api/getPlaylists", service.GetPlaylists)
	mux.HandleFunc("POST /api/putPlaylistMusic", service.PutPlaylistMusic)
	mux.HandleFunc("POST /api/putPlaylistMusicBulk", service.PutPlaylistMusicBulk)
	mux.HandleFunc("POST /api/deletePlaylistMusic", service.DeletePlaylistMusic)

	slog.Info("Server is starting", "port", *portFlag)
	addr := ":" + *portFlag
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server crashed or failed to start", "error", err)
	}
}
