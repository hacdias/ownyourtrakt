package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

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
	ID        int64        `json:"id"`
	WatchedAt time.Time    `json:"watched_at"`
	Action    string       `json:"action"`
	Type      string       `json:"type"`
	Movie     traktMovie   `json:"movie"`
	Episode   traktEpisode `json:"episode"`
	Show      traktShow    `json:"show"`
}

type traktHistory []traktHistoryItem

func traktToMicroformats(item traktHistoryItem) (interface{}, error) {
	watch := map[string]interface{}{}
	watch["trakt-watch-id"] = []int64{item.ID}

	summary := ""

	if item.Type == "episode" {
		episodeOf := map[string]interface{}{}

		episodeOf["name"] = []string{item.Show.Title}
		episodeOf["url"] = []string{"https://trakt.tv/shows/" + item.Show.IDs.Slug}
		episodeOf["published"] = []string{time.Date(item.Show.Year, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)}
		episodeOf["trakt-ids"] = item.Show.IDs

		watch["name"] = []string{item.Episode.Title}
		watch["url"] = []string{
			"https://trakt.tv/shows/" +
				item.Show.IDs.Slug +
				"/seasons/" +
				strconv.Itoa(item.Episode.Season) +
				"/episodes/" +
				strconv.Itoa(item.Episode.Number),
		}
		watch["episode"] = []int{item.Episode.Number}
		watch["season"] = []int{item.Episode.Season}
		watch["trakt-ids"] = item.Episode.IDs
		watch["episode-of"] = []interface{}{
			map[string]interface{}{
				"type":       []string{"h-cite"},
				"properties": episodeOf,
			},
		}

		summary = fmt.Sprintf("Just watched: %s (%s S%dE%d)", item.Episode.Title, item.Show.Title, item.Episode.Season, item.Episode.Number)
	} else if item.Type == "movie" {
		watch["name"] = []string{item.Movie.Title}
		watch["url"] = []string{"https://trakt.tv/movies/" + item.Movie.IDs.Slug}
		watch["published"] = []string{time.Date(item.Movie.Year, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)}
		watch["trakt-ids"] = item.Movie.IDs

		summary = "Just watched: " + item.Movie.Title
	} else {
		return nil, errors.New("invalid type " + item.Type)
	}

	mf2 := map[string]interface{}{
		"type": []string{"h-entry"},
		"properties": map[string]interface{}{
			"published": []string{item.WatchedAt.Format(time.RFC3339)},
			"summary":   []string{summary},
			"watch-of": []interface{}{
				map[string]interface{}{
					"type":       []string{"h-cite"},
					"properties": watch,
				},
			},
		},
	}

	return mf2, nil
}
