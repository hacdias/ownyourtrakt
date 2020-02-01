package main

import "time"

// https://trakt.docs.apiary.io/#introduction/standard-media-objects

type traktGeneric struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   traktIDs `json:"ids"`
}

type traktMovie traktGeneric

type traktShow traktGeneric

type traktEpisode struct {
	Title  string   `json:"title"`
	Season int      `json:"season"`
	Number int      `json:"number"`
	IDs    traktIDs `json:"ids"`
}

type traktIDs struct {
	Trakt int    `json:"trakt"`
	IMDb  string `json:"imdb"`
	TMDb  int    `json:"tmdb"`
	Slug  string `json:"slug,omitempty"`
	TVDb  int    `json:"tvdb,omitempty"`
}

type traktHistoryItem struct {
	ID        int          `json:"id"`
	WatchedAt time.Time    `json:"watched_at"`
	Action    string       `json:"action"`
	Type      string       `json:"type"`
	Movie     traktMovie   `json:"movie"`
	Episode   traktEpisode `json:"episode"`
	Show      traktShow    `json:"show"`
}

type traktHistory []traktHistoryItem
