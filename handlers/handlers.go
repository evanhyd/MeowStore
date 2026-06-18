package handlers

import (
	"encoding/json"
	"log"
	"meowstore/storages"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	Store storages.Storage
	Auth  *AuthHandler
}

type PlaylistDetailResponse struct {
	Playlist      storages.Playlist        `json:"playlist"`
	Music         []storages.Music         `json:"tracks"`
	PlaylistMusic []storages.PlaylistMusic `json:"metadata"`
}

// --- Common Helper Functions ---

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

// extractPlaylistID simplifies the repeated URL parsing logic
func extractPlaylistID(r *http.Request) (int64, error) {
	idStr := strings.TrimPrefix(r.URL.Path, "/playlist/")
	if idStr == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(idStr, 10, 64)
}

// --- The Auth Wrapper ---

func (h *Handler) WithAuth(handler func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := h.Auth.getUserIDFromHeader(r)
		if err != nil {
			sendError(w, err.Error(), http.StatusUnauthorized)
			return
		}
		handler(w, r, userID)
	}
}

// --- Handlers ---

// GetAllPlaylists retrieves all playlists for the authenticated user.
func (h *Handler) GetAllPlaylists(w http.ResponseWriter, r *http.Request, userID string) {
	playlists, err := h.Store.GetPlaylistsFromUser(userID)
	if err != nil {
		sendError(w, "Failed to retrieve playlists", http.StatusInternalServerError)
		return
	}

	if playlists == nil {
		playlists = []storages.Playlist{}
	}

	sendJSON(w, http.StatusOK, playlists)
}

// GetPlaylist retrieves a specific playlist.
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request, userID string) {
	playlistID, err := extractPlaylistID(r)
	if err != nil {
		sendError(w, "Invalid playlist ID format", http.StatusBadRequest)
		return
	}

	playlist, err := h.Store.GetPlaylist(userID, playlistID)
	if err != nil {
		sendError(w, "Playlist not found", http.StatusNotFound)
		return
	}

	tracks, err := h.Store.GetMusicFromPlaylist(userID, playlistID)
	if err != nil {
		sendError(w, "Failed to retrieve tracks", http.StatusInternalServerError)
		return
	}

	metadata, err := h.Store.GetPlaylistMusicFromPlaylist(userID, playlistID)
	if err != nil {
		sendError(w, "Failed to retrieve playlist metadata", http.StatusInternalServerError)
		return
	}

	if tracks == nil {
		tracks = []storages.Music{}
	}
	if metadata == nil {
		metadata = []storages.PlaylistMusic{}
	}

	sendJSON(w, http.StatusOK, PlaylistDetailResponse{
		Playlist:      playlist,
		Music:         tracks,
		PlaylistMusic: metadata,
	})
}

// PutPlaylist creates or overwrites the playlist metadata.
func (h *Handler) PutPlaylist(w http.ResponseWriter, r *http.Request, userID string) {
	playlistID, err := extractPlaylistID(r)
	if err != nil {
		sendError(w, "Invalid playlist ID format", http.StatusBadRequest)
		return
	}

	var req storages.Playlist
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	req.UserId = userID
	req.PlaylistId = playlistID

	if err := h.Store.PutPlaylist(req); err != nil {
		sendError(w, "Failed to save playlist", http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DeletePlaylist removes the playlist from storage.
func (h *Handler) DeletePlaylist(w http.ResponseWriter, r *http.Request, userID string) {
	playlistID, err := extractPlaylistID(r)
	if err != nil {
		sendError(w, "Invalid playlist ID format", http.StatusBadRequest)
		return
	}

	if err := h.Store.DeletePlaylist(userID, playlistID); err != nil {
		sendError(w, "Failed to delete playlist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Routing ---

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	// Wrap GetAllPlaylists
	mux.HandleFunc("/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Apply the wrapper
		h.WithAuth(h.GetAllPlaylists)(w, r)
	})

	// Wrap specific playlist operations
	mux.HandleFunc("/playlist/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.WithAuth(h.GetPlaylist)(w, r)
		case http.MethodPut:
			h.WithAuth(h.PutPlaylist)(w, r)
		case http.MethodDelete:
			h.WithAuth(h.DeletePlaylist)(w, r)
		default:
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
