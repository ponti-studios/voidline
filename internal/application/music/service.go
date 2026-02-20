package music

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"voidline/internal/domain/music"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type Service struct {
	repo *sqlite.MusicRepository
}

func NewService(repo *sqlite.MusicRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	DryRun bool
	Force  bool
}

func (s *Service) ImportSpotify(ctx context.Context, sourceDir string, options ImportOptions) (*music.ImportResult, error) {
	result := &music.ImportResult{}
	start := time.Now()

	fmt.Println("Importing Spotify data from:", sourceDir)

	if err := s.importSpotifyStreamingHistory(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing streaming history: %v\n", err)
	}

	if err := s.importSpotifyExtendedStreamingHistory(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing extended streaming history: %v\n", err)
	}

	if err := s.importSpotifyPlaylists(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing playlists: %v\n", err)
	}

	if err := s.importSpotifyLibrary(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing library: %v\n", err)
	}

	result.Duration = time.Since(start)
	fmt.Printf("Spotify import complete. Artists: %d, Albums: %d, Tracks: %d, Playlists: %d, PlaylistItems: %d, ListeningHistory: %d\n",
		result.TotalArtists, result.TotalAlbums, result.TotalTracks, result.TotalPlaylists,
		result.TotalPlaylistItems, result.TotalListeningHistory)

	return result, nil
}

