package handlers

import (
	"encoding/json"
	"meowstore/storages"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// GetPlaylist (Lightweight: Returns only the playlist metadata)
type GetPlaylistRequest struct {
	Token      string `json:"token"`
	PlaylistId int64  `json:"playlistId"`
}
type GetPlaylistResponse struct {
	Playlist storages.Playlist `json:"playlist"`
}

// GetPlaylistContent (Heavyweight: Returns playlist, musics, and relations)
type GetPlaylistContentRequest struct {
	Token      string `json:"token"`
	PlaylistId int64  `json:"playlistId"`
}
type GetPlaylistContentResponse struct {
	Playlist  storages.Playlist        `json:"playlist"`
	Musics    []storages.Music         `json:"musics"`
	Relations []storages.PlaylistMusic `json:"relations"`
}

// PutPlaylist
type PutPlaylistRequest struct {
	Token    string            `json:"token"`
	Playlist storages.Playlist `json:"playlist"`
}
type PutPlaylistResponse struct {
	Playlist storages.Playlist `json:"playlist"`
}

// DeletePlaylist
type DeletePlaylistRequest struct {
	Token      string `json:"token"`
	PlaylistId int64  `json:"playlistId"`
}
type DeletePlaylistResponse struct {
}

// GetMusic
type GetMusicRequest struct {
	Token   string               `json:"token"`
	MusicId string               `json:"musicId"`
	Source  storages.MusicSource `json:"source"`
}
type GetMusicResponse struct {
	Music storages.Music `json:"music"`
}

// PutMusic
type PutMusicRequest struct {
	Token string         `json:"token"`
	Music storages.Music `json:"music"`
}
type PutMusicResponse struct {
}

// PutMusicBulk
type PutMusicBulkRequest struct {
	Token string           `json:"token"`
	Music []storages.Music `json:"music"`
}
type PutMusicBulkResponse struct {
}

// GetPlaylistsFromUser
type GetPlaylistsFromUserRequest struct {
	Token string `json:"token"`
}
type GetPlaylistsFromUserResponse struct {
	Playlists []storages.Playlist `json:"playlists"`
}

// GetMusicFromPlaylist
type GetMusicFromPlaylistRequest struct {
	Token      string `json:"token"`
	PlaylistId int64  `json:"playlistId"`
}
type GetMusicFromPlaylistResponse struct {
	Musics []storages.Music `json:"musics"`
}

// PutMusicInPlaylist
type PutMusicInPlaylistRequest struct {
	Token      string               `json:"token"`
	PlaylistId int64                `json:"playlistId"`
	MusicId    string               `json:"musicId"`
	Source     storages.MusicSource `json:"source"`
	AddedAt    int64                `json:"addedAt"` // unix nano
}
type PutMusicInPlaylistResponse struct {
}

// PutMusicInPlaylistBulk
type PutMusicInPlaylistBulkRequest struct {
	Token     string                   `json:"token"`
	Relations []storages.PlaylistMusic `json:"relations"`
}
type PutMusicInPlaylistBulkResponse struct {
}

// DeleteMusicFromPlaylist
type DeleteMusicFromPlaylistRequest struct {
	Token      string               `json:"token"`
	PlaylistId int64                `json:"playlistId"`
	MusicId    string               `json:"musicId"`
	Source     storages.MusicSource `json:"source"`
}
type DeleteMusicFromPlaylistResponse struct {
}

func sendJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func sendError(w http.ResponseWriter, code int, msg string) {
	sendJSON(w, code, ErrorResponse{Error: msg})
}
