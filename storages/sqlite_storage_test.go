package storages

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "sqlite_server_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "server_test.db")

	storage := NewSQLiteStorage(dbPath)
	if storage == nil {
		t.Fatalf("failed to create SQLiteStorage")
	}

	return storage, func() {
		storage.Close()
		os.RemoveAll(tmpDir)
	}
}

func TestSQLiteStorage_Server_Playlist(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	p := Playlist{
		UserId:       "user_100",
		PlaylistId:   1,
		Title:        "Server Sync Playlist",
		ModifiedDate: 1000,
		CoverBlob:    []byte("cover_data"),
	}

	// 1. Test Insert
	if _, err := s.PutPlaylist(p); err != nil {
		t.Fatalf("PutPlaylist failed: %v", err)
	}

	got, err := s.GetPlaylist(p.UserId, p.PlaylistId)
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if got.Title != p.Title {
		t.Errorf("GetPlaylist mismatch. got title %v, want %v", got.Title, p.Title)
	}

	// 2. Test UPSERT Overwrite Protection (Stale Update should be ignored)
	stalePlaylist := p
	stalePlaylist.Title = "Stale Title"
	stalePlaylist.ModifiedDate = 500 // Older than 1000

	if _, err := s.PutPlaylist(stalePlaylist); err != nil {
		t.Fatalf("PutPlaylist (stale) failed: %v", err)
	}

	gotAfterStale, _ := s.GetPlaylist(p.UserId, p.PlaylistId)
	if gotAfterStale.Title == stalePlaylist.Title {
		t.Errorf("Overwrite protection failed! Stale update overwrote the database.")
	}

	// 3. Test UPSERT Overwrite Protection (Fresh Update should succeed)
	freshPlaylist := p
	freshPlaylist.Title = "Fresh Title"
	freshPlaylist.ModifiedDate = 2000 // Newer than 1000

	if _, err := s.PutPlaylist(freshPlaylist); err != nil {
		t.Fatalf("PutPlaylist (fresh) failed: %v", err)
	}

	gotAfterFresh, _ := s.GetPlaylist(p.UserId, p.PlaylistId)
	if gotAfterFresh.Title != freshPlaylist.Title {
		t.Errorf("Fresh update failed to overwrite the database. got %v", gotAfterFresh.Title)
	}

	// 4. Test GetPlaylists (Multiple)
	_, _ = s.PutPlaylist(Playlist{UserId: "user_100", PlaylistId: 2, Title: "List 2", ModifiedDate: 3000, CoverBlob: []byte{}})

	lists, err := s.GetPlaylists("user_100")
	if err != nil {
		t.Fatalf("GetPlaylists failed: %v", err)
	}
	if len(lists) != 2 {
		t.Fatalf("Expected 2 playlists, got %d", len(lists))
	}
	// Verify sorting (DESC by modified_date)
	if lists[0].PlaylistId != 2 || lists[1].PlaylistId != 1 {
		t.Errorf("GetPlaylists returned incorrect order")
	}

	// 5. Test Delete
	if err := s.DeletePlaylist("user_100", 1); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	if _, err := s.GetPlaylist("user_100", 1); err == nil {
		t.Error("GetPlaylist expected error after deletion, got nil")
	}
}

func TestSQLiteStorage_Server_Music(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	m := Music{
		MusicId:       "track_1",
		Source:        YouTubeSource,
		Title:         "Epic Song",
		LengthSeconds: 210,
	}

	if err := s.PutMusic(m); err != nil {
		t.Fatalf("PutMusic failed: %v", err)
	}

	got, err := s.GetMusic(m.MusicId, m.Source)
	if err != nil {
		t.Fatalf("GetMusic failed: %v", err)
	}
	if !reflect.DeepEqual(m, got) {
		t.Errorf("GetMusic mismatch. got %v, want %v", got, m)
	}

	m.Title = "Epic Song (Remastered)"
	if err := s.PutMusic(m); err != nil {
		t.Fatalf("PutMusic UPSERT failed: %v", err)
	}

	gotUpdate, _ := s.GetMusic(m.MusicId, m.Source)
	if gotUpdate.Title != "Epic Song (Remastered)" {
		t.Errorf("PutMusic UPSERT mismatch. got %v", gotUpdate.Title)
	}

	if err := s.DeleteMusic(m.MusicId, m.Source); err != nil {
		t.Fatalf("DeleteMusic failed: %v", err)
	}
}

func TestSQLiteStorage_Server_PlaylistMusic_And_GetAllMusic(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	userId := "user_999"
	playlistId := int64(10)

	_, _ = s.PutPlaylist(Playlist{UserId: userId, PlaylistId: playlistId, Title: "Mix", ModifiedDate: 1, CoverBlob: []byte{}})

	m1 := Music{MusicId: "m1", Source: YouTubeSource, Title: "Song A", LengthSeconds: 100}
	m2 := Music{MusicId: "m2", Source: SpotifySource, Title: "Song B", LengthSeconds: 200}
	_ = s.PutMusic(m1)
	_ = s.PutMusic(m2)

	rel1 := PlaylistMusic{
		UserId:       userId,
		PlaylistId:   playlistId,
		MusicId:      m1.MusicId,
		Source:       int64(m1.Source),
		ModifiedDate: 100,
	}
	rel2 := PlaylistMusic{
		UserId:       userId,
		PlaylistId:   playlistId,
		MusicId:      m2.MusicId,
		Source:       int64(m2.Source),
		ModifiedDate: 200,
	}

	if err := s.PutPlaylistMusic(rel1); err != nil {
		t.Fatalf("PutPlaylistMusic rel1 failed: %v", err)
	}
	if err := s.PutPlaylistMusic(rel2); err != nil {
		t.Fatalf("PutPlaylistMusic rel2 failed: %v", err)
	}

	// 2. Test Stale Overwrite Protection on Relations
	staleRel := rel1
	staleRel.ModifiedDate = 50
	if err := s.PutPlaylistMusic(staleRel); err != nil {
		t.Fatalf("PutPlaylistMusic stale failed: %v", err)
	}

	// 3. Test GetAllPlaylistMusic
	rels, err := s.GetAllPlaylistMusic(userId, playlistId)
	if err != nil {
		t.Fatalf("GetAllPlaylistMusic failed: %v", err)
	}
	if len(rels) != 2 {
		t.Fatalf("Expected 2 relations, got %d", len(rels))
	}
	if rels[0].ModifiedDate != 100 {
		t.Errorf("Overwrite protection failed on relation! Time dropped to %d", rels[0].ModifiedDate)
	}
	if rels[0].MusicId != "m1" || rels[1].MusicId != "m2" {
		t.Errorf("GetAllPlaylistMusic returned incorrect order")
	}

	// 4. Test GetAllMusic (JOIN logic)
	musics, err := s.GetAllMusic(userId, playlistId)
	if err != nil {
		t.Fatalf("GetAllMusic failed: %v", err)
	}
	if len(musics) != 2 {
		t.Fatalf("GetAllMusic returned %d musics, want 2", len(musics))
	}
	if musics[0].Title != "Song A" || musics[1].Title != "Song B" {
		t.Errorf("GetAllMusic JOIN or sorting logic failed")
	}

	// 5. Test DeletePlaylistMusic
	if err := s.DeletePlaylistMusic(rel1); err != nil {
		t.Fatalf("DeletePlaylistMusic failed: %v", err)
	}

	relsAfter, _ := s.GetAllPlaylistMusic(userId, playlistId)
	if len(relsAfter) != 1 {
		t.Errorf("Expected 1 relation after deletion, got %d", len(relsAfter))
	}
}