func (s *Service) importSpotifyStreamingHistory(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Spotify Streaming History...")

	for i := 0; i < 2; i++ {
		filePath := filepath.Join(sourceDir, fmt.Sprintf("StreamingHistory%d.json", i))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var records []map[string]interface{}
		if err := json.Unmarshal(data, &records); err != nil {
			fmt.Printf("Error parsing %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("Processing %d records from %s...\n", len(records), filepath.Base(filePath))

		var historyBatch []music.ListeningHistory

		for _, r := range records {
			artistName := toString(r["artistName"])
			trackName := toString(r["trackName"])
			msPlayed := toInt(r["msPlayed"])
			endTime := toString(r["endTime"])

			if artistName == "" || trackName == "" {
				continue
			}

			artistID, err := s.repo.UpsertArtist(ctx, artistName, "spotify")
			if err != nil {
				continue
			}
			result.TotalArtists++

			albumName := ""
			albumID, _ := s.repo.UpsertAlbum(ctx, &music.Album{
				Name:     albumName,
				Platform: "spotify",
				ArtistID: &artistID,
			})

			track := &music.Track{
				Title:      trackName,
				Platform:   "spotify",
				AlbumID:    &albumID,
				ArtistID:   &artistID,
				DurationMs: &msPlayed,
			}
			trackID, err := s.repo.UpsertTrack(ctx, track)
			if err != nil {
				continue
			}
			result.TotalTracks++

			playedAt, _ := time.Parse("2006-01-02 15:04", endTime)
			history := music.ListeningHistory{
				TrackID:  trackID,
				Platform: "spotify",
				PlayedAt: playedAt,
				MsPlayed: &msPlayed,
			}
			historyBatch = append(historyBatch, history)

			if len(historyBatch) >= 1000 {
				if err := s.repo.InsertListeningHistoryBatch(ctx, historyBatch); err == nil {
					result.TotalListeningHistory += len(historyBatch)
				}
				historyBatch = nil
			}
		}

		if len(historyBatch) > 0 {
			if err := s.repo.InsertListeningHistoryBatch(ctx, historyBatch); err == nil {
				result.TotalListeningHistory += len(historyBatch)
			}
		}
	}

	return nil
}

func (s *Service) importSpotifyExtendedStreamingHistory(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Spotify Extended Streaming History...")

	dir := filepath.Join(sourceDir, "Spotify Extended Streaming History")
	matches, err := filepath.Glob(filepath.Join(dir, "Streaming_History_Audio_*.json"))
	if err != nil || len(matches) == 0 {
		fmt.Printf("No Spotify Extended files found at %s\n", dir)
		return nil
	}

	fmt.Printf("Found %d extended JSON files.\n", len(matches))

	var historyBatch []music.ListeningHistory

	for _, jsonFile := range matches {
		fmt.Printf("Processing %s...\n", filepath.Base(jsonFile))
		fileData, err := os.ReadFile(jsonFile)
		if err != nil {
			continue
		}

		var data []map[string]interface{}
		if err := json.Unmarshal(fileData, &data); err != nil {
			fmt.Printf("Error decoding %s: %v\n", jsonFile, err)
			continue
		}

		for _, r := range data {
			trackName := toString(r["master_metadata_track_name"])
			artistName := toString(r["master_metadata_album_artist_name"])
			albumName := toString(r["master_metadata_album_album_name"])

			if trackName == "" || artistName == "" {
				continue
			}

			artistID, err := s.repo.UpsertArtist(ctx, artistName, "spotify")
			if err != nil {
				continue
			}
			result.TotalArtists++

			albumID, _ := s.repo.UpsertAlbum(ctx, &music.Album{
				Name:     albumName,
				Platform: "spotify",
				ArtistID: &artistID,
			})
			result.TotalAlbums++

			msPlayed := toInt(r["ms_played"])
			track := &music.Track{
				Title:      trackName,
				Platform:   "spotify",
				AlbumID:    &albumID,
				ArtistID:   &artistID,
				DurationMs: &msPlayed,
				SpotifyURI: strPtr(toString(r["spotify_track_uri"])),
			}
			trackID, err := s.repo.UpsertTrack(ctx, track)
			if err != nil {
				continue
			}
			result.TotalTracks++

			ts := toString(r["ts"])
			playedAt, _ := time.Parse(time.RFC3339, ts)

			history := music.ListeningHistory{
				TrackID:       trackID,
				Platform:      "spotify",
				PlayedAt:      playedAt,
				MsPlayed:      &msPlayed,
				Country:       strPtr(toString(r["conn_country"])),
				IPAddress:     strPtr(toString(r["ip_addr_decrypted"])),
				UserAgent:     strPtr(toString(r["user_agent_decrypted"])),
				ReasonStart:   strPtr(toString(r["reason_start"])),
				ReasonEnd:     strPtr(toString(r["reason_end"])),
				Shuffle:       toBool(r["shuffle"]),
				Skipped:       toBool(r["skipped"]),
				Offline:       toBool(r["offline"]),
				IncognitoMode: toBool(r["incognito_mode"]),
			}
			historyBatch = append(historyBatch, history)

			if len(historyBatch) >= 1000 {
				if err := s.repo.InsertListeningHistoryBatch(ctx, historyBatch); err == nil {
					result.TotalListeningHistory += len(historyBatch)
				}
				historyBatch = nil
			}
		}
	}

	if len(historyBatch) > 0 {
		if err := s.repo.InsertListeningHistoryBatch(ctx, historyBatch); err == nil {
			result.TotalListeningHistory += len(historyBatch)
		}
	}

	return nil
}

func (s *Service) importSpotifyPlaylists(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Spotify Playlists...")

	playlistPath := filepath.Join(sourceDir, "Playlist1.json")
	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		fmt.Println("No Playlist1.json found")
		return nil
	}

	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("failed to read Playlist1.json: %w", err)
	}

	var playlistData map[string]interface{}
	if err := json.Unmarshal(data, &playlistData); err != nil {
		return fmt.Errorf("failed to parse Playlist1.json: %w", err)
	}

	playlists, ok := playlistData["playlists"].([]interface{})
	if !ok {
		return nil
	}

	fmt.Printf("Found %d Spotify playlists.\n", len(playlists))

	for _, pl := range playlists {
		p := pl.(map[string]interface{})
		name := toString(p["name"])
		if name == "" {
			continue
		}

		lastModified := toString(p["lastModifiedDate"])
		var modifiedAt *time.Time
		if lastModified != "" {
			if t, err := time.Parse("2006-01-02", lastModified); err == nil {
				modifiedAt = &t
			}
		}

		description := toString(p["description"])
		var desc *string
		if description != "" {
			desc = &description
		}

		followerCount := toInt(p["numberOfFollowers"])
		var fc *int
		if followerCount > 0 {
			fc = &followerCount
		}

		playlist := &music.Playlist{
			Name:             name,
			Platform:         "spotify",
			Description:      desc,
			FollowerCount:    fc,
			LastModifiedDate: modifiedAt,
		}

		playlistID, err := s.repo.UpsertPlaylist(ctx, playlist)
		if err != nil {
			continue
		}
		result.TotalPlaylists++

		items, ok := p["items"].([]interface{})
		if !ok {
			continue
		}

		for position, item := range items {
			it, ok := item.(map[string]interface{})["track"].(map[string]interface{})
			if !ok {
				continue
			}

			trackName := toString(it["trackName"])
			artistName := toString(it["artistName"])
			albumName := toString(it["albumName"])

			if trackName == "" {
				continue
			}

			var artistID int64
			if artistName != "" {
				artistID, _ = s.repo.UpsertArtist(ctx, artistName, "spotify")
				result.TotalArtists++
			}

			var albumID int64
			if albumName != "" {
				albumID, _ = s.repo.UpsertAlbum(ctx, &music.Album{
					Name:     albumName,
					Platform: "spotify",
					ArtistID: &artistID,
				})
				result.TotalAlbums++
			}

			pos := position + 1
			track := &music.Track{
				Title:    trackName,
				Platform: "spotify",
				AlbumID:  &albumID,
				ArtistID: &artistID,
			}
			trackID, err := s.repo.UpsertTrack(ctx, track)
			if err != nil {
				continue
			}
			result.TotalTracks++

			playlistItem := &music.PlaylistItem{
				PlaylistID: playlistID,
				TrackID:    trackID,
				Position:   &pos,
			}
			if err := s.repo.InsertPlaylistItem(ctx, playlistItem); err == nil {
				result.TotalPlaylistItems++
			}
		}
	}

	return nil
}

