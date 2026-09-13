package storages

import (
	"database/sql"
	"log/slog"

	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var _ Storage = (*SQLiteStorage)(nil)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) *SQLiteStorage {
	dsn := dbPath + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		return nil
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaSQL); err != nil {
		slog.Error("failed to create schema", "error", err)
		return nil
	}

	return &SQLiteStorage{
		db: db,
	}
}

// ---------------- Playlist Methods ----------------

func (s *SQLiteStorage) PutPlaylist(p Playlist) (Playlist, error) {
	_, err := s.db.Exec(
		`INSERT INTO playlist (user_id, playlist_id, title, modified_date, cover_blob)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(user_id, playlist_id) DO UPDATE SET 
            title = excluded.title, 
            modified_date = excluded.modified_date, 
            cover_blob = excluded.cover_blob
        WHERE excluded.modified_date > playlist.modified_date`,
		p.UserId, p.PlaylistId, p.Title, p.ModifiedDate, p.CoverBlob,
	)
	return p, err
}

func (s *SQLiteStorage) GetPlaylist(userId string, playlistId int64) (Playlist, error) {
	var p Playlist
	err := s.db.QueryRow(
		`SELECT user_id, playlist_id, title, modified_date, cover_blob
        FROM playlist WHERE user_id = ? AND playlist_id = ?`,
		userId, playlistId,
	).Scan(&p.UserId, &p.PlaylistId, &p.Title, &p.ModifiedDate, &p.CoverBlob)

	return p, err
}

func (s *SQLiteStorage) GetPlaylists(userId string) ([]Playlist, error) {
	rows, err := s.db.Query(
		`SELECT user_id, playlist_id, title, modified_date, cover_blob 
         FROM playlist WHERE user_id = ? ORDER BY modified_date DESC`,
		userId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make([]Playlist, 0)
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.UserId, &p.PlaylistId, &p.Title, &p.ModifiedDate, &p.CoverBlob); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return playlists, nil
}

func (s *SQLiteStorage) DeletePlaylist(userId string, playlistId int64) error {
	_, err := s.db.Exec(`DELETE FROM playlist WHERE user_id = ? AND playlist_id = ?`, userId, playlistId)
	return err
}

// ---------------- PlaylistMusic Methods ----------------

func (s *SQLiteStorage) PutPlaylistMusic(relation PlaylistMusic) error {
	_, err := s.db.Exec(
		`INSERT INTO playlist_music (user_id, playlist_id, music_id, source, modified_date) 
         VALUES (?, ?, ?, ?, ?)
         ON CONFLICT(user_id, playlist_id, music_id, source) 
         DO UPDATE SET modified_date = excluded.modified_date
         WHERE excluded.modified_date > playlist_music.modified_date`,
		relation.UserId, relation.PlaylistId, relation.MusicId, relation.Source, relation.ModifiedDate,
	)
	return err
}

func (s *SQLiteStorage) GetAllPlaylistMusic(userId string, playlistId int64) ([]PlaylistMusic, error) {
	rows, err := s.db.Query(
		`SELECT user_id, playlist_id, music_id, source, modified_date 
         FROM playlist_music WHERE user_id = ? AND playlist_id = ?
         ORDER BY modified_date ASC`,
		userId, playlistId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relations := make([]PlaylistMusic, 0)
	for rows.Next() {
		var r PlaylistMusic
		if err := rows.Scan(&r.UserId, &r.PlaylistId, &r.MusicId, &r.Source, &r.ModifiedDate); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return relations, nil
}

func (s *SQLiteStorage) DeletePlaylistMusic(rel PlaylistMusic) error {
	_, err := s.db.Exec(
		`DELETE FROM playlist_music WHERE user_id = ? AND playlist_id = ? AND music_id = ? AND source = ?`,
		rel.UserId, rel.PlaylistId, rel.MusicId, rel.Source,
	)
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

func (s *SQLiteStorage) GetMusic(musicId string, source MusicSource) (Music, error) {
	var m Music
	err := s.db.QueryRow(
		`SELECT music_id, source, title, length_seconds FROM music WHERE music_id = ? AND source = ?`,
		musicId, int64(source),
	).Scan(&m.MusicId, &m.Source, &m.Title, &m.LengthSeconds)

	return m, err
}

func (s *SQLiteStorage) GetAllMusic(userId string, playlistId int64) ([]Music, error) {
	rows, err := s.db.Query(
		`SELECT m.music_id, m.source, m.title, m.length_seconds
         FROM music m
         JOIN playlist_music pm ON m.music_id = pm.music_id AND m.source = pm.source
         WHERE pm.user_id = ? AND pm.playlist_id = ?
         ORDER BY pm.modified_date ASC`,
		userId, playlistId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	musics := make([]Music, 0)
	for rows.Next() {
		var m Music
		if err := rows.Scan(&m.MusicId, &m.Source, &m.Title, &m.LengthSeconds); err != nil {
			return nil, err
		}
		musics = append(musics, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return musics, nil
}

func (s *SQLiteStorage) DeleteMusic(musicId string, source MusicSource) error {
	_, err := s.db.Exec(`DELETE FROM music WHERE music_id = ? AND source = ?`, musicId, int64(source))
	return err
}

// ---------------- Closer ----------------

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
