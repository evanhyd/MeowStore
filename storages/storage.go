package storages

import (
	"io"
)

type PlaylistStorer interface {
	PutPlaylist(playlist Playlist) (Playlist, error)
	GetPlaylist(userId string, playlistId int64) (Playlist, error)
	GetPlaylists(userId string) ([]Playlist, error)
	DeletePlaylist(userId string, playlistId int64) error

	PutPlaylistMusic(playlistMusic PlaylistMusic) error
	GetAllPlaylistMusic(userId string, playlistId int64) ([]PlaylistMusic, error)
	DeletePlaylistMusic(playlistMusic PlaylistMusic) error
}

type MusicStorer interface {
	PutMusic(music Music) error
	GetMusic(musicId string, source MusicSource) (Music, error)
	GetAllMusic(userId string, playlistId int64) ([]Music, error)
	DeleteMusic(musicId string, source MusicSource) error
}

type Storage interface {
	PlaylistStorer
	MusicStorer
	io.Closer
}