func (s *Service) importSpotifyLibrary(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Spotify Library...")

	libraryPath := filepath.Join(sourceDir, "YourLibrary.json")
	if _, err := os.Stat(libraryPath); os.IsNotExist(err) {
		fmt.Println("No YourLibrary.json found")
		return nil
	}

	data, err := os.ReadFile(libraryPath)
	if err != nil {
		return fmt.Errorf("failed to read YourLibrary.json: %w", err)
	}

	var library map[string]interface{}
	if err := json.Unmarshal(data, &library); err != nil {
		return fmt.Errorf("failed to parse YourLibrary.json: %w", err)
	}

	if tracks, ok := library["tracks"].([]interface{}); ok {
		fmt.Printf("Processing %d library tracks...\n", len(tracks))
		for _, t := range tracks {
			tr := t.(map[string]interface{})
			trackName := toString(tr["track"])
			artistName := toString(tr["artist"])
			albumName := toString(tr["album"])

			if trackName == "" {
				continue
			}

			var artistID int64
			if artistName != "" {
				artistID, _ = s.repo.UpsertArtist(ctx, artistName, "spotify")
				result.TotalArtists++
			}

			var albumID int64
			if albumName != "" {
				albumID, _ = s.repo.UpsertAlbum(ctx, &music.Album{
					Name:     albumName,
					Platform: "spotify",
					ArtistID: &artistID,
				})
				result.TotalAlbums++
			}

			track := &music.Track{
				Title:    trackName,
				Platform: "spotify",
				AlbumID:  &albumID,
				ArtistID: &artistID,
			}
			if _, err := s.repo.UpsertTrack(ctx, track); err == nil {
				result.TotalTracks++
			}
		}
	}

	if albums, ok := library["albums"].([]interface{}); ok {
		fmt.Printf("Processing %d library albums...\n", len(albums))
		for _, a := range albums {
			al := a.(map[string]interface{})
			albumName := toString(al["album"])
			artistName := toString(al["artist"])

			if albumName == "" {
				continue
			}

			var artistID int64
			if artistName != "" {
				artistID, _ = s.repo.UpsertArtist(ctx, artistName, "spotify")
				result.TotalArtists++
			}

			album := &music.Album{
				Name:     albumName,
				Platform: "spotify",
				ArtistID: &artistID,
			}
			if _, err := s.repo.UpsertAlbum(ctx, album); err == nil {
				result.TotalAlbums++
			}
		}
	}

	return nil
}

func (s *Service) ImportAppleMusic(ctx context.Context, sourceDir string, options ImportOptions) (*music.ImportResult, error) {
	result := &music.ImportResult{}
	start := time.Now()

	fmt.Println("Importing Apple Music data from:", sourceDir)

	if err := s.importAppleMusicLibraryTracks(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing library tracks: %v\n", err)
	}

	if err := s.importAppleMusicLibraryAlbums(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing library albums: %v\n", err)
	}

	if err := s.importAppleMusicLibraryArtists(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing library artists: %v\n", err)
	}

	if err := s.importAppleMusicLibraryPlaylists(ctx, sourceDir, result); err != nil {
		fmt.Printf("Warning: Error importing playlists: %v\n", err)
	}

	result.Duration = time.Since(start)
	fmt.Printf("Apple Music import complete. Artists: %d, Albums: %d, Tracks: %d, Playlists: %d, PlaylistItems: %d\n",
		result.TotalArtists, result.TotalAlbums, result.TotalTracks, result.TotalPlaylists, result.TotalPlaylistItems)

	return result, nil
}

