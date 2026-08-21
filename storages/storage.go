package storages

import (
	"io"
)

// All interface implementation must be idempotent.
// Put creates or updates the data.
// Delete deletes the data or do nothing if the data doesn't exist.

type PlaylistAccessor interface {
	PutPlaylist(playlist Playlist) error
	GetPlaylist(userId string, playlistId int64) (Playlist, error)
	GetPlaylistsFromUser(userId string) ([]Playlist, error)
	DeletePlaylist(userId string, playlistId int64) error
}

type MusicAccessor interface {
	PutMusic(music Music) error
	GetMusic(musicId string, source MusicSource) (Music, error)
	DeleteMusic(musicId string, source MusicSource) error
}

type PlaylistRelationAccessor interface {
	PutMusicInPlaylist(playlistMusic PlaylistMusic) error
	GetMusicFromPlaylist(userId string, playlistId int64) ([]Music, []PlaylistMusic, error)
	DeleteMusicFromPlaylist(userId string, playlistId int64, musicId string, source MusicSource) error
}

type Storage interface {
	PlaylistAccessor
	MusicAccessor
	PlaylistRelationAccessor
	io.Closer
}
