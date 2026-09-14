package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return "", fmt.Errorf("token verification failed: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		if claims.Subject == "" {
			return "", errors.New("token payload missing subject (userId)")
		}
		return claims.Subject, nil
	}

	return "", errors.New("invalid or expired token claims")
}

// ==========================================
// PLAYLIST HANDLERS
// ==========================================

func (h *ServiceHandler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.GetPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		slog.Error("failed to get playlist", "userId", userId, "playlistId", req.PlaylistId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get playlist (id=%d): %v", req.PlaylistId, err))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.GetPlaylistResponse{
		Playlist: schemas.Playlist(playlist),
	})
}

func (h *ServiceHandler) GetPlaylistContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.GetPlaylistContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		slog.Error("failed to get playlist metadata for content", "userId", userId, "playlistId", req.PlaylistId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get playlist metadata (id=%d): %v", req.PlaylistId, err))
		return
	}

	storageMusics, err := h.storage.GetAllMusic(userId, req.PlaylistId)
	if err != nil {
		slog.Error("failed to get tracks for playlist", "userId", userId, "playlistId", req.PlaylistId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve music for playlist (id=%d): %v", req.PlaylistId, err))
		return
	}

	storageRelations, err := h.storage.GetAllPlaylistMusic(userId, req.PlaylistId)
	if err != nil {
		slog.Error("failed to get track relations for playlist", "userId", userId, "playlistId", req.PlaylistId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve relations for playlist (id=%d): %v", req.PlaylistId, err))
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
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.PutPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	req.Playlist.UserId = userId
	slog.Debug("help", "a", req.Playlist)
	slog.Debug("help", "a", storages.Playlist(req.Playlist))
	if _, err := h.storage.PutPlaylist(storages.Playlist(req.Playlist)); err != nil {
		slog.Error("failed to put playlist", "userId", userId, "playlistId", req.Playlist.PlaylistId, "title", req.Playlist.Title, "modifiedDate", req.Playlist.ModifiedDate, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save playlist (id=%d, title=%q): %v", req.Playlist.PlaylistId, req.Playlist.Title, err))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistResponse{Playlist: req.Playlist})
}

func (h *ServiceHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.DeletePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	if err := h.storage.DeletePlaylist(userId, req.PlaylistId); err != nil {
		slog.Error("failed to delete playlist", "userId", userId, "playlistId", req.PlaylistId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete playlist (id=%d): %v", req.PlaylistId, err))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.DeletePlaylistResponse{})
}

// ==========================================
// MUSIC HANDLERS
// ==========================================

func (h *ServiceHandler) GetMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.GetMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	music, err := h.storage.GetMusic(req.MusicId, req.Source)
	if err != nil {
		slog.Error("failed to get music", "musicId", req.MusicId, "source", req.Source, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get music (id=%s, source=%d): %v", req.MusicId, req.Source, err))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.GetMusicResponse{Music: schemas.Music(music)})
}

func (h *ServiceHandler) PutMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.PutMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	if err := h.storage.PutMusic(storages.Music(req.Music)); err != nil {
		slog.Error("failed to put music entity", "musicId", req.Music.MusicId, "source", req.Music.Source, "title", req.Music.Title, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save music entity (id=%s, source=%d, title=%q): %v", req.Music.MusicId, req.Music.Source, req.Music.Title, err))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutMusicResponse{})
}

func (h *ServiceHandler) PutMusicBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.PutMusicBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	if _, err := h.validateToken(req.Token); err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	for i := range req.Music {
		if err := h.storage.PutMusic(storages.Music(req.Music[i])); err != nil {
			slog.Error("failed to put music entity in bulk batch",
				"index", i,
				"musicId", req.Music[i].MusicId,
				"source", req.Music[i].Source,
				"title", req.Music[i].Title,
				"error", err,
			)
			schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf(
				"Failed to save music entity at index %d (id=%s, source=%d, title=%q): %v",
				i, req.Music[i].MusicId, req.Music[i].Source, req.Music[i].Title, err,
			))
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
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.GetPlaylistsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	playlists, err := h.storage.GetPlaylists(userId)
	if err != nil {
		slog.Error("failed to fetch user playlists", "userId", userId, "error", err)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch playlists for user %s: %v", userId, err))
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
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.PutPlaylistMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	req.PlaylistMusic.UserId = userId

	if err := h.storage.PutPlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic)); err != nil {
		slog.Error("failed to put playlist music relation",
			"userId", userId,
			"playlistId", req.PlaylistMusic.PlaylistId,
			"musicId", req.PlaylistMusic.MusicId,
			"source", req.PlaylistMusic.Source,
			"error", err,
		)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf(
			"Failed to save playlist relation (playlistId=%d, musicId=%s, source=%d): %v",
			req.PlaylistMusic.PlaylistId, req.PlaylistMusic.MusicId, req.PlaylistMusic.Source, err,
		))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistMusicResponse{})
}

func (h *ServiceHandler) PutPlaylistMusicBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.PutPlaylistMusicBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	for i := range req.PlaylistMusic {
		req.PlaylistMusic[i].UserId = userId

		if err := h.storage.PutPlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic[i])); err != nil {
			slog.Error("failed to put playlist music relation in bulk batch",
				"index", i,
				"userId", userId,
				"playlistId", req.PlaylistMusic[i].PlaylistId,
				"musicId", req.PlaylistMusic[i].MusicId,
				"source", req.PlaylistMusic[i].Source,
				"error", err,
			)
			schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf(
				"Failed to save playlist relation at index %d (playlistId=%d, musicId=%s, source=%d): %v",
				i, req.PlaylistMusic[i].PlaylistId, req.PlaylistMusic[i].MusicId, req.PlaylistMusic[i].Source, err,
			))
			return
		}
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.PutPlaylistMusicBulkResponse{})
}

func (h *ServiceHandler) DeletePlaylistMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schemas.ReplyError(w, http.StatusMethodNotAllowed, "Method not allowed: must be POST")
		return
	}

	var req schemas.DeletePlaylistMusicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.ReplyError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
		return
	}

	userId, err := h.validateToken(req.Token)
	if err != nil {
		schemas.ReplyError(w, http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	req.PlaylistMusic.UserId = userId

	if err := h.storage.DeletePlaylistMusic(storages.PlaylistMusic(req.PlaylistMusic)); err != nil {
		slog.Error("failed to delete playlist music relation",
			"userId", userId,
			"playlistId", req.PlaylistMusic.PlaylistId,
			"musicId", req.PlaylistMusic.MusicId,
			"source", req.PlaylistMusic.Source,
			"error", err,
		)
		schemas.ReplyError(w, http.StatusInternalServerError, fmt.Sprintf(
			"Failed to delete playlist music relation (playlistId=%d, musicId=%s, source=%d): %v",
			req.PlaylistMusic.PlaylistId, req.PlaylistMusic.MusicId, req.PlaylistMusic.Source, err,
		))
		return
	}

	schemas.ReplyJSON(w, http.StatusOK, schemas.DeletePlaylistMusicResponse{})
}
