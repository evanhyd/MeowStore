package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"meowstore/schemas"
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
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.GetPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Playlist not found")
		return
	}

	// Map storage to schema
	schemas.ReplyJSON(w, http.StatusOK, schemas.GetPlaylistResponse{
		Playlist: schemas.Playlist(playlist),
	})
}

func (h *ServiceHandler) GetPlaylistContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.GetPlaylistContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Playlist not found")
		return
	}

	storageMusics, err := h.storage.GetAllMusic(userId, req.PlaylistId)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to retrieve music")
		return
	}

	storageRelations, err := h.storage.GetAllPlaylistMusic(userId, req.PlaylistId)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to retrieve playlist relations")
		return
	}

	schemaMusics := make([]schemas.Music, len(storageMusics))
	for i, m := range storageMusics {
		schemaMusics[i] = schemas.Music(m)
	}

	schemaRelations := make([]schemas.PlaylistMusic, len(storageRelations))
	for i, r := range storageRelations {
		schemaRelations[i] = schemas.PlaylistMusic(r)
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.GetPlaylistContentResponse{
		Playlist:  schemas.Playlist(playlist),
		Musics:    schemaMusics,
		Relations: schemaRelations,
	})
}

func (h *ServiceHandler) PutPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.PutPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	req.Playlist.UserId = userId
	if _, err := h.storage.PutPlaylist(storages.Playlist(req.Playlist)); err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to save playlist")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistResponse{Playlist: req.Playlist})
}

func (h *ServiceHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.DeletePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	if err := h.storage.DeletePlaylist(userId, req.PlaylistId); err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to delete playlist")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.DeletePlaylistResponse{})
}

// ==========================================
// MUSIC HANDLERS
// ==========================================

func (h *ServiceHandler) GetMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.GetMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	music, err := h.storage.GetMusic(req.MusicId, req.Source)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Music not found")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.GetMusicResponse{Music: schemas.Music(music)})
}

func (h *ServiceHandler) PutMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.PutMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	if err := h.storage.PutMusic(storages.Music(req.Music)); err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to save music entity")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutMusicResponse{})
}

func (h *ServiceHandler) PutMusicBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.PutMusicBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	for i := range req.Music {
		if err := h.storage.PutMusic(storages.Music(req.Music[i])); err != nil {
			schemas.ReplyError(w, http.StatusInternalServerError, "Failed to save music entity")
			return
		}
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutMusicBulkResponse{})
}

// ==========================================
// PLAYLIST RELATION HANDLERS
// ==========================================

func (h *ServiceHandler) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.GetPlaylistsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	playlists, err := h.storage.GetPlaylists(userId)
	if err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to fetch playlists")
		return
	}

	schemaPlaylists := make([]schemas.Playlist, len(playlists))
	for i, p := range playlists {
		schemaPlaylists[i] = schemas.Playlist(p)
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.GetPlaylistsResponse{Playlists: schemaPlaylists})
}

func (h *ServiceHandler) PutPlaylistMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.PutPlaylistMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	req.PlaylistMusic.UserId = userId

	if err := h.storage.PutPlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic)); err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to save playlist relation")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistMusicResponse{})
}

func (h *ServiceHandler) PutPlaylistMusicBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.PutPlaylistMusicBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	for i := range req.PlaylistMusic {
		req.PlaylistMusic[i].UserId = userId

		if err := h.storage.PutPlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic[i])); err != nil {
			schemas.ReplyError(w, http.StatusInternalServerError, "Failed to save playlist relation")
			return
		}
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistMusicBulkResponse{})
}

func (h *ServiceHandler) DeletePlaylistMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req schemas.DeletePlaylistMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
		return
	}

	req.PlaylistMusic.UserId = userId

	if err := h.storage.DeletePlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic)); err != nil {
		schemas.ReplyError(w, http.StatusInternalServerError, "Failed to delete music from playlist")
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.DeletePlaylistMusicResponse{})
}
