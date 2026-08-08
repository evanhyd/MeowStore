package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"meowstore/storages"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockStorage implements the storages.Storage interface for testing purposes.
type MockStorage struct {
	// PlaylistAccessor functions
	PutPlaylistFunc              func(playlist storages.Playlist) error
	GetPlaylistFunc              func(userId string, playlistId int64) (storages.Playlist, error)
	GetPlaylistsFromUserFunc     func(userId string) ([]storages.Playlist, error)
	GetPlaylistsMetaFromUserFunc func(userId string) ([]storages.PlaylistMeta, error)
	DeletePlaylistFunc           func(userId string, playlistId int64) error

	// MusicAccessor functions
	PutMusicFunc    func(music storages.Music) error
	GetMusicFunc    func(musicId string, source storages.MusicSource) (storages.Music, error)
	DeleteMusicFunc func(musicId string, source storages.MusicSource) error

	// PlaylistRelationAccessor functions
	PutMusicInPlaylistFunc      func(pm storages.PlaylistMusic) error
	GetMusicFromPlaylistFunc    func(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error)
	DeleteMusicFromPlaylistFunc func(userId string, playlistId int64, musicId string, source storages.MusicSource) error

	// io.Closer function
	CloseFunc func() error
}

// --- PlaylistAccessor Implementation ---

func (m *MockStorage) PutPlaylist(playlist storages.Playlist) error {
	if m.PutPlaylistFunc != nil {
		return m.PutPlaylistFunc(playlist)
	}
	return nil
}

func (m *MockStorage) GetPlaylist(userId string, playlistId int64) (storages.Playlist, error) {
	if m.GetPlaylistFunc != nil {
		return m.GetPlaylistFunc(userId, playlistId)
	}
	return storages.Playlist{}, nil
}

func (m *MockStorage) GetPlaylistsFromUser(userId string) ([]storages.Playlist, error) {
	if m.GetPlaylistsFromUserFunc != nil {
		return m.GetPlaylistsFromUserFunc(userId)
	}
	return nil, nil
}

func (m *MockStorage) GetPlaylistsMetaFromUser(userId string) ([]storages.PlaylistMeta, error) {
	if m.GetPlaylistsMetaFromUserFunc != nil {
		return m.GetPlaylistsMetaFromUserFunc(userId)
	}
	return nil, nil
}

func (m *MockStorage) DeletePlaylist(userId string, playlistId int64) error {
	if m.DeletePlaylistFunc != nil {
		return m.DeletePlaylistFunc(userId, playlistId)
	}
	return nil
}

// --- MusicAccessor Implementation ---

func (m *MockStorage) PutMusic(music storages.Music) error {
	if m.PutMusicFunc != nil {
		return m.PutMusicFunc(music)
	}
	return nil
}

func (m *MockStorage) GetMusic(musicId string, source storages.MusicSource) (storages.Music, error) {
	if m.GetMusicFunc != nil {
		return m.GetMusicFunc(musicId, source)
	}
	return storages.Music{}, nil
}

func (m *MockStorage) DeleteMusic(musicId string, source storages.MusicSource) error {
	if m.DeleteMusicFunc != nil {
		return m.DeleteMusicFunc(musicId, source)
	}
	return nil
}

// --- PlaylistRelationAccessor Implementation ---

func (m *MockStorage) PutMusicInPlaylist(pm storages.PlaylistMusic) error {
	if m.PutMusicInPlaylistFunc != nil {
		return m.PutMusicInPlaylistFunc(pm)
	}
	return nil
}

func (m *MockStorage) GetMusicFromPlaylist(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error) {
	if m.GetMusicFromPlaylistFunc != nil {
		return m.GetMusicFromPlaylistFunc(userId, playlistId)
	}
	return nil, nil, nil
}

func (m *MockStorage) DeleteMusicFromPlaylist(userId string, playlistId int64, musicId string, source storages.MusicSource) error {
	if m.DeleteMusicFromPlaylistFunc != nil {
		return m.DeleteMusicFromPlaylistFunc(userId, playlistId, musicId, source)
	}
	return nil
}

// --- io.Closer Implementation ---

