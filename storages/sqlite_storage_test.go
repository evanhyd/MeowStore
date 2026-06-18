package storages

import (
	"database/sql"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *SQLiteStorage {
	store := NewSQLiteStorage(":memory:")
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func TestPlaylistCRUD(t *testing.T) {
	store := setupTestDB(t)

	userID := "user_123"
	playlistID := int64(1)
	now := time.Now().UnixNano()

	p := Playlist{
		UserId:       userID,
		PlaylistId:   playlistID,
		Title:        "Workout Mix",
		ModifiedDate: now,
		CoverBlob:    []byte("fake_image_data"),
	}

	// 1. Create
	if err := store.PutPlaylist(p); err != nil {
		t.Fatalf("PutPlaylist failed: %v", err)
	}

	// 2. Read
	fetched, err := store.GetPlaylist(userID, playlistID)
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if fetched.Title != p.Title {
		t.Errorf("Expected title %s, got %s", p.Title, fetched.Title)
	}

	// 3. Update (Testing ON CONFLICT)
	p.Title = "Updated Workout Mix"
	if err := store.PutPlaylist(p); err != nil {
		t.Fatalf("PutPlaylist (Update) failed: %v", err)
	}

	fetched, _ = store.GetPlaylist(userID, playlistID)
	if fetched.Title != "Updated Workout Mix" {
		t.Errorf("Expected updated title, got %s", fetched.Title)
	}

	// 4. Delete
	if err := store.DeletePlaylist(userID, playlistID); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}

	// 5. Verify Deletion
	_, err = store.GetPlaylist(userID, playlistID)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows after deletion, got: %v", err)
	}
}

func TestMusicCRUD(t *testing.T) {
	store := setupTestDB(t)

	m := Music{
		MusicId:       "yt_abc123",
		Source:        YouTubeSource, // Assuming YouTubeSource is defined
		Title:         "Never Gonna Give You Up",
		LengthSeconds: 212,
	}

	// 1. Create
	if err := store.PutMusic(m); err != nil {
		t.Fatalf("PutMusic failed: %v", err)
	}

	// 2. Read
	fetched, err := store.GetMusic(m.MusicId, m.Source)
	if err != nil {
		t.Fatalf("GetMusic failed: %v", err)
	}
	if fetched.Title != m.Title {
		t.Errorf("Expected title %s, got %s", m.Title, fetched.Title)
	}

	// 3. Delete
	if err := store.DeleteMusic(m.MusicId, m.Source); err != nil {
		t.Fatalf("DeleteMusic failed: %v", err)
	}

	// 4. Verify Deletion
	_, err = store.GetMusic(m.MusicId, m.Source)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows after deletion, got: %v", err)
	}
}

func TestPlaylistManagerLifecycle(t *testing.T) {
	store := setupTestDB(t)

	userID := "user_456"
	playlistID := int64(10)
	now := time.Now().UnixNano()

	// Setup: Create a playlist and a music track first to satisfy foreign keys
	// (Assuming your schema.sql enforces foreign keys. If not, this is still good practice).
	if err := store.PutPlaylist(Playlist{UserId: userID, PlaylistId: playlistID, Title: "My Favorites", ModifiedDate: now, CoverBlob: []byte("")}); err != nil {
		t.Fatalf("PutPlaylist failed: %v", err)
	}

	m1 := Music{MusicId: "song_1", Source: SpotifySource, Title: "Song One", LengthSeconds: 100}
	m2 := Music{MusicId: "song_2", Source: SpotifySource, Title: "Song Two", LengthSeconds: 200}
	store.PutMusic(m1)
	store.PutMusic(m2)

	// 1. Add Music to Playlist
	pm1 := PlaylistMusic{
		UserId:     userID,
		PlaylistId: playlistID,
		MusicId:    m1.MusicId,
		Source:     m1.Source,
		AddedAt:    now,
	}
	pm2 := PlaylistMusic{
		UserId:     userID,
		PlaylistId: playlistID,
		MusicId:    m2.MusicId,
		Source:     m2.Source,
		AddedAt:    now + 1000,
	}

	if err := store.PutMusicInPlaylist(pm1); err != nil {
		t.Fatalf("PutMusicInPlaylist 1 failed: %v", err)
	}
	if err := store.PutMusicInPlaylist(pm2); err != nil {
		t.Fatalf("PutMusicInPlaylist 2 failed: %v", err)
	}

	// 2. GetPlaylistsFromUser
	playlists, err := store.GetPlaylistsFromUser(userID)
	if err != nil {
		t.Fatalf("GetPlaylistsFromUser failed: %v", err)
	}
	if len(playlists) != 1 {
		t.Errorf("Expected 1 playlist, got %d", len(playlists))
	}

	// 3. GetPlaylistMusicFromPlaylist (Returns relation table data)
	pmRelations, err := store.GetPlaylistMusicFromPlaylist(userID, playlistID)
	if err != nil {
		t.Fatalf("GetPlaylistMusicFromPlaylist failed: %v", err)
	}
	if len(pmRelations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(pmRelations))
	}
	// Check sorting (ORDER BY added_at ASC)
	if pmRelations[0].MusicId != m1.MusicId {
		t.Errorf("Expected first song to be %s, got %s", m1.MusicId, pmRelations[0].MusicId)
	}

	// 4. GetMusicFromPlaylist (Returns joined Music data)
	musicList, err := store.GetMusicFromPlaylist(userID, playlistID)
	if err != nil {
		t.Fatalf("GetMusicFromPlaylist failed: %v", err)
	}
	if len(musicList) != 2 {
		t.Errorf("Expected 2 music tracks, got %d", len(musicList))
	}
	if musicList[0].Title != m1.Title {
		t.Errorf("Expected joined title to be %s, got %s", m1.Title, musicList[0].Title)
	}

	// 5. Delete Music From Playlist
	if err := store.DeleteMusicFromPlaylist(userID, playlistID, m1.MusicId, m1.Source); err != nil {
		t.Fatalf("DeleteMusicFromPlaylist failed: %v", err)
	}

	// Verify it was removed
	remaining, _ := store.GetPlaylistMusicFromPlaylist(userID, playlistID)
	if len(remaining) != 1 {
		t.Errorf("Expected 1 relation remaining, got %d", len(remaining))
	}
	if remaining[0].MusicId != m2.MusicId {
		t.Errorf("Expected remaining song to be %s", m2.MusicId)
	}
}
