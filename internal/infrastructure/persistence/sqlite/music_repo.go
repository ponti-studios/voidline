package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"voidline/internal/domain/music"
)

type MusicRepository struct {
	db *sql.DB
}

func NewMusicRepository(db *sql.DB) *MusicRepository {
	return &MusicRepository{db: db}
}

func (r *MusicRepository) UpsertArtist(ctx context.Context, name, platform string) (int64, error) {
	query := `
		INSERT INTO music_artists (name, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name, platform) DO UPDATE SET
			name = excluded.name,
			platform = excluded.platform,
			updated_at = excluded.updated_at
		RETURNING id
	`
	now := time.Now().Format(time.RFC3339)
	var id int64
	err := r.db.QueryRowContext(ctx, query, name, platform, now, now).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert artist: %w", err)
	}
	return id, nil
}

func (r *MusicRepository) GetArtistByName(ctx context.Context, name, platform string) (*music.Artist, error) {
	query := `SELECT id, name, platform, created_at, updated_at FROM music_artists WHERE name = ? AND platform = ?`
	var artist music.Artist
	var createdAt, updatedAt sql.NullString
	err := r.db.QueryRowContext(ctx, query, name, platform).Scan(
		&artist.ID, &artist.Name, &artist.Platform, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}
	if createdAt.Valid {
		t, _ := time.Parse(time.RFC3339, createdAt.String)
		artist.CreatedAt = &t
	}
	if updatedAt.Valid {
		t, _ := time.Parse(time.RFC3339, updatedAt.String)
		artist.UpdatedAt = &t
	}
	return &artist, nil
}

func (r *MusicRepository) UpsertAlbum(ctx context.Context, album *music.Album) (int64, error) {
	query := `
		INSERT INTO music_albums (
			name, platform, artist_id, genre, year, track_count, disc_count,
			date_created, date_added_to_library, last_modified, rating,
			is_compilation, is_purchased
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, artist_id, platform) DO UPDATE SET
			name = excluded.name,
			platform = excluded.platform,
			artist_id = excluded.artist_id,
			genre = excluded.genre,
			year = excluded.year,
			track_count = excluded.track_count,
			disc_count = excluded.disc_count,
			date_created = excluded.date_created,
			date_added_to_library = excluded.date_added_to_library,
			last_modified = excluded.last_modified,
			rating = excluded.rating,
			is_compilation = excluded.is_compilation,
			is_purchased = excluded.is_purchased
		RETURNING id
	`
	var rating sql.NullInt64
	if album.Rating != nil {
		rating = sql.NullInt64{Int64: int64(*album.Rating), Valid: true}
	}
	var year sql.NullInt32
	if album.Year != nil {
		year = sql.NullInt32{Int32: int32(*album.Year), Valid: true}
	}
	var trackCount, discCount sql.NullInt32
	if album.TrackCount != nil {
		trackCount = sql.NullInt32{Int32: int32(*album.TrackCount), Valid: true}
	}
	if album.DiscCount != nil {
		discCount = sql.NullInt32{Int32: int32(*album.DiscCount), Valid: true}
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		album.Name, album.Platform, album.ArtistID, album.Genre, year,
		trackCount, discCount, album.DateCreated, album.DateAddedToLibrary,
		album.LastModified, rating, album.IsCompilation, album.IsPurchased,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert album: %w", err)
	}
	return id, nil
}

func (r *MusicRepository) GetAlbumByName(ctx context.Context, name, platform string, artistID *int64) (*music.Album, error) {
	query := `SELECT id, name, platform, artist_id, genre, year, track_count, disc_count,
		date_created, date_added_to_library, last_modified, rating, is_compilation, is_purchased
		FROM music_albums WHERE name = ? AND platform = ?`
	var album music.Album
	var year, trackCount, discCount sql.NullInt32
	var rating sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, name, platform).Scan(
		&album.ID, &album.Name, &album.Platform, &album.ArtistID, &album.Genre,
		&year, &trackCount, &discCount, &album.DateCreated, &album.DateAddedToLibrary,
		&album.LastModified, &rating, &album.IsCompilation, &album.IsPurchased,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	if year.Valid {
		y := int(year.Int32)
		album.Year = &y
	}
	if trackCount.Valid {
		tc := int(trackCount.Int32)
		album.TrackCount = &tc
	}
	if discCount.Valid {
		dc := int(discCount.Int32)
		album.DiscCount = &dc
	}
	if rating.Valid {
		r := int(rating.Int64)
		album.Rating = &r
	}
	return &album, nil
}

