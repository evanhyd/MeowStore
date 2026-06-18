package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meowstore/storages"

	"github.com/golang-jwt/jwt/v5"
)

// --- Mock Storage ---

type MockStorage struct {
	storages.Storage

	GetPlaylistsFromUserFunc         func(userId string) ([]storages.Playlist, error)
	GetPlaylistFunc                  func(userId string, playlistID int64) (storages.Playlist, error)
	GetMusicFromPlaylistFunc         func(userId string, playlistID int64) ([]storages.Music, error)
	GetPlaylistMusicFromPlaylistFunc func(userId string, playlistID int64) ([]storages.PlaylistMusic, error)
	PutPlaylistFunc                  func(playlist storages.Playlist) error
	DeletePlaylistFunc               func(userId string, playlistID int64) error
}

func (m *MockStorage) GetPlaylistsFromUser(userId string) ([]storages.Playlist, error) {
	return m.GetPlaylistsFromUserFunc(userId)
}
func (m *MockStorage) GetPlaylist(userId string, playlistID int64) (storages.Playlist, error) {
	return m.GetPlaylistFunc(userId, playlistID)
}
func (m *MockStorage) GetMusicFromPlaylist(userId string, playlistID int64) ([]storages.Music, error) {
	return m.GetMusicFromPlaylistFunc(userId, playlistID)
}
func (m *MockStorage) GetPlaylistMusicFromPlaylist(userId string, playlistID int64) ([]storages.PlaylistMusic, error) {
	return m.GetPlaylistMusicFromPlaylistFunc(userId, playlistID)
}
func (m *MockStorage) PutPlaylist(playlist storages.Playlist) error {
	return m.PutPlaylistFunc(playlist)
}
func (m *MockStorage) DeletePlaylist(userId string, playlistID int64) error {
	return m.DeletePlaylistFunc(userId, playlistID)
}

// --- Test Helpers ---

var testJWTSecret = []byte("test_secret_key")

func createTestAuthHeader(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(testJWTSecret)
	return "Bearer " + tokenString
}

// createWrongSignatureHeader generates a token signed with an invalid key
func createWrongSignatureHeader(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("fake_hacker_key"))
	return "Bearer " + tokenString
}

func setupTestHandler() (*Handler, *MockStorage) {
	mockStore := &MockStorage{}
	authHandler := &AuthHandler{
		jwtSecret: testJWTSecret,
	}
	h := &Handler{
		Store: mockStore,
		Auth:  authHandler,
	}
	return h, mockStore
}

// --- Edge Case Tests ---

func TestAuthEdgeCases(t *testing.T) {
	h, _ := setupTestHandler()

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{"Missing Header", "", http.StatusUnauthorized},
		{"Malformed Prefix", "Basic xyz123", http.StatusUnauthorized},
		{"Missing Token", "Bearer ", http.StatusUnauthorized},
		{"Fake/Wrong Signature", createWrongSignatureHeader("user_123"), http.StatusUnauthorized},
		{"Completely Invalid Token", "Bearer not.a.real.token", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/playlists", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			h.WithAuth(h.GetAllPlaylists)(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetAllPlaylistsEdgeCases(t *testing.T) {
	h, mockStore := setupTestHandler()
	userID := "user_123"

	tests := []struct {
		name           string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name: "Success Empty List",
			mockSetup: func() {
				mockStore.GetPlaylistsFromUserFunc = func(uid string) ([]storages.Playlist, error) {
					return nil, nil // Ensures it returns [] instead of null
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Database Error",
			mockSetup: func() {
				mockStore.GetPlaylistsFromUserFunc = func(uid string) ([]storages.Playlist, error) {
					return nil, errors.New("db connection lost")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			req := httptest.NewRequest(http.MethodGet, "/playlists", nil)
			req.Header.Set("Authorization", createTestAuthHeader(userID))
			w := httptest.NewRecorder()

			h.WithAuth(h.GetAllPlaylists)(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetPlaylistEdgeCases(t *testing.T) {
	h, mockStore := setupTestHandler()
	userID := "user_123"

	tests := []struct {
		name           string
		targetURL      string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:           "Invalid ID Format (Letters)",
			targetURL:      "/playlist/abc",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing ID",
			targetURL:      "/playlist/",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Playlist Not Found (404)",
			targetURL: "/playlist/99",
			mockSetup: func() {
				mockStore.GetPlaylistFunc = func(uid string, pid int64) (storages.Playlist, error) {
					return storages.Playlist{}, errors.New("not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "DB Error on Joined Data",
			targetURL: "/playlist/42",
			mockSetup: func() {
				mockStore.GetPlaylistFunc = func(uid string, pid int64) (storages.Playlist, error) {
					return storages.Playlist{}, nil
				}
				mockStore.GetMusicFromPlaylistFunc = func(uid string, pid int64) ([]storages.Music, error) {
					return nil, errors.New("join failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			req := httptest.NewRequest(http.MethodGet, tt.targetURL, nil)
			req.Header.Set("Authorization", createTestAuthHeader(userID))
			w := httptest.NewRecorder()

			h.WithAuth(h.GetPlaylist)(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestPutPlaylistEdgeCases(t *testing.T) {
	h, mockStore := setupTestHandler()
	userID := "user_123"

	tests := []struct {
		name           string
		targetURL      string
		bodyPayload    any // Use string for raw JSON testing, struct for auto-marshal
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:           "Invalid ID Format",
			targetURL:      "/playlist/xyz",
			bodyPayload:    storages.Playlist{Title: "Valid Title"},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Malformed JSON Body",
			targetURL:      "/playlist/42",
			bodyPayload:    `{ "title": "Missing quotes closing }`, // Invalid JSON string
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Storage Write Failure",
			targetURL:   "/playlist/42",
			bodyPayload: storages.Playlist{Title: "Good Title"},
			mockSetup: func() {
				mockStore.PutPlaylistFunc = func(p storages.Playlist) error {
					return errors.New("disk full")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var bodyBytes []byte
			switch v := tt.bodyPayload.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, tt.targetURL, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Authorization", createTestAuthHeader(userID))
			w := httptest.NewRecorder()

			h.WithAuth(h.PutPlaylist)(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDeletePlaylistEdgeCases(t *testing.T) {
	h, mockStore := setupTestHandler()
	userID := "user_123"

	tests := []struct {
		name           string
		targetURL      string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:           "Invalid ID Format",
			targetURL:      "/playlist/DROP_TABLE",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Storage Delete Failure",
			targetURL: "/playlist/42",
			mockSetup: func() {
				mockStore.DeletePlaylistFunc = func(uid string, pid int64) error {
					return errors.New("locked row")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			req := httptest.NewRequest(http.MethodDelete, tt.targetURL, nil)
			req.Header.Set("Authorization", createTestAuthHeader(userID))
			w := httptest.NewRecorder()

			h.WithAuth(h.DeletePlaylist)(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
