package storages

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *SQLiteStorage {
	storage := NewSQLiteStorage(":memory:")
	t.Cleanup(func() {
		err := storage.Close()
		if err != nil {
			t.Logf("failed to close test db: %v", err)
		}
	})
	return storage
}

func TestPlaylistCRUD(t *testing.T) {
	storage := setupTestDB(t)
	userId := "user-123"

	// 1. Put Playlist
	p1 := Playlist{
		UserId:       userId,
		PlaylistId:   1,
		Deleted:      false,
		Title:        "My Favorite Songs",
		ModifiedDate: time.Now().UnixNano(),
		CoverBlob:    []byte("fake-image-data"),
	}

	if err := storage.PutPlaylist(p1); err != nil {
		t.Fatalf("failed to insert playlist: %v", err)
	}

	// 2. Get Playlist
	fetched, err := storage.GetPlaylist(userId, p1.PlaylistId)
	if err != nil {
		t.Fatalf("failed to get playlist: %v", err)
	}
	if !reflect.DeepEqual(p1, fetched) {
		t.Errorf("expected %+v, got %+v", p1, fetched)
	}

	// 3. Put second Playlist (to test GetPlaylistsFromUser)
	p2 := Playlist{
		UserId:       userId,
		PlaylistId:   2,
		Deleted:      true,
		Title:        "Deleted Playlist",
		ModifiedDate: time.Now().UnixNano(),
		CoverBlob:    []byte(""),
	}
	if err := storage.PutPlaylist(p2); err != nil {
		t.Fatalf("failed to insert second playlist: %v", err)
	}

	// 4. GetPlaylistsFromUser
	playlists, err := storage.GetPlaylistsFromUser(userId)
	if err != nil {
		t.Fatalf("failed to get playlists from user: %v", err)
	}
	if len(playlists) != 2 {
		t.Errorf("expected 2 playlists, got %d", len(playlists))
	}

	// 5. Update existing Playlist (Upsert test)
	p1.Title = "Updated Title"
	if err := storage.PutPlaylist(p1); err != nil {
		t.Fatalf("failed to update playlist: %v", err)
	}
	fetchedUpdated, _ := storage.GetPlaylist(userId, p1.PlaylistId)
	if fetchedUpdated.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", fetchedUpdated.Title)
	}

	// 6. Delete Playlist
	if err := storage.DeletePlaylist(userId, p2.PlaylistId); err != nil {
		t.Fatalf("failed to delete playlist: %v", err)
	}

	// Verify deletion
	_, err = storage.GetPlaylist(userId, p2.PlaylistId)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestMusicCRUD(t *testing.T) {
	storage := setupTestDB(t)

	// 1. Put Music
	m1 := Music{
		MusicId:       "vid-123",
		Source:        YouTubeSource,
		Title:         "Awesome Song",
		LengthSeconds: 215,
	}

	if err := storage.PutMusic(m1); err != nil {
		t.Fatalf("failed to insert music: %v", err)
	}

	// 2. Get Music
	fetched, err := storage.GetMusic(m1.MusicId, m1.Source)
	if err != nil {
		t.Fatalf("failed to get music: %v", err)
	}
	if !reflect.DeepEqual(m1, fetched) {
		t.Errorf("expected %+v, got %+v", m1, fetched)
	}

	// 3. Update Music (Upsert)
	m1.Title = "Awesome Song (Live)"
	if err := storage.PutMusic(m1); err != nil {
		t.Fatalf("failed to update music: %v", err)
	}
	fetchedUpdated, _ := storage.GetMusic(m1.MusicId, m1.Source)
	if fetchedUpdated.Title != "Awesome Song (Live)" {
		t.Errorf("expected updated title, got '%s'", fetchedUpdated.Title)
	}

	// 4. Delete Music
	if err := storage.DeleteMusic(m1.MusicId, m1.Source); err != nil {
		t.Fatalf("failed to delete music: %v", err)
	}

	// Verify deletion
	_, err = storage.GetMusic(m1.MusicId, m1.Source)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestPlaylistMusicCRUD(t *testing.T) {
	storage := setupTestDB(t)
	userId := "user-456"
	playlistId := int64(99)

	// Setup: Foreign keys require the Playlist and Music to exist first
	playlist := Playlist{
		UserId:       userId,
		PlaylistId:   playlistId,
		Deleted:      false,
		Title:        "Mix 2026",
		ModifiedDate: time.Now().UnixNano(),
		CoverBlob:    []byte{},
	}
	if err := storage.PutPlaylist(playlist); err != nil {
		t.Fatalf("failed to setup playlist: %v", err)
	}

	music := Music{
		MusicId:       "spot-789",
		Source:        SpotifySource,
		Title:         "Cool Track",
		LengthSeconds: 180,
	}
	if err := storage.PutMusic(music); err != nil {
		t.Fatalf("failed to setup music: %v", err)
	}

	// 1. Put Music in Playlist
	pm := PlaylistMusic{
		UserId:     userId,
		PlaylistId: playlistId,
		MusicId:    music.MusicId,
		Source:     music.Source,
		AddedAt:    time.Now().UnixNano(),
	}

	if err := storage.PutMusicInPlaylist(pm); err != nil {
		t.Fatalf("failed to put music in playlist: %v", err)
	}

	// 2. Get Music from Playlist
	musics, pms, err := storage.GetMusicFromPlaylist(userId, playlistId)
	if err != nil {
		t.Fatalf("failed to get music from playlist: %v", err)
	}

	if len(musics) != 1 || len(pms) != 1 {
		t.Fatalf("expected 1 record, got musics: %d, pms: %d", len(musics), len(pms))
	}

	if !reflect.DeepEqual(music, musics[0]) {
		t.Errorf("expected music %+v, got %+v", music, musics[0])
	}

	if !reflect.DeepEqual(pm, pms[0]) {
		t.Errorf("expected playlist_music %+v, got %+v", pm, pms[0])
	}

	// 3. Delete Music from Playlist
	if err := storage.DeleteMusicFromPlaylist(userId, playlistId, music.MusicId, music.Source); err != nil {
		t.Fatalf("failed to delete music from playlist: %v", err)
	}

	// Verify deletion
	musicsAfterDelete, _, err := storage.GetMusicFromPlaylist(userId, playlistId)
	if err != nil {
		t.Fatalf("failed to query after deletion: %v", err)
	}
	if len(musicsAfterDelete) != 0 {
		t.Errorf("expected empty playlist, got %d items", len(musicsAfterDelete))
	}
}

func TestGetPlaylistsMetaFromUser(t *testing.T) {
	storage := setupTestDB(t)
	userId := "user-meta-test"
	otherUserId := "user-other"

	// 1. Setup: Insert test playlists
	p1 := Playlist{
		UserId:       userId,
		PlaylistId:   1,
		Deleted:      false,
		Title:        "Active Playlist",
		ModifiedDate: 1700000000,
		CoverBlob:    []byte{},
	}
	p2 := Playlist{
		UserId:       userId,
		PlaylistId:   2,
		Deleted:      true,
		Title:        "Deleted Playlist",
		ModifiedDate: 1700000500,
		CoverBlob:    []byte{},
	}
	p3 := Playlist{ // Should not be retrieved (belongs to different user)
		UserId:       otherUserId,
		PlaylistId:   3,
		Deleted:      false,
		Title:        "Other User Playlist",
		ModifiedDate: 1700001000,
		CoverBlob:    []byte{},
	}

	if err := storage.PutPlaylist(p1); err != nil {
		t.Fatalf("failed to insert p1: %v", err)
	}
	if err := storage.PutPlaylist(p2); err != nil {
		t.Fatalf("failed to insert p2: %v", err)
	}
	if err := storage.PutPlaylist(p3); err != nil {
		t.Fatalf("failed to insert p3: %v", err)
	}

	// 2. Execute: Get metadata
	metas, err := storage.GetPlaylistsMetaFromUser(userId)
	if err != nil {
		t.Fatalf("failed to get playlist meta: %v", err)
	}

	// 3. Verify: Check results length and content
	if len(metas) != 2 {
		t.Fatalf("expected 2 playlist metas, got %d", len(metas))
	}

	// Create a map for easy lookup and assertion
	metaMap := make(map[int64]PlaylistMeta)
	for _, m := range metas {
		metaMap[m.PlaylistId] = m
	}

	// Verify p1 metadata
	if m1, ok := metaMap[p1.PlaylistId]; !ok {
		t.Errorf("expected playlist %d to be in results", p1.PlaylistId)
	} else {
		if m1.UserId != p1.UserId {
			t.Errorf("expected UserId %s, got %s", p1.UserId, m1.UserId)
		}
		if m1.Deleted != p1.Deleted {
			t.Errorf("expected Deleted %v, got %v", p1.Deleted, m1.Deleted)
		}
		if m1.ModifiedDate != p1.ModifiedDate {
			t.Errorf("expected ModifiedDate %d, got %d", p1.ModifiedDate, m1.ModifiedDate)
		}
	}

	// Verify p2 metadata (specifically checking the Deleted flag logic)
	if m2, ok := metaMap[p2.PlaylistId]; !ok {
		t.Errorf("expected playlist %d to be in results", p2.PlaylistId)
	} else {
		if m2.Deleted != p2.Deleted {
			t.Errorf("expected Deleted %v for p2, got %v", p2.Deleted, m2.Deleted)
		}
	}
}
