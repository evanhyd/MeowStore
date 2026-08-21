package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meowstore/storages"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("super-secret-test-key")

// ==========================================
// TEST INFRASTRUCTURE
// ==========================================

func generateValidToken(userId string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: userId,
	})
	tokenString, _ := token.SignedString(testSecret)
	return tokenString
}

type MockStorage struct {
	GetPlaylistFunc              func(userId string, playlistId int64) (storages.Playlist, error)
	PutPlaylistFunc              func(playlist storages.Playlist) error
	GetPlaylistsFromUserFunc     func(userId string) ([]storages.Playlist, error)
	GetPlaylistsMetaFromUserFunc func(userId string) ([]storages.PlaylistMeta, error)
	DeletePlaylistFunc           func(userId string, playlistId int64) error

	PutMusicFunc    func(music storages.Music) error
	GetMusicFunc    func(musicId string, source storages.MusicSource) (storages.Music, error)
	DeleteMusicFunc func(musicId string, source storages.MusicSource) error

	PutMusicInPlaylistFunc      func(pm storages.PlaylistMusic) error
	GetMusicFromPlaylistFunc    func(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error)
	DeleteMusicFromPlaylistFunc func(userId string, playlistId int64, musicId string, source storages.MusicSource) error
}

func (m *MockStorage) GetPlaylist(userId string, playlistId int64) (storages.Playlist, error) {
	return m.GetPlaylistFunc(userId, playlistId)
}
func (m *MockStorage) PutPlaylist(playlist storages.Playlist) error {
	return m.PutPlaylistFunc(playlist)
}
func (m *MockStorage) GetPlaylistsFromUser(userId string) ([]storages.Playlist, error) {
	return m.GetPlaylistsFromUserFunc(userId)
}
func (m *MockStorage) GetPlaylistsMetaFromUser(userId string) ([]storages.PlaylistMeta, error) {
	return m.GetPlaylistsMetaFromUserFunc(userId)
}
func (m *MockStorage) DeletePlaylist(userId string, playlistId int64) error {
	return m.DeletePlaylistFunc(userId, playlistId)
}
func (m *MockStorage) PutMusic(music storages.Music) error { return m.PutMusicFunc(music) }
func (m *MockStorage) GetMusic(musicId string, source storages.MusicSource) (storages.Music, error) {
	return m.GetMusicFunc(musicId, source)
}
func (m *MockStorage) DeleteMusic(musicId string, source storages.MusicSource) error {
	return m.DeleteMusicFunc(musicId, source)
}
func (m *MockStorage) PutMusicInPlaylist(pm storages.PlaylistMusic) error {
	return m.PutMusicInPlaylistFunc(pm)
}
func (m *MockStorage) GetMusicFromPlaylist(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error) {
	return m.GetMusicFromPlaylistFunc(userId, playlistId)
}
func (m *MockStorage) DeleteMusicFromPlaylist(userId string, playlistId int64, musicId string, source storages.MusicSource) error {
	return m.DeleteMusicFromPlaylistFunc(userId, playlistId, musicId, source)
}
func (m *MockStorage) Close() error { return nil }

