package handlers

import (
	"encoding/json"
	"log"
	"meowstore/storages"
	"net/http"
)

type GetPlaylistsMetaRequest struct {
}

type GetPlaylistsMetaResponse struct {
	PlaylistsMeta []storages.PlaylistMeta `json:"playlists_meta"`
}

type GetPlaylistRequest struct {
	PlaylistId int64 `json:"playlist_id"`
}

type GetPlaylistResponse struct {
	Playlist  storages.Playlist        `json:"playlist"`
	Musics    []storages.Music         `json:"musics"`
	Relations []storages.PlaylistMusic `json:"relations"`
}

type PutPlaylistRequest struct {
	Playlist  storages.Playlist        `json:"playlist"`
	Musics    []storages.Music         `json:"musics"`
	Relations []storages.PlaylistMusic `json:"relations"`
}

type PutPlaylistResponse struct {
}

func sendJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func sendError(w http.ResponseWriter, msg string, code int) {
	sendJSON(w, code, map[string]string{"error": msg})
}
