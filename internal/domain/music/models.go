package music

import "time"

type Artist struct {
	ID        int64
	Name      string
	Platform  string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

type Album struct {
	ID                 int64
	Name               string
	Platform           string
	ArtistID           *int64
	Genre              *string
	Year               *int
	TrackCount         *int
	DiscCount          *int
	DateCreated        *time.Time
	DateAddedToLibrary *time.Time
	LastModified       *time.Time
	Rating             *int
	IsCompilation      bool
	IsPurchased        bool
}

type Track struct {
	ID                 int64
	Title              string
	Platform           string
	AlbumID            *int64
	ArtistID           *int64
	Genre              *string
	DurationMs         *int
	TrackNumber        *int
	DiscNumber         *int
	PlayCount          int
	SkipCount          int
	Rating             int
	DateAddedToLibrary *time.Time
	DateAddedToiCloud  *time.Time
	LastPlayed         *time.Time
	AudioExtension     *string
	SpotifyURI         *string
	AppleIdentifier    *int64
}

type Playlist struct {
	ID               int64
	Name             string
	Platform         string
	Description      *string
	FollowerCount    *int
	ContainerType    *string
	IsFavorite       bool
	FavoriteDate     *time.Time
	CreatedAt        *time.Time
	ModifiedAt       *time.Time
	LastModifiedDate *time.Time
}

type PlaylistItem struct {
	ID         int64
	PlaylistID int64
	TrackID    int64
	Position   *int
	AddedAt    *time.Time
}

type ListeningHistory struct {
	ID            int64
	TrackID       int64
	Platform      string
	PlayedAt      time.Time
	MsPlayed      *int
	Country       *string
	IPAddress     *string
	UserAgent     *string
	ReasonStart   *string
	ReasonEnd     *string
	Shuffle       bool
	Skipped       bool
	Offline       bool
	IncognitoMode bool
}

type ImportResult struct {
	TotalArtists          int
	TotalAlbums           int
	TotalTracks           int
	TotalPlaylists        int
	TotalPlaylistItems    int
	TotalListeningHistory int
	Inserted              int
	Skipped               int
	Errors                []ImportError
	Duration              time.Duration
}

type ImportError struct {
	Row  int
	Col  int
	Err  error
	Data map[string]string
}

func (e ImportError) Error() string {
	return e.Err.Error()
}

func (r ImportResult) IsSuccess() bool {
	return len(r.Errors) == 0
}