func executeTestRequest(handler http.HandlerFunc, reqBody any) *httptest.ResponseRecorder {
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// ==========================================
// TESTS
// ==========================================

func TestGetPlaylist(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock := &MockStorage{
			GetPlaylistFunc: func(userId string, playlistId int64) (storages.Playlist, error) {
				return storages.Playlist{PlaylistId: playlistId, Title: "Test"}, nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := GetPlaylistRequest{Token: generateValidToken("user_123"), PlaylistId: 5}

		rr := executeTestRequest(svc.GetPlaylist, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var res GetPlaylistResponse
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res.Playlist.PlaylistId != 5 {
			t.Errorf("expected playlist ID 5")
		}
	})
}

func TestGetPlaylistContent(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock := &MockStorage{
			GetPlaylistFunc: func(userId string, playlistId int64) (storages.Playlist, error) {
				return storages.Playlist{UserId: userId, PlaylistId: playlistId, Title: "Test Playlist"}, nil
			},
			GetMusicFromPlaylistFunc: func(userId string, playlistId int64) ([]storages.Music, []storages.PlaylistMusic, error) {
				musics := []storages.Music{{MusicId: "m1", Title: "Song 1"}}
				relations := []storages.PlaylistMusic{{UserId: userId, PlaylistId: playlistId, MusicId: "m1"}}
				return musics, relations, nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := GetPlaylistContentRequest{Token: generateValidToken("user_123"), PlaylistId: 1}

		rr := executeTestRequest(svc.GetPlaylistContent, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var res GetPlaylistContentResponse
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res.Playlist.Title != "Test Playlist" {
			t.Errorf("expected 'Test Playlist', got '%s'", res.Playlist.Title)
		}
		if len(res.Musics) != 1 || len(res.Relations) != 1 {
			t.Errorf("expected 1 music and 1 relation")
		}
	})
}

func TestPutPlaylist(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := false
		mock := &MockStorage{
			PutPlaylistFunc: func(playlist storages.Playlist) error {
				called = true
				if playlist.UserId != "user_123" {
					t.Errorf("expected user_123, got %s", playlist.UserId)
				}
				return nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := PutPlaylistRequest{
			Token:    generateValidToken("user_123"),
			Playlist: storages.Playlist{PlaylistId: 10, Title: "New"},
		}

		rr := executeTestRequest(svc.PutPlaylist, req)

		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and called=true")
		}
	})
}

func TestDeletePlaylist(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := false
		mock := &MockStorage{
			DeletePlaylistFunc: func(userId string, playlistId int64) error {
				called = true
				if playlistId != 15 {
					t.Errorf("expected 15, got %d", playlistId)
				}
				return nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := DeletePlaylistRequest{Token: generateValidToken("user_123"), PlaylistId: 15}

		rr := executeTestRequest(svc.DeletePlaylist, req)

		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and called=true")
		}
	})
}

func TestGetMusic(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock := &MockStorage{
			GetMusicFunc: func(musicId string, source storages.MusicSource) (storages.Music, error) {
				return storages.Music{MusicId: musicId, Source: source}, nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := GetMusicRequest{Token: generateValidToken("user_123"), MusicId: "track1", Source: storages.YouTubeSource}

		rr := executeTestRequest(svc.GetMusic, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var res GetMusicResponse
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res.Music.MusicId != "track1" {
			t.Errorf("expected track1")
		}
	})
}

func TestPutMusic(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := false
		mock := &MockStorage{
			PutMusicFunc: func(music storages.Music) error {
				called = true
				return nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := PutMusicRequest{Token: generateValidToken("user_123"), Music: storages.Music{MusicId: "track1"}}

		rr := executeTestRequest(svc.PutMusic, req)

		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and called=true")
		}
	})
}

func TestGetPlaylistsFromUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock := &MockStorage{
			GetPlaylistsFromUserFunc: func(userId string) ([]storages.Playlist, error) {
				return []storages.Playlist{{UserId: userId, PlaylistId: 99}}, nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := GetPlaylistsFromUserRequest{Token: generateValidToken("user_123")}

		rr := executeTestRequest(svc.GetPlaylistsFromUser, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var res GetPlaylistsFromUserResponse
		json.Unmarshal(rr.Body.Bytes(), &res)
		if len(res.Playlists) != 1 || res.Playlists[0].PlaylistId != 99 {
			t.Errorf("expected playlist ID 99")
		}
	})
}

func TestPutMusicInPlaylist(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := false
		mock := &MockStorage{
			PutMusicInPlaylistFunc: func(pm storages.PlaylistMusic) error {
				called = true
				if pm.PlaylistId != 5 || pm.MusicId != "m1" || pm.UserId != "user_123" {
					t.Errorf("unexpected payload values")
				}
				return nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := PutMusicInPlaylistRequest{
			Token:      generateValidToken("user_123"),
			PlaylistId: 5,
			MusicId:    "m1",
			Source:     storages.SpotifySource,
		}

		rr := executeTestRequest(svc.PutMusicInPlaylist, req)

		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and called=true")
		}
	})
}

func TestDeleteMusicFromPlaylist(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := false
		mock := &MockStorage{
			DeleteMusicFromPlaylistFunc: func(userId string, playlistId int64, musicId string, source storages.MusicSource) error {
				called = true
				if playlistId != 5 || musicId != "m1" {
					t.Errorf("unexpected parameters")
				}
				return nil
			},
		}
		svc := NewServiceHandler(mock, testSecret)
		req := DeleteMusicFromPlaylistRequest{
			Token:      generateValidToken("user_123"),
			PlaylistId: 5,
			MusicId:    "m1",
			Source:     storages.YouTubeSource,
		}

		rr := executeTestRequest(svc.DeleteMusicFromPlaylist, req)

		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and called=true")
		}
	})
}
