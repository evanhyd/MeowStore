# MeowStore

MeowStore is a lightweight Go service for managing a user's music library and playlists. It exposes a JSON API over HTTP, stores data in SQLite, and authenticates requests with JWTs. The service is designed for client apps that need to read and write playlist metadata, music records, and playlist-to-music relations without a full application server stack.

## What it does

- Manage playlists per user
- Store music entries by source and identifier
- Associate music tracks with playlists
- Return full playlist content including music metadata and relation timestamps
- Support bulk music and playlist inserts
- Validate requests with JWT-based auth using the token subject as the user identifier

## Tech stack

- Go
- SQLite (via `modernc.org/sqlite`)
- JWT authentication (`github.com/golang-jwt/jwt/v5`)
- Standard library HTTP server

## Project structure

- `main.go` – bootstraps the server and registers routes
- `handlers/` – HTTP handlers and request/response models
- `storages/` – SQLite storage layer and schema definitions
- `loggers/` – logger setup
- `storages/schema.sql` – database schema

## Data model

### Playlist

```json
{
  "userId": "user-123",
  "playlistId": 42,
  "title": "Chill Mix",
  "modifiedDate": 1720000000000000000,
  "coverBlob": "base64-encoded-image-bytes"
}
```

### Music

```json
{
  "musicId": "abc123",
  "source": 1,
  "title": "Midnight City",
  "lengthSeconds": 240
}
```

### Playlist music relation

```json
{
  "userId": "user-123",
  "playlistId": 42,
  "musicId": "abc123",
  "source": 1,
  "modifiedDate": 1720000000000000000
}
```

### Supported music sources

```text
0 = UnknownSource
1 = YouTubeSource
2 = SpotifySource
```

## Authentication

Every request includes a `token` field. The server validates a JWT and reads the `sub` claim as the authenticated user ID.

Example JWT claim:

```json
{
  "sub": "user-123"
}
```

The service expects a JWT secret/key to be provided as a file via the `-key` flag.

## Running the server

### Build

```bash
go build -o meowstore .
```

### Run

```bash
./meowstore -db ./meowstore.db -key ./jwt.key -log ./meowstore.log -port 80
```

### Flags

- `-db` – SQLite database path
- `-key` – JWT secret file path
- `-log` – log file path
- `-port` – HTTP port (default `80`)

## API endpoints

All endpoints accept `POST` requests and expect JSON bodies. The `token` field is required for protected routes.

### Playlist endpoints

#### POST /api/getPlaylist

Fetch a single playlist by ID.

Request:

```json
{
  "token": "<jwt>",
  "playlistId": 42
}
```

Response:

```json
{
  "playlist": {
    "userId": "user-123",
    "playlistId": 42,
    "title": "Chill Mix",
    "modifiedDate": 1720000000000000000,
    "coverBlob": [255, 216, 255, 224]
  }
}
```

#### POST /api/getPlaylistContent

Fetch a playlist and all of its music-related data.

Request:

```json
{
  "token": "<jwt>",
  "playlistId": 42
}
```

Response:

```json
{
  "playlist": { "userId": "user-123", "playlistId": 42, "title": "Chill Mix" },
  "musics": [
    { "musicId": "abc123", "source": 1, "title": "Midnight City", "lengthSeconds": 240 }
  ],
  "relations": [
    { "userId": "user-123", "playlistId": 42, "musicId": "abc123", "source": 1, "modifiedDate": 1720000000000000000 }
  ]
}
```

#### POST /api/putPlaylist

Create or update a playlist for the authenticated user.

Request:

```json
{
  "token": "<jwt>",
  "playlist": {
    "playlistId": 42,
    "title": "Chill Mix",
    "modifiedDate": 1720000000000000000,
    "coverBlob": "base64-data"
  }
}
```

Response:

```json
{
  "playlist": {
    "userId": "user-123",
    "playlistId": 42,
    "title": "Chill Mix",
    "modifiedDate": 1720000000000000000,
    "coverBlob": "base64-data"
  }
}
```

#### POST /api/deletePlaylist

Delete a playlist owned by the authenticated user.

Request:

```json
{
  "token": "<jwt>",
  "playlistId": 42
}
```

Response:

```json
{}
```

#### POST /api/getPlaylists

List all playlists for the authenticated user.

Request:

```json
{
  "token": "<jwt>"
}
```

Response:

```json
{
  "playlists": [
    { "userId": "user-123", "playlistId": 1, "title": "Favorites" },
    { "userId": "user-123", "playlistId": 2, "title": "Road Trip" }
  ]
}
```

### Music endpoints

#### POST /api/getMusic

Fetch a single music record by ID and source.

Request:

```json
{
  "token": "<jwt>",
  "musicId": "abc123",
  "source": 1
}
```

Response:

```json
{
  "music": {
    "musicId": "abc123",
    "source": 1,
    "title": "Midnight City",
    "lengthSeconds": 240
  }
}
```

#### POST /api/putMusic

Insert or update music metadata.

Request:

```json
{
  "token": "<jwt>",
  "music": {
    "musicId": "abc123",
    "source": 1,
    "title": "Midnight City",
    "lengthSeconds": 240
  }
}
```

Response:

```json
{}
```

#### POST /api/putMusicBulk

Insert or update multiple music records in a single request.

Request:

```json
{
  "token": "<jwt>",
  "music": [
    { "musicId": "abc123", "source": 1, "title": "Midnight City", "lengthSeconds": 240 },
    { "musicId": "def456", "source": 2, "title": "Another Song", "lengthSeconds": 188 }
  ]
}
```

Response:

```json
{}
```

### Playlist relation endpoints

#### POST /api/putPlaylistMusic

Add a music track to a playlist.

Request:

```json
{
  "token": "<jwt>",
  "playlistMusic": {
    "playlistId": 42,
    "musicId": "abc123",
    "source": 1,
    "modifiedDate": 1720000000000000000
  }
}
```

Response:

```json
{}
```

#### POST /api/putPlaylistMusicBulk

Add many tracks to a playlist in one request.

Request:

```json
{
  "token": "<jwt>",
  "playlistMusic": [
    { "userId": "user-123", "playlistId": 42, "musicId": "abc123", "source": 1, "modifiedDate": 1720000000000000000 },
    { "userId": "user-123", "playlistId": 42, "musicId": "def456", "source": 2, "modifiedDate": 1720000000000000001 }
  ]
}
```

Response:

```json
{}
```

#### POST /api/deletePlaylistMusic

Remove a music entry from a playlist.

Request:

```json
{
  "token": "<jwt>",
  "playlistId": 42,
  "musicId": "abc123",
  "source": 1
}
```

Response:

```json
{}
```

## Error handling

The API returns JSON errors in the following format:

```json
{
  "error": "Unauthorized: ..."
}
```

Typical HTTP status codes include:

- `200 OK` – successful request
- `400 Bad Request` – malformed JSON or request payload
- `401 Unauthorized` – invalid or missing JWT
- `405 Method Not Allowed` – wrong HTTP method
- `500 Internal Server Error` – storage or server-side failure

## Database schema

The SQLite schema is defined in `storages/schema.sql` and creates three tables:

- `playlist`
- `music`
- `playlist_music`

Key relationships:

- `playlist` is keyed by `(user_id, playlist_id)`
- `music` is keyed by `(music_id, source)`
- `playlist_music` joins playlist and music records and keeps `added_at`
- Foreign keys are enabled, and playlist deletions cascade to related entries

## Example: generating a JWT

The service only cares that the token is valid and that `sub` is set to the user ID.

Example in Go:

```go
import (
    "fmt"
    "github.com/golang-jwt/jwt/v5"
)

func main() {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
        Subject: "user-123",
    })

    signed, err := token.SignedString([]byte("super-secret-key"))
    if err != nil {
        panic(err)
    }

    fmt.Println(signed)
}
```

## Notes

- This project is intentionally compact and backend-focused; it does not include a web front-end or user account system.
- The JWT secret file should be kept private and should be rotated for production use.
- The system assumes the client is responsible for constructing valid `musicId` and `source` values for external sources.

## License

This project is distributed under the MIT license. See `LICENSE` for details.
