package storages

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

// setupTestDB creates an isolated in-memory database for each test.
// We include the foreign_keys pragma in the DSN to ensure the test connection enforces it,
// even if the driver resets connection states before executing schema.sql.
func setupTestDB(t *testing.T) *SQLiteStorage {
	dbPath := "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	storage := NewSQLiteStorage(dbPath)

	t.Cleanup(func() {
		storage.Close()
	})

	return storage
}

func TestPlaylistCRUD(t *testing.T) {
	storage := setupTestDB(t)

	now := time.Now().UnixNano()
	expected := Playlist{
		UserId:       1,
		PlaylistId:   100,
		Title:        "Meow Mix",
		ModifiedDate: now,
		CoverBlob:    []byte("blob_data"),
	}

	// 1. Put
	if err := storage.PutPlaylist(expected); err != nil {
		t.Fatalf("PutPlaylist failed: %v", err)
	}

	// 2. Get
	actual, err := storage.GetPlaylist(expected.UserId, expected.PlaylistId)
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}

	// 3. Update (Upsert)
	expected.Title = "Updated Meow Mix"
	if err := storage.PutPlaylist(expected); err != nil {
		t.Fatalf("PutPlaylist (update) failed: %v", err)
	}

	updated, _ := storage.GetPlaylist(expected.UserId, expected.PlaylistId)
	if updated.Title != "Updated Meow Mix" {
		t.Errorf("expected title 'Updated Meow Mix', got '%s'", updated.Title)
	}

	// 4. Delete
	if err := storage.DeletePlaylist(expected.UserId, expected.PlaylistId); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}

	if _, err := storage.GetPlaylist(expected.UserId, expected.PlaylistId); err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after deletion, got %v", err)
	}
}

func TestMusicCRUD(t *testing.T) {
	storage := setupTestDB(t)

	expected := Music{
		MusicId:       "vid_123",
		Source:        YouTubeSource,
		Title:         "Cat Video Compilation",
		LengthSeconds: 600,
	}

	// 1. Put
	if err := storage.PutMusic(expected); err != nil {
		t.Fatalf("PutMusic failed: %v", err)
	}

	// 2. Get
	actual, err := storage.GetMusic(expected.MusicId, expected.Source)
	if err != nil {
		t.Fatalf("GetMusic failed: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}

	// 3. Delete
	if err := storage.DeleteMusic(expected.MusicId, expected.Source); err != nil {
		t.Fatalf("DeleteMusic failed: %v", err)
	}

	if _, err := storage.GetMusic(expected.MusicId, expected.Source); err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after deletion, got %v", err)
	}
}

func TestPlaylistManagerMethods(t *testing.T) {
	storage := setupTestDB(t)

	userId := int64(1)
	playlistId := int64(10)

	// Seed Data
	storage.PutPlaylist(Playlist{UserId: userId, PlaylistId: playlistId, Title: "Test Playlist", CoverBlob: []byte("a")})
	storage.PutPlaylist(Playlist{UserId: userId, PlaylistId: 11, Title: "Second Playlist", CoverBlob: []byte("a"), ModifiedDate: 1}) // Older

	m1 := Music{MusicId: "m1", Source: YouTubeSource, Title: "Song 1"}
	m2 := Music{MusicId: "m2", Source: SpotifySource, Title: "Song 2"}
	storage.PutMusic(m1)
	storage.PutMusic(m2)

	// 1. PutMusicInPlaylist
	pm1 := PlaylistMusic{UserId: userId, PlaylistId: playlistId, MusicId: m1.MusicId, Source: m1.Source, AddedAt: 100}
	pm2 := PlaylistMusic{UserId: userId, PlaylistId: playlistId, MusicId: m2.MusicId, Source: m2.Source, AddedAt: 200}

	if err := storage.PutMusicInPlaylist(pm1); err != nil {
		t.Fatalf("PutMusicInPlaylist 1 failed: %v", err)
	}
	if err := storage.PutMusicInPlaylist(pm2); err != nil {
		t.Fatalf("PutMusicInPlaylist 2 failed: %v", err)
	}

	// 2. GetPlaylistsFromUser
	playlists, err := storage.GetPlaylistsFromUser(userId)
	if err != nil {
		t.Fatalf("GetPlaylistsFromUser failed: %v", err)
	}
	if len(playlists) != 2 {
		t.Errorf("expected 2 playlists, got %d", len(playlists))
	}

	// 3. GetMusicFromPlaylist
	musics, err := storage.GetMusicFromPlaylist(userId, playlistId)
	if err != nil {
		t.Fatalf("GetMusicFromPlaylist failed: %v", err)
	}
	if len(musics) != 2 {
		t.Fatalf("expected 2 musics, got %d", len(musics))
	}
	if musics[0].MusicId != "m1" || musics[1].MusicId != "m2" {
		t.Errorf("expected ordering by added_at (m1 then m2)")
	}

	// 4. GetPlaylistMusicFromPlaylist
	pms, err := storage.GetPlaylistMusicFromPlaylist(userId, playlistId)
	if err != nil {
		t.Fatalf("GetPlaylistMusicFromPlaylist failed: %v", err)
	}
	if len(pms) != 2 {
		t.Fatalf("expected 2 PlaylistMusic records, got %d", len(pms))
	}
	if !reflect.DeepEqual(pms[0], pm1) {
		t.Errorf("expected %+v, got %+v", pm1, pms[0])
	}

	// 5. DeleteMusicFromPlaylist
	if err := storage.DeleteMusicFromPlaylist(userId, playlistId, m1.MusicId, m1.Source); err != nil {
		t.Fatalf("DeleteMusicFromPlaylist failed: %v", err)
	}

	musicsAfterDelete, _ := storage.GetMusicFromPlaylist(userId, playlistId)
	if len(musicsAfterDelete) != 1 {
		t.Errorf("expected 1 music after deletion, got %d", len(musicsAfterDelete))
	}
}

func TestCascadeDelete(t *testing.T) {
	storage := setupTestDB(t)

	userId := int64(2)
	playlistId := int64(20)
	musicId := "cascade_test"

	storage.PutPlaylist(Playlist{UserId: userId, PlaylistId: playlistId, Title: "Cascade"})
	storage.PutMusic(Music{MusicId: musicId, Source: YouTubeSource, Title: "Cascade Song"})
	storage.PutMusicInPlaylist(PlaylistMusic{UserId: userId, PlaylistId: playlistId, MusicId: musicId, Source: YouTubeSource})

	// Delete the playlist
	if err := storage.DeletePlaylist(userId, playlistId); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}

	// Verify PlaylistMusic was cascaded
	pms, err := storage.GetPlaylistMusicFromPlaylist(userId, playlistId)
	if err != nil {
		t.Fatalf("GetPlaylistMusicFromPlaylist failed: %v", err)
	}
	if len(pms) != 0 {
		t.Errorf("expected PlaylistMusic to be deleted via cascade, found %d records", len(pms))
	}

	// Verify Music still exists (RESTRICT/No cascade on music table)
	if _, err := storage.GetMusic(musicId, YouTubeSource); err != nil {
		t.Errorf("expected music to remain after playlist deletion, got error: %v", err)
	}
}
