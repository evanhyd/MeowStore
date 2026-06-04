package storages

import (
	"io"
)

// All interface implementation must be idempotent.
// Put creates or updates the data.
// Delete deletes the data or do nothing if the data doesn't exist.

type PlaylistStorer interface {
	PutPlaylist(playlist Playlist) error
	GetPlaylist(userId int64, playlistID int64) (Playlist, error)
	DeletePlaylist(userId int64, playlistID int64) error
}

type MusicStorer interface {
	PutMusic(music Music) error
	GetMusic(musicID string, source MusicSource) (Music, error)
	DeleteMusic(musicID string, source MusicSource) error
}

type PlaylistManager interface {
	PutMusicInPlaylist(playlistMusic PlaylistMusic) error
	GetPlaylistsFromUser(userId int64) ([]Playlist, error)
	GetMusicFromPlaylist(userId int64, playlistID int64) ([]Music, error)
	GetPlaylistMusicFromPlaylist(userId int64, playlistID int64) ([]PlaylistMusic, error)
	DeleteMusicFromPlaylist(userId int64, playlistID int64, musicID string, source MusicSource) error
}

type Storage interface {
	PlaylistStorer
	MusicStorer
	PlaylistManager
	io.Closer
}
