PRAGMA foreign_keys = ON;

-- Playlist table
CREATE TABLE IF NOT EXISTS playlists (
    user_id TEXT NOT NULL,
    playlist_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    modified_date INTEGER NOT NULL,  -- Unix nano
    cover_blob BLOB NOT NULL,
    PRIMARY KEY(user_id, playlist_id)
);

-- Music table
CREATE TABLE IF NOT EXISTS music (
    music_id TEXT NOT NULL,
    source INTEGER NOT NULL,
    title TEXT NOT NULL,
    length_seconds INTEGER NOT NULL,
    PRIMARY KEY(music_id, source)
);

-- Playlist_Music table
CREATE TABLE IF NOT EXISTS playlist_music (
    user_id TEXT NOT NULL,
    playlist_id INTEGER NOT NULL,
    music_id TEXT NOT NULL,
    source INTEGER NOT NULL,
    added_at INTEGER NOT NULL,      -- Unix nano
    PRIMARY KEY(playlist_id, music_id, source),
    FOREIGN KEY(user_id, playlist_id) REFERENCES playlists(user_id, playlist_id) ON DELETE CASCADE,
    FOREIGN KEY(music_id, source) REFERENCES music(music_id, source) ON DELETE RESTRICT
);