func (r *MusicRepository) UpsertTrack(ctx context.Context, track *music.Track) (int64, error) {
	query := `
		INSERT INTO music_tracks (
			title, platform, album_id, artist_id, genre, duration_ms, track_number, disc_number,
			play_count, skip_count, rating, date_added_to_library, date_added_to_icloud,
			last_played, audio_extension, spotify_uri, apple_identifier
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(title, album_id, platform) DO UPDATE SET
			title = excluded.title,
			platform = excluded.platform,
			album_id = excluded.album_id,
			artist_id = excluded.artist_id,
			genre = excluded.genre,
			duration_ms = excluded.duration_ms,
			track_number = excluded.track_number,
			disc_number = excluded.disc_number,
			play_count = COALESCE(excluded.play_count, play_count),
			skip_count = COALESCE(excluded.skip_count, skip_count),
			rating = COALESCE(excluded.rating, rating),
			date_added_to_library = COALESCE(excluded.date_added_to_library, date_added_to_library),
			date_added_to_icloud = COALESCE(excluded.date_added_to_icloud, date_added_to_icloud),
			last_played = COALESCE(excluded.last_played, last_played),
			audio_extension = excluded.audio_extension,
			spotify_uri = COALESCE(excluded.spotify_uri, spotify_uri),
			apple_identifier = COALESCE(excluded.apple_identifier, apple_identifier)
		RETURNING id
	`
	var durationMs, trackNumber, discNumber sql.NullInt32
	var rating sql.NullInt64
	var playCount, skipCount sql.NullInt32
	var appleID sql.NullInt64

	if track.DurationMs != nil {
		durationMs = sql.NullInt32{Int32: int32(*track.DurationMs), Valid: true}
	}
	if track.TrackNumber != nil {
		trackNumber = sql.NullInt32{Int32: int32(*track.TrackNumber), Valid: true}
	}
	if track.DiscNumber != nil {
		discNumber = sql.NullInt32{Int32: int32(*track.DiscNumber), Valid: true}
	}
	if track.Rating > 0 {
		rating = sql.NullInt64{Int64: int64(track.Rating), Valid: true}
	}
	if track.PlayCount > 0 {
		playCount = sql.NullInt32{Int32: int32(track.PlayCount), Valid: true}
	}
	if track.SkipCount > 0 {
		skipCount = sql.NullInt32{Int32: int32(track.SkipCount), Valid: true}
	}
	if track.AppleIdentifier != nil {
		appleID = sql.NullInt64{Int64: *track.AppleIdentifier, Valid: true}
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		track.Title, track.Platform, track.AlbumID, track.ArtistID, track.Genre,
		durationMs, trackNumber, discNumber, playCount, skipCount, rating,
		track.DateAddedToLibrary, track.DateAddedToiCloud, track.LastPlayed,
		track.AudioExtension, track.SpotifyURI, appleID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert track: %w", err)
	}
	return id, nil
}