func (s *Service) importAppleMusicLibraryTracks(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Apple Music Library Tracks...")

	filePath := filepath.Join(sourceDir, "Apple Music Library Tracks.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("No Apple Music Library Tracks.json found")
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Apple Music Library Tracks.json: %w", err)
	}

	var tracks []map[string]interface{}
	if err := json.Unmarshal(data, &tracks); err != nil {
		return fmt.Errorf("failed to parse Apple Music Library Tracks.json: %w", err)
	}

	fmt.Printf("Processing %d tracks...\n", len(tracks))

	for _, t := range tracks {
		trackTitle := toString(t["Title"])
		artistName := toString(t["Artist"])
		albumName := toString(t["Album"])
		albumArtist := toString(t["Album Artist"])
		genre := toString(t["Genre"])

		if trackTitle == "" {
			continue
		}

		if artistName == "" {
			artistName = albumArtist
		}

		var artistID int64
		if artistName != "" {
			artistID, _ = s.repo.UpsertArtist(ctx, artistName, "apple_music")
			result.TotalArtists++
		}

		var albumID int64
		if albumName != "" {
			album := &music.Album{
				Name:     albumName,
				Platform: "apple_music",
				ArtistID: &artistID,
				Genre:    strPtr(genre),
			}
			if year, ok := t["Track Year"].(float64); ok && year > 0 {
				y := int(year)
				album.Year = &y
			}
			if trackCount, ok := t["Track Count On Album"].(float64); ok && trackCount > 0 {
				tc := int(trackCount)
				album.TrackCount = &tc
			}
			if discCount, ok := t["Disc Count On Album"].(float64); ok && discCount > 0 {
				dc := int(discCount)
				album.DiscCount = &dc
			}
			albumID, _ = s.repo.UpsertAlbum(ctx, album)
			result.TotalAlbums++
		}

		durationMs := toInt(t["Track Duration"])
		trackNumber := toInt(t["Track Number On Album"])
		discNumber := toInt(t["Disc Number On Album"])
		playCount := toInt(t["Track Play Count"])
		skipCount := toInt(t["Skip Count"])
		rating := toInt(t["Rating"])

		var audioExt *string
		if ext := toString(t["Audio File Extension"]); ext != "" {
			audioExt = &ext
		}

		var dateAdded *time.Time
		if dateStr := toString(t["Date Added To Library"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				dateAdded = &t
			}
		}

		var dateiCloud *time.Time
		if dateStr := toString(t["Date Added To iCloud Music Library"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				dateiCloud = &t
			}
		}

		var lastModified *time.Time
		if dateStr := toString(t["Last Modified Date"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				lastModified = &t
			}
		}

		var appleID int64
		if id, ok := t["Track Identifier"].(float64); ok {
			appleID = int64(id)
		}

		isCompilation := toString(t["Is Part Of Compilation"]) == "true"
		isPurchased := toString(t["Is Purchased"]) == "true"

		track := &music.Track{
			Title:              trackTitle,
			Platform:           "apple_music",
			AlbumID:            &albumID,
			ArtistID:           &artistID,
			Genre:              strPtr(genre),
			DurationMs:         &durationMs,
			TrackNumber:        &trackNumber,
			DiscNumber:         &discNumber,
			PlayCount:          playCount,
			SkipCount:          skipCount,
			Rating:             rating,
			DateAddedToLibrary: dateAdded,
			DateAddedToiCloud:  dateiCloud,
			LastPlayed:         lastModified,
			AudioExtension:     audioExt,
			AppleIdentifier:    &appleID,
		}

		if isCompilation {
			album, _ := s.repo.GetAlbumByName(ctx, albumName, "apple_music", &artistID)
			if album != nil {
				album.IsCompilation = true
				s.repo.UpsertAlbum(ctx, album)
			}
		}
		if isPurchased {
			album, _ := s.repo.GetAlbumByName(ctx, albumName, "apple_music", &artistID)
			if album != nil {
				album.IsPurchased = true
				s.repo.UpsertAlbum(ctx, album)
			}
		}

		if _, err := s.repo.UpsertTrack(ctx, track); err == nil {
			result.TotalTracks++
		}
	}

	return nil
}