func (m *MockStorage) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Helper to generate JWT tokens for tests
func generateTestToken(subject string, secret []byte, exp time.Time) string {
	claims := &jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

var testSecret = []byte("super-secret-key")

func TestExtractUserID(t *testing.T) {
	handler := NewServiceHandler(nil, testSecret)

	validToken := generateTestToken("user-123", testSecret, time.Now().Add(1*time.Hour))
	expiredToken := generateTestToken("user-123", testSecret, time.Now().Add(-1*time.Hour))
	emptySubToken := generateTestToken("", testSecret, time.Now().Add(1*time.Hour))

	tests := []struct {
		name        string
		authHeader  string
		wantSubject string
		expectErr   bool
	}{
		{"Valid Token", "Bearer " + validToken, "user-123", false},
		{"Missing Header", "", "", true},
		{"Invalid Format", validToken, "", true}, // Missing "Bearer "
		{"Expired Token", "Bearer " + expiredToken, "", true},
		{"Empty Subject", "Bearer " + emptySubToken, "", true},
		{"Invalid Signature", "Bearer " + validToken + "bad", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			subject, err := handler.extractUserID(req)
			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectErr, err)
			}
			if subject != tc.wantSubject {
				t.Errorf("expected subject %s, got %s", tc.wantSubject, subject)
			}
		})
	}
}

func TestGetPlaylistsMeta(t *testing.T) {
	mockStorage := &MockStorage{
		GetPlaylistsMetaFromUserFunc: func(userId string) ([]storages.PlaylistMeta, error) {
			if userId == "user-error" {
				return nil, errors.New("db error")
			}
			return []storages.PlaylistMeta{{PlaylistId: 1}}, nil
		},
	}
	handler := NewServiceHandler(mockStorage, testSecret)

	t.Run("Success", func(t *testing.T) {
		token := generateTestToken("user-123", testSecret, time.Now().Add(1*time.Hour))
		req := httptest.NewRequest(http.MethodPost, "/meta", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.GetPlaylistsMeta(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		var resp GetPlaylistsMetaResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.PlaylistsMeta) != 1 || resp.PlaylistsMeta[0].PlaylistId != 1 {
			t.Errorf("unexpected response body: %+v", resp)
		}
	})

	t.Run("Wrong Method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/meta", nil)
		w := httptest.NewRecorder()
		handler.GetPlaylistsMeta(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})

	t.Run("DB Error", func(t *testing.T) {
		token := generateTestToken("user-error", testSecret, time.Now().Add(1*time.Hour))
		req := httptest.NewRequest(http.MethodPost, "/meta", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.GetPlaylistsMeta(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestGetPlaylist(t *testing.T) {
	mockStorage := &MockStorage{
		GetPlaylistFunc: func(userId string, playlistId int64) (storages.Playlist, error) {
			if playlistId == 999 {
				return storages.Playlist{}, errors.New("not found")
			}
			return storages.Playlist{PlaylistId: playlistId, UserId: userId}, nil
		},
		GetMusicFromPlaylistFunc: func(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error) {
			return []storages.Music{{MusicId: "m1"}}, []storages.PlaylistMusic{{MusicId: "m1"}}, nil
		},
	}
	handler := NewServiceHandler(mockStorage, testSecret)
	token := generateTestToken("user-123", testSecret, time.Now().Add(1*time.Hour))

	t.Run("Success", func(t *testing.T) {
		body := GetPlaylistRequest{PlaylistId: 1}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/playlist", bytes.NewBuffer(b))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.GetPlaylist(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		var resp GetPlaylistResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Playlist.PlaylistId != 1 || len(resp.Musics) != 1 {
			t.Errorf("unexpected response body: %+v", resp)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		body := GetPlaylistRequest{PlaylistId: 999}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/playlist", bytes.NewBuffer(b))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.GetPlaylist(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestPutPlaylist(t *testing.T) {
	mockStorage := &MockStorage{
		PutPlaylistFunc:        func(p storages.Playlist) error { return nil },
		PutMusicFunc:           func(m storages.Music) error { return nil },
		PutMusicInPlaylistFunc: func(pm storages.PlaylistMusic) error { return nil },
	}
	handler := NewServiceHandler(mockStorage, testSecret)
	token := generateTestToken("user-123", testSecret, time.Now().Add(1*time.Hour))

	t.Run("Success", func(t *testing.T) {
		body := PutPlaylistRequest{
			Playlist:  storages.Playlist{PlaylistId: 1},
			Musics:    []storages.Music{{MusicId: "m1"}},
			Relations: []storages.PlaylistMusic{{MusicId: "m1"}},
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/playlist/put", bytes.NewBuffer(b))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.PutPlaylist(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/playlist/put", bytes.NewBuffer([]byte(`{bad-json}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.PutPlaylist(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}
