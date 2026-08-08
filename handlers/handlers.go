package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"meowstore/storages"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ServiceHandler struct {
	storage   storages.Storage
	jwtSecret []byte
}

func NewServiceHandler(storage storages.Storage, secret []byte) *ServiceHandler {
	return &ServiceHandler{storage: storage, jwtSecret: secret}
}

func (h *ServiceHandler) extractUserID(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	tokenString := parts[1]

	// jwt.ParseWithClaims automatically validates the expiration date if present in RegisteredClaims
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

func (h *ServiceHandler) GetPlaylistsMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userId, err := h.extractUserID(r)
	if err != nil {
		sendError(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Decode the body for consistency, ignoring EOF if the body is completely empty
	var req GetPlaylistsMetaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	metas, err := h.storage.GetPlaylistsMetaFromUser(userId)
	if err != nil {
		sendError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if metas == nil {
		metas = []storages.PlaylistMeta{}
	}

	sendJSON(w, http.StatusOK, GetPlaylistsMetaResponse{PlaylistsMeta: metas})
}

func (h *ServiceHandler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userId, err := h.extractUserID(r)
	if err != nil {
		sendError(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req GetPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	playlist, err := h.storage.GetPlaylist(userId, req.PlaylistId)
	if err != nil {
		sendError(w, "Playlist not found or database error", http.StatusInternalServerError)
		return
	}

	musics, relations, err := h.storage.GetMusicFromPlaylist(userId, req.PlaylistId)
	if err != nil {
		sendError(w, "Failed to retrieve music", http.StatusInternalServerError)
		return
	}

	if musics == nil {
		musics = []storages.Music{}
	}
	if relations == nil {
		relations = []storages.PlaylistMusic{}
	}

	sendJSON(w, http.StatusOK, GetPlaylistResponse{
		Playlist:  playlist,
		Musics:    musics,
		Relations: relations,
	})
}

func (h *ServiceHandler) PutPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userId, err := h.extractUserID(r)
	if err != nil {
		sendError(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req PutPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Playlist.UserId = userId
	if err := h.storage.PutPlaylist(req.Playlist); err != nil {
		sendError(w, "Failed to save playlist", http.StatusInternalServerError)
		return
	}

	for _, music := range req.Musics {
		if err := h.storage.PutMusic(music); err != nil {
			sendError(w, "Failed to save music", http.StatusInternalServerError)
			return
		}
	}

	for _, relation := range req.Relations {
		relation.UserId = userId
		relation.PlaylistId = req.Playlist.PlaylistId
		if err := h.storage.PutMusicInPlaylist(relation); err != nil {
			sendError(w, "Failed to save playlist relation", http.StatusInternalServerError)
			return
		}
	}

	sendJSON(w, http.StatusOK, PutPlaylistResponse{})
}
