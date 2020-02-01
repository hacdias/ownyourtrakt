package trakt

import "time"

// https://trakt.docs.apiary.io/#introduction/standard-media-objects

type generic struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   IDs    `json:"ids"`
}

type Movie generic

type Show generic

type Episode struct {
	Title  string `json:"title"`
	Season int    `json:"season"`
	Number int    `json:"number"`
	IDs    IDs    `json:"ids"`
}

type IDs struct {
	Trakt int    `json:"trakt"`
	IMDb  string `json:"imdb"`
	TMDb  int    `json:"tmdb"`
	Slug  string `json:"slug,omitempty"`
	TVDb  int    `json:"tvdb,omitempty"`
}

type HistoryItem struct {
	ID        int       `json:"id"`
	WatchedAt time.Time `json:"watched_at"`
	Action    string    `json:"action"`
	Type      string    `json:"type"`
	Movie     Movie     `json:"movie"`
	Episode   Episode   `json:"episode"`
	Show      Show      `json:"show"`
}

type History []HistoryItem
