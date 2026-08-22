package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"meowstore/storages"

	"github.com/golang-jwt/jwt/v5"
)

type ServiceHandler struct {
	storage   storages.Storage
	jwtSecret []byte
}

func NewServiceHandler(storage storages.Storage, secret []byte) *ServiceHandler {
	return &ServiceHandler{storage: storage, jwtSecret: secret}
}

// Parses the JWT and returns the subject (userId).
func (h *ServiceHandler) validateToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("missing token in request body")
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		if claims.Subject == "" {
			return "", errors.New("missing subject in token")
		}
		return claims.Subject, nil
	}

	return "", errors.New("invalid token")
}

// ==========================================
// PLAYLIST HANDLERS
// ==========================================

func (h *ServiceHandler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req GetPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Playlist not found")
		return
	}

	sendJSON(w, http.StatusOK, GetPlaylistResponse{Playlist: playlist})
}

func (h *ServiceHandler) GetPlaylistContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req GetPlaylistContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	// 1. Get the playlist base data
	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Playlist not found")
		return
	}

	// 2. Get the populated tracks and relations
	musics, relations, err := h.storage.GetMusicFromPlaylist(userId, req.PlaylistId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to retrieve music")
		return
	}

	if musics == nil {
		musics = []storages.Music{}
	}
	if relations == nil {
		relations = []storages.PlaylistMusic{}
	}

	sendJSON(w, http.StatusOK, GetPlaylistContentResponse{
		Playlist:  playlist,
		Musics:    musics,
		Relations: relations,
	})
}

func (h *ServiceHandler) PutPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PutPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	req.Playlist.UserId = userId
	if err := h.storage.PutPlaylist(req.Playlist); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save playlist")
		return
	}

	sendJSON(w, http.StatusOK, PutPlaylistResponse{Playlist: req.Playlist})
}

func (h *ServiceHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeletePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	if err := h.storage.DeletePlaylist(userId, req.PlaylistId); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete playlist")
		return
	}

	sendJSON(w, http.StatusOK, DeletePlaylistResponse{})
}

// ==========================================
// MUSIC HANDLERS
// ==========================================

func (h *ServiceHandler) GetMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req GetMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	music, err := h.storage.GetMusic(req.MusicId, req.Source)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Music not found")
		return
	}

	sendJSON(w, http.StatusOK, GetMusicResponse{Music: music})
}

func (h *ServiceHandler) PutMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PutMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	if err := h.storage.PutMusic(req.Music); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save music entity")
		return
	}

	sendJSON(w, http.StatusOK, PutMusicResponse{})
}

func (h *ServiceHandler) PutMusicBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PutMusicBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	for i := range req.Music {
		if err := h.storage.PutMusic(req.Music[i]); err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to save music entity")
			return
		}
	}

	sendJSON(w, http.StatusOK, PutMusicBulkResponse{})
}

// ==========================================
// PLAYLIST RELATION HANDLERS
// ==========================================

func (h *ServiceHandler) GetPlaylistsFromUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req GetPlaylistsFromUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	playlists, err := h.storage.GetPlaylistsFromUser(userId)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to fetch playlists")
		return
	}

	if playlists == nil {
		playlists = []storages.Playlist{}
	}

	sendJSON(w, http.StatusOK, GetPlaylistsFromUserResponse{Playlists: playlists})
}

func (h *ServiceHandler) PutMusicInPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PutMusicInPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	relation := storages.PlaylistMusic{
		UserId:     userId,
		PlaylistId: req.PlaylistId,
		MusicId:    req.MusicId,
		Source:     req.Source,
		AddedAt:    req.AddedAt,
	}

	if err := h.storage.PutMusicInPlaylist(relation); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save playlist relation")
		return
	}

	sendJSON(w, http.StatusOK, PutMusicInPlaylistResponse{})
}

func (h *ServiceHandler) PutMusicInPlaylistBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PutMusicInPlaylistBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	for i := range req.Relations {
		if err := h.storage.PutMusicInPlaylist(req.Relations[i]); err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to save playlist relation")
			return
		}
	}

	sendJSON(w, http.StatusOK, PutMusicInPlaylistBulkResponse{})
}

func (h *ServiceHandler) DeleteMusicFromPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeleteMusicFromPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	if err := h.storage.DeleteMusicFromPlaylist(userId, req.PlaylistId, req.MusicId, req.Source); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete music from playlist")
		return
	}

	sendJSON(w, http.StatusOK, DeleteMusicFromPlaylistResponse{})
}