func (r *MusicRepository) GetTrackByTitle(ctx context.Context, title, platform string, albumID *int64) (*music.Track, error) {
	query := `SELECT id, title, platform, album_id, artist_id, genre, duration_ms, track_number,
		disc_number, play_count, skip_count, rating, date_added_to_library, date_added_to_icloud,
		last_played, audio_extension, spotify_uri, apple_identifier
		FROM music_tracks WHERE title = ? AND platform = ?`
	var track music.Track
	var durationMs, trackNumber, discNumber sql.NullInt32
	var playCount, skipCount sql.NullInt32
	var rating sql.NullInt64
	var appleID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, title, platform).Scan(
		&track.ID, &track.Title, &track.Platform, &track.AlbumID, &track.ArtistID,
		&track.Genre, &durationMs, &trackNumber, &discNumber, &playCount, &skipCount,
		&rating, &track.DateAddedToLibrary, &track.DateAddedToiCloud, &track.LastPlayed,
		&track.AudioExtension, &track.SpotifyURI, &appleID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	if durationMs.Valid {
		dm := int(durationMs.Int32)
		track.DurationMs = &dm
	}
	if trackNumber.Valid {
		tn := int(trackNumber.Int32)
		track.TrackNumber = &tn
	}
	if discNumber.Valid {
		dn := int(discNumber.Int32)
		track.DiscNumber = &dn
	}
	if playCount.Valid {
		track.PlayCount = int(playCount.Int32)
	}
	if skipCount.Valid {
		track.SkipCount = int(skipCount.Int32)
	}
	if rating.Valid {
		track.Rating = int(rating.Int64)
	}
	if appleID.Valid {
		ai := appleID.Int64
		track.AppleIdentifier = &ai
	}
	return &track, nil
}

func (r *MusicRepository) GetTrackByID(ctx context.Context, id int64) (*music.Track, error) {
	query := `SELECT id, title, platform, album_id, artist_id, genre, duration_ms, track_number,
		disc_number, play_count, skip_count, rating, date_added_to_library, date_added_to_icloud,
		last_played, audio_extension, spotify_uri, apple_identifier
		FROM music_tracks WHERE id = ?`
	var track music.Track
	var durationMs, trackNumber, discNumber sql.NullInt32
	var playCount, skipCount sql.NullInt32
	var rating sql.NullInt64
	var appleID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&track.ID, &track.Title, &track.Platform, &track.AlbumID, &track.ArtistID,
		&track.Genre, &durationMs, &trackNumber, &discNumber, &playCount, &skipCount,
		&rating, &track.DateAddedToLibrary, &track.DateAddedToiCloud, &track.LastPlayed,
		&track.AudioExtension, &track.SpotifyURI, &appleID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	if durationMs.Valid {
		dm := int(durationMs.Int32)
		track.DurationMs = &dm
	}
	if trackNumber.Valid {
		tn := int(trackNumber.Int32)
		track.TrackNumber = &tn
	}
	if discNumber.Valid {
		dn := int(discNumber.Int32)
		track.DiscNumber = &dn
	}
	if playCount.Valid {
		track.PlayCount = int(playCount.Int32)
	}
	if skipCount.Valid {
		track.SkipCount = int(skipCount.Int32)
	}
	if rating.Valid {
		track.Rating = int(rating.Int64)
	}
	if appleID.Valid {
		ai := appleID.Int64
		track.AppleIdentifier = &ai
	}
	return &track, nil
}

func (r *MusicRepository) UpsertPlaylist(ctx context.Context, playlist *music.Playlist) (int64, error) {
	query := `
		INSERT INTO music_playlists (
			name, platform, description, follower_count, container_type,
			is_favorite, favorite_date, created_at, modified_at, last_modified_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, platform) DO UPDATE SET
			name = excluded.name,
			platform = excluded.platform,
			description = COALESCE(excluded.description, description),
			follower_count = COALESCE(excluded.follower_count, follower_count),
			container_type = COALESCE(excluded.container_type, container_type),
			is_favorite = COALESCE(excluded.is_favorite, is_favorite),
			favorite_date = COALESCE(excluded.favorite_date, favorite_date),
			modified_at = COALESCE(excluded.modified_at, modified_at),
			last_modified_date = COALESCE(excluded.last_modified_date, last_modified_date)
		RETURNING id
	`
	var followerCount sql.NullInt32
	var favoriteDate sql.NullString

	if playlist.FollowerCount != nil {
		followerCount = sql.NullInt32{Int32: int32(*playlist.FollowerCount), Valid: true}
	}
	if playlist.FavoriteDate != nil {
		favoriteDate = sql.NullString{String: playlist.FavoriteDate.Format(time.RFC3339), Valid: true}
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		playlist.Name, playlist.Platform, playlist.Description, followerCount,
		playlist.ContainerType, playlist.IsFavorite, favoriteDate,
		playlist.CreatedAt, playlist.ModifiedAt, playlist.LastModifiedDate,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert playlist: %w", err)
	}
	return id, nil
}

