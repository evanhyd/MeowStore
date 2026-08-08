package storages

type MusicSource int64

const (
	UnknownSource MusicSource = iota
	YouTubeSource
	SpotifySource
)

type PlaylistMeta struct {
	UserId       string
	PlaylistId   int64
	Deleted      bool
	ModifiedDate int64 // Unix nano
}

// Playlist table
type Playlist struct {
	UserId       string
	PlaylistId   int64
	Deleted      bool
	ModifiedDate int64 // Unix nano
	Title        string
	CoverBlob    []byte
}

// Music table
type Music struct {
	MusicId       string
	Source        MusicSource
	Title         string
	LengthSeconds int64
}

// PlaylistMusic table
type PlaylistMusic struct {
	UserId     string
	PlaylistId int64
	MusicId    string
	Source     MusicSource
	AddedAt    int64 // Unix nano
}
