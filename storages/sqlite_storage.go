package storages

import (
	"database/sql"
	_ "embed"
	"log/slog"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var _ Storage = &SQLiteStorage{}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) *SQLiteStorage {
	storage := SQLiteStorage{}

	var err error
	storage.db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("failed to open SQLite database", "error", err)
		panic("failed to open SQLite database")
	}
	if _, err := storage.db.Exec(schemaSQL); err != nil {
		slog.Error("failed to create schema", "error", err)
		panic("failed to create schema")
	}
	return &storage
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) PutPlaylist(playlist Playlist) error {
	_, err := s.db.Exec(`INSERT INTO playlist(user_id, playlist_id, deleted, title, modified_date, cover_blob) VALUES (?, ?, ?, ?, ?, ?) 
	ON CONFLICT (user_id, playlist_id) DO UPDATE SET deleted = excluded.deleted, title = excluded.title, modified_date = excluded.modified_date, cover_blob = excluded.cover_blob`,
		playlist.UserId, playlist.PlaylistId, playlist.Deleted, playlist.Title, playlist.ModifiedDate, playlist.CoverBlob)
	return err
}

func (s *SQLiteStorage) GetPlaylist(userId string, playlistId int64) (Playlist, error) {
	row := s.db.QueryRow(`SELECT deleted, title, modified_date, cover_blob FROM playlist WHERE user_id = ? AND playlist_id = ?`, userId, playlistId)

	playlist := Playlist{UserId: userId, PlaylistId: playlistId}
	if err := row.Scan(&playlist.Deleted, &playlist.Title, &playlist.ModifiedDate, &playlist.CoverBlob); err != nil {
		return Playlist{}, err
	}
	return playlist, nil
}

func (s *SQLiteStorage) GetPlaylistsMetaFromUser(userId string) ([]PlaylistMeta, error) {
	rows, err := s.db.Query(`SELECT playlist_id, deleted, modified_date FROM playlist WHERE user_id = ?`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlistMetas []PlaylistMeta
	for rows.Next() {
		p := PlaylistMeta{UserId: userId}
		if err := rows.Scan(&p.PlaylistId, &p.Deleted, &p.ModifiedDate); err != nil {
			return nil, err
		}
		playlistMetas = append(playlistMetas, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return playlistMetas, nil
}

func (s *SQLiteStorage) GetPlaylistsFromUser(userId string) ([]Playlist, error) {
	rows, err := s.db.Query(`SELECT playlist_id, deleted, title, modified_date, cover_blob FROM playlist WHERE user_id = ?`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		p := Playlist{UserId: userId}
		if err := rows.Scan(&p.PlaylistId, &p.Deleted, &p.Title, &p.ModifiedDate, &p.CoverBlob); err != nil {
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

func (s *SQLiteStorage) PutMusic(music Music) error {
	_, err := s.db.Exec(`INSERT INTO music(music_id, source, title, length_seconds) VALUES (?, ?, ?, ?) 
	ON CONFLICT (music_id, source) DO UPDATE SET title = excluded.title, length_seconds = excluded.length_seconds`,
		music.MusicId, music.Source, music.Title, music.LengthSeconds)
	return err
}

func (s *SQLiteStorage) GetMusic(musicId string, source MusicSource) (Music, error) {
	row := s.db.QueryRow(`SELECT title, length_seconds FROM music WHERE music_id = ? AND source = ?`, musicId, source)

	music := Music{MusicId: musicId, Source: source}
	if err := row.Scan(&music.Title, &music.LengthSeconds); err != nil {
		return Music{}, err
	}
	return music, nil
}

func (s *SQLiteStorage) DeleteMusic(musicId string, source MusicSource) error {
	_, err := s.db.Exec(`DELETE FROM music WHERE music_id = ? AND source = ?`, musicId, source)
	return err
}

func (s *SQLiteStorage) PutMusicInPlaylist(pm PlaylistMusic) error {
	_, err := s.db.Exec(`INSERT INTO playlist_music(user_id, playlist_id, music_id, source, added_at) VALUES (?, ?, ?, ?, ?) 
	ON CONFLICT (user_id, playlist_id, music_id, source) DO UPDATE SET added_at = excluded.added_at`,
		pm.UserId, pm.PlaylistId, pm.MusicId, pm.Source, pm.AddedAt)
	return err
}

func (s *SQLiteStorage) GetMusicFromPlaylist(userId string, playlistId int64) ([]Music, []PlaylistMusic, error) {
	query := `SELECT m.music_id, m.source, m.title, m.length_seconds, pm.added_at 
		FROM playlist_music pm
		JOIN music m ON pm.music_id = m.music_id AND pm.source = m.source
		WHERE pm.user_id = ? AND pm.playlist_id = ?`
	rows, err := s.db.Query(query, userId, playlistId)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var musics []Music
	var playlistMusics []PlaylistMusic

	for rows.Next() {
		var m Music
		pm := PlaylistMusic{UserId: userId, PlaylistId: playlistId}

		if err := rows.Scan(&m.MusicId, &m.Source, &m.Title, &m.LengthSeconds, &pm.AddedAt); err != nil {
			return nil, nil, err
		}

		pm.MusicId = m.MusicId
		pm.Source = m.Source

		musics = append(musics, m)
		playlistMusics = append(playlistMusics, pm)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return musics, playlistMusics, nil
}

func (s *SQLiteStorage) DeleteMusicFromPlaylist(userId string, playlistId int64, musicId string, source MusicSource) error {
	_, err := s.db.Exec(`DELETE FROM playlist_music WHERE user_id = ? AND playlist_id = ? AND music_id = ? AND source = ?`,
		userId, playlistId, musicId, source)
	return err
}