func (r *MusicRepository) GetPlaylistByName(ctx context.Context, name, platform string) (*music.Playlist, error) {
	query := `SELECT id, name, platform, description, follower_count, container_type,
		is_favorite, favorite_date, created_at, modified_at, last_modified_date
		FROM music_playlists WHERE name = ? AND platform = ?`
	var playlist music.Playlist
	var followerCount sql.NullInt32
	var favoriteDate sql.NullString

	err := r.db.QueryRowContext(ctx, query, name, platform).Scan(
		&playlist.ID, &playlist.Name, &playlist.Platform, &playlist.Description,
		&followerCount, &playlist.ContainerType, &playlist.IsFavorite, &favoriteDate,
		&playlist.CreatedAt, &playlist.ModifiedAt, &playlist.LastModifiedDate,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}
	if followerCount.Valid {
		fc := int(followerCount.Int32)
		playlist.FollowerCount = &fc
	}
	if favoriteDate.Valid {
		t, _ := time.Parse(time.RFC3339, favoriteDate.String)
		playlist.FavoriteDate = &t
	}
	return &playlist, nil
}

func (r *MusicRepository) InsertPlaylistItem(ctx context.Context, item *music.PlaylistItem) error {
	query := `
		INSERT OR IGNORE INTO music_playlist_items (playlist_id, track_id, position, added_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, item.PlaylistID, item.TrackID, item.Position, item.AddedAt)
	if err != nil {
		return fmt.Errorf("failed to insert playlist item: %w", err)
	}
	return nil
}

func (r *MusicRepository) GetPlaylistItems(ctx context.Context, playlistID int64) ([]music.PlaylistItem, error) {
	query := `SELECT id, playlist_id, track_id, position, added_at FROM music_playlist_items WHERE playlist_id = ?`
	rows, err := r.db.QueryContext(ctx, query, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist items: %w", err)
	}
	defer rows.Close()

	var items []music.PlaylistItem
	for rows.Next() {
		var item music.PlaylistItem
		var position sql.NullInt32
		var addedAt sql.NullString
		err := rows.Scan(&item.ID, &item.PlaylistID, &item.TrackID, &position, &addedAt)
		if err != nil {
			continue
		}
		if position.Valid {
			p := int(position.Int32)
			item.Position = &p
		}
		if addedAt.Valid {
			t, _ := time.Parse(time.RFC3339, addedAt.String)
			item.AddedAt = &t
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *MusicRepository) InsertListeningHistory(ctx context.Context, history *music.ListeningHistory) error {
	query := `
		INSERT OR IGNORE INTO music_listening_history (
			track_id, platform, played_at, ms_played, country, ip_address, user_agent,
			reason_start, reason_end, shuffle, skipped, offline, incognito_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		history.TrackID, history.Platform, history.PlayedAt, history.MsPlayed,
		history.Country, history.IPAddress, history.UserAgent,
		history.ReasonStart, history.ReasonEnd, history.Shuffle, history.Skipped,
		history.Offline, history.IncognitoMode,
	)
	if err != nil {
		return fmt.Errorf("failed to insert listening history: %w", err)
	}
	return nil
}

func (r *MusicRepository) InsertListeningHistoryBatch(ctx context.Context, records []music.ListeningHistory) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO music_listening_history (
			track_id, platform, played_at, ms_played, country, ip_address, user_agent,
			reason_start, reason_end, shuffle, skipped, offline, incognito_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		_, err := stmt.ExecContext(ctx,
			record.TrackID, record.Platform, record.PlayedAt, record.MsPlayed,
			record.Country, record.IPAddress, record.UserAgent,
			record.ReasonStart, record.ReasonEnd, record.Shuffle, record.Skipped,
			record.Offline, record.IncognitoMode,
		)
		if err != nil {
			return fmt.Errorf("failed to insert listening history: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *MusicRepository) UpdateTrackPlayCount(ctx context.Context, trackID int64, playCount, skipCount int) error {
	query := `UPDATE music_tracks SET play_count = ?, skip_count = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, playCount, skipCount, trackID)
	if err != nil {
		return fmt.Errorf("failed to update track play count: %w", err)
	}
	return nil
}
