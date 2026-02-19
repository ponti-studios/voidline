-- +goose Up
-- Migration: music_domain
-- Description: Creates music tables (Spotify, Apple Music)
-- Created: February 2026

-- Artists
CREATE TABLE IF NOT EXISTS music_artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    platform TEXT NOT NULL,
    created_at TEXT,
    updated_at TEXT,
    UNIQUE(name, platform)
);
CREATE INDEX IF NOT EXISTS idx_music_artists_name ON music_artists(name);
CREATE INDEX IF NOT EXISTS idx_music_artists_platform ON music_artists(platform);

-- Albums
CREATE TABLE IF NOT EXISTS music_albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    platform TEXT NOT NULL,
    artist_id INTEGER REFERENCES music_artists(id),
    genre TEXT,
    year INTEGER,
    track_count INTEGER,
    disc_count INTEGER,
    date_created TEXT,
    date_added_to_library TEXT,
    last_modified TEXT,
    rating INTEGER,
    is_compilation BOOLEAN DEFAULT 0,
    is_purchased BOOLEAN DEFAULT 0,
    UNIQUE(name, artist_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_music_albums_name ON music_albums(name);
CREATE INDEX IF NOT EXISTS idx_music_albums_artist ON music_albums(artist_id);
CREATE INDEX IF NOT EXISTS idx_music_albums_platform ON music_albums(platform);

-- Tracks
CREATE TABLE IF NOT EXISTS music_tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    platform TEXT NOT NULL,
    album_id INTEGER REFERENCES music_albums(id),
    artist_id INTEGER REFERENCES music_artists(id),
    genre TEXT,
    duration_ms INTEGER,
    track_number INTEGER,
    disc_number INTEGER,
    play_count INTEGER DEFAULT 0,
    skip_count INTEGER DEFAULT 0,
    rating INTEGER DEFAULT 0,
    date_added_to_library TEXT,
    date_added_to_icloud TEXT,
    last_played TEXT,
    audio_extension TEXT,
    spotify_uri TEXT,
    apple_identifier INTEGER,
    UNIQUE(title, album_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_music_tracks_title ON music_tracks(title);
CREATE INDEX IF NOT EXISTS idx_music_tracks_album ON music_tracks(album_id);
CREATE INDEX IF NOT EXISTS idx_music_tracks_artist ON music_tracks(artist_id);
CREATE INDEX IF NOT EXISTS idx_music_tracks_platform ON music_tracks(platform);

-- Playlists
CREATE TABLE IF NOT EXISTS music_playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    platform TEXT NOT NULL,
    description TEXT,
    follower_count INTEGER DEFAULT 0,
    container_type TEXT,
    is_favorite BOOLEAN DEFAULT 0,
    favorite_date TEXT,
    created_at TEXT,
    modified_at TEXT,
    last_modified_date TEXT,
    UNIQUE(name, platform)
);
CREATE INDEX IF NOT EXISTS idx_music_playlists_name ON music_playlists(name);
CREATE INDEX IF NOT EXISTS idx_music_playlists_platform ON music_playlists(platform);

-- Playlist Items
CREATE TABLE IF NOT EXISTS music_playlist_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    playlist_id INTEGER REFERENCES music_playlists(id),
    track_id INTEGER REFERENCES music_tracks(id),
    position INTEGER,
    added_at TEXT,
    UNIQUE(playlist_id, track_id)
);
CREATE INDEX IF NOT EXISTS idx_music_playlist_items_playlist ON music_playlist_items(playlist_id);
CREATE INDEX IF NOT EXISTS idx_music_playlist_items_track ON music_playlist_items(track_id);

-- Listening History
CREATE TABLE IF NOT EXISTS music_listening_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id INTEGER REFERENCES music_tracks(id),
    platform TEXT NOT NULL,
    played_at TEXT NOT NULL,
    ms_played INTEGER DEFAULT 0,
    country TEXT,
    ip_address TEXT,
    user_agent TEXT,
    reason_start TEXT,
    reason_end TEXT,
    shuffle BOOLEAN DEFAULT 0,
    skipped BOOLEAN DEFAULT 0,
    offline BOOLEAN DEFAULT 0,
    incognito_mode BOOLEAN DEFAULT 0,
    UNIQUE(track_id, played_at, platform)
);
CREATE INDEX IF NOT EXISTS idx_music_listening_history_track ON music_listening_history(track_id);
CREATE INDEX IF NOT EXISTS idx_music_listening_history_played_at ON music_listening_history(played_at);
CREATE INDEX IF NOT EXISTS idx_music_listening_history_platform ON music_listening_history(platform);

-- +goose Down
DROP TABLE IF EXISTS music_listening_history;
DROP TABLE IF EXISTS music_playlist_items;
DROP TABLE IF EXISTS music_playlists;
DROP TABLE IF EXISTS music_tracks;
DROP TABLE IF EXISTS music_albums;
DROP TABLE IF EXISTS music_artists;