func (s *Service) importAppleMusicLibraryAlbums(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Apple Music Library Albums...")

	filePath := filepath.Join(sourceDir, "Apple Music Library Albums.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Apple Music Library Albums.json: %w", err)
	}

	var albums []map[string]interface{}
	if err := json.Unmarshal(data, &albums); err != nil {
		return fmt.Errorf("failed to parse Apple Music Library Albums.json: %w", err)
	}

	fmt.Printf("Processing %d albums...\n", len(albums))

	for _, a := range albums {
		title := toString(a["Title"])
		if title == "" {
			continue
		}

		album := &music.Album{
			Name:     title,
			Platform: "apple_music",
		}

		if dateStr := toString(a["Date Created"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				album.DateCreated = &t
			}
		}

		if dateStr := toString(a["Date Created In Library"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				album.DateAddedToLibrary = &t
			}
		}

		if _, err := s.repo.UpsertAlbum(ctx, album); err == nil {
			result.TotalAlbums++
		}
	}

	return nil
}

func (s *Service) importAppleMusicLibraryArtists(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Apple Music Library Artists...")

	filePath := filepath.Join(sourceDir, "Apple Music Library Artists.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Apple Music Library Artists.json: %w", err)
	}

	var artists []map[string]interface{}
	if err := json.Unmarshal(data, &artists); err != nil {
		return fmt.Errorf("failed to parse Apple Music Library Artists.json: %w", err)
	}

	fmt.Printf("Processing %d artists...\n", len(artists))

	for _, a := range artists {
		name := toString(a["Artist Name"])
		if name == "" {
			continue
		}

		if _, err := s.repo.UpsertArtist(ctx, name, "apple_music"); err == nil {
			result.TotalArtists++
		}
	}

	return nil
}

func (s *Service) importAppleMusicLibraryPlaylists(ctx context.Context, sourceDir string, result *music.ImportResult) error {
	fmt.Println("Importing Apple Music Library Playlists...")

	filePath := filepath.Join(sourceDir, "Apple Music Library Playlists.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Apple Music Library Playlists.json: %w", err)
	}

	var playlists []map[string]interface{}
	if err := json.Unmarshal(data, &playlists); err != nil {
		return fmt.Errorf("failed to parse Apple Music Library Playlists.json: %w", err)
	}

	fmt.Printf("Processing %d playlists...\n", len(playlists))

	for _, p := range playlists {
		title := toString(p["Title"])
		if title == "" {
			continue
		}

		containerType := toString(p["Container Type"])
		isFavorite := toString(p["Favorite Status - Playlist"]) == "true"

		var favoriteDate *time.Time
		if dateStr := toString(p["Favorite Date - Playlist"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				favoriteDate = &t
			}
		}

		var createdAt *time.Time
		if dateStr := toString(p["Added Date"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				createdAt = &t
			}
		}

		var modifiedAt *time.Time
		if dateStr := toString(p["Name or Description Modified Date"]); dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				modifiedAt = &t
			}
		}

		playlist := &music.Playlist{
			Name:          title,
			Platform:      "apple_music",
			ContainerType: strPtr(containerType),
			IsFavorite:    isFavorite,
			FavoriteDate:  favoriteDate,
			CreatedAt:     createdAt,
			ModifiedAt:    modifiedAt,
		}

		playlistID, err := s.repo.UpsertPlaylist(ctx, playlist)
		if err != nil {
			continue
		}
		result.TotalPlaylists++

		if itemIDs, ok := p["Playlist Item Identifiers"].([]interface{}); ok {
			for position, itemID := range itemIDs {
				trackIDFloat, ok := itemID.(float64)
				if !ok {
					continue
				}
				trackID := int64(trackIDFloat)

				existingTrack, err := s.repo.GetTrackByID(ctx, trackID)
				if err != nil || existingTrack == nil {
					continue
				}

				pos := position + 1
				playlistItem := &music.PlaylistItem{
					PlaylistID: playlistID,
					TrackID:    existingTrack.ID,
					Position:   &pos,
					AddedAt:    createdAt,
				}
				if err := s.repo.InsertPlaylistItem(ctx, playlistItem); err == nil {
					result.TotalPlaylistItems++
				}
			}
		}
	}

	return nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	}
	return false
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
