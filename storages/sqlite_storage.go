package storages

import (
	"database/sql"
	_ "embed"
	"log/slog"
	"os"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var _ Storage = &SQLiteStorage{}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) *SQLiteStorage {
	storage := &SQLiteStorage{}

	var err error
	storage.db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		os.Exit(1)
	}
	if _, err := storage.db.Exec(schemaSQL); err != nil {
		slog.Error("failed to create schema", "error", err)
		os.Exit(1)
	}
	return storage
}

// ---------------- Playlist Methods ----------------

func (s *SQLiteStorage) PutPlaylist(playlist Playlist) error {
	_, err := s.db.Exec(
		`INSERT INTO playlists (user_id, playlist_id, title, modified_date, cover_blob)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(user_id, playlist_id) DO UPDATE SET 
            title = excluded.title, 
            modified_date = excluded.modified_date, 
            cover_blob = excluded.cover_blob`,
		playlist.UserId, playlist.PlaylistId, playlist.Title, playlist.ModifiedDate, playlist.CoverBlob,
	)
	return err
}

func (s *SQLiteStorage) GetPlaylist(userId int64, playlistID int64) (Playlist, error) {
	p := Playlist{UserId: userId}
	err := s.db.QueryRow(
		`SELECT playlist_id, title, modified_date, cover_blob
        FROM playlists WHERE user_id = ? AND playlist_id = ?`,
		p.UserId, playlistID,
	).Scan(&p.PlaylistId, &p.Title, &p.ModifiedDate, &p.CoverBlob)

	return p, err
}

func (s *SQLiteStorage) DeletePlaylist(userId int64, playlistID int64) error {
	_, err := s.db.Exec(`DELETE FROM playlists WHERE user_id = ? AND playlist_id = ?`, userId, playlistID)
	return err
}

// ---------------- Music Methods ----------------

func (s *SQLiteStorage) PutMusic(m Music) error {
	_, err := s.db.Exec(
		`INSERT INTO music (music_id, source, title, length_seconds) VALUES (?, ?, ?, ?)
        ON CONFLICT(music_id, source) DO UPDATE SET title = excluded.title, length_seconds = excluded.length_seconds`,
		m.MusicId, m.Source, m.Title, m.LengthSeconds,
	)
	return err
}

func (s *SQLiteStorage) GetMusic(musicID string, source MusicSource) (Music, error) {
	var m Music
	err := s.db.QueryRow(
		`SELECT music_id, source, title, length_seconds FROM music WHERE music_id = ? AND source = ?`,
		musicID, source,
	).Scan(&m.MusicId, &m.Source, &m.Title, &m.LengthSeconds)

	return m, err
}

func (s *SQLiteStorage) DeleteMusic(musicID string, source MusicSource) error {
	_, err := s.db.Exec(`DELETE FROM music WHERE music_id = ? AND source = ?`, musicID, source)
	return err
}

// ---------------- Playlist Manager Methods ----------------

func (s *SQLiteStorage) PutMusicInPlaylist(playlistMusic PlaylistMusic) error {
	_, err := s.db.Exec(
		`INSERT INTO playlist_music (user_id, playlist_id, music_id, source, added_at) 
         VALUES (?, ?, ?, ?, ?)
         ON CONFLICT(playlist_id, music_id, source) DO UPDATE SET added_at = excluded.added_at`,
		playlistMusic.UserId, playlistMusic.PlaylistId, playlistMusic.MusicId, playlistMusic.Source, playlistMusic.AddedAt,
	)
	return err
}

func (s *SQLiteStorage) GetPlaylistsFromUser(userId int64) ([]Playlist, error) {
	rows, err := s.db.Query(
		`SELECT playlist_id, title, modified_date, cover_blob FROM playlists WHERE user_id = ? ORDER BY modified_date DESC`,
		userId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		p := Playlist{UserId: userId}
		if err := rows.Scan(&p.PlaylistId, &p.Title, &p.ModifiedDate, &p.CoverBlob); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

func (s *SQLiteStorage) GetMusicFromPlaylist(userId int64, playlistID int64) ([]Music, error) {
	rows, err := s.db.Query(
		`SELECT m.music_id, m.source, m.title, m.length_seconds
        FROM music m
        JOIN playlist_music pm ON m.music_id = pm.music_id AND m.source = pm.source
        WHERE pm.user_id = ? AND pm.playlist_id = ?
        ORDER BY pm.added_at ASC`,
		userId, playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var musics []Music
	for rows.Next() {
		var m Music
		if err := rows.Scan(&m.MusicId, &m.Source, &m.Title, &m.LengthSeconds); err != nil {
			return nil, err
		}
		musics = append(musics, m)
	}
	return musics, nil
}

func (s *SQLiteStorage) GetPlaylistMusicFromPlaylist(userId int64, playlistID int64) ([]PlaylistMusic, error) {
	rows, err := s.db.Query(
		`SELECT p.music_id, p.source, p.added_at
        FROM playlist_music p
        WHERE p.user_id = ? AND p.playlist_id = ?
        ORDER BY p.added_at ASC`,
		userId, playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlistMusic []PlaylistMusic
	for rows.Next() {
		pm := PlaylistMusic{UserId: userId, PlaylistId: playlistID}
		if err := rows.Scan(&pm.MusicId, &pm.Source, &pm.AddedAt); err != nil {
			return nil, err
		}
		playlistMusic = append(playlistMusic, pm)
	}
	return playlistMusic, nil
}

func (s *SQLiteStorage) DeleteMusicFromPlaylist(userId int64, playlistID int64, musicID string, source MusicSource) error {
	_, err := s.db.Exec(
		`DELETE FROM playlist_music WHERE user_id = ? AND playlist_id = ? AND music_id = ? AND source = ?`,
		userId, playlistID, musicID, source,
	)
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
