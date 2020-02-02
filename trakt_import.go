package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func importRequest(user *user, page int, endAt time.Time) (traktHistory, bool, error) {
	limit := 100
	u, err := url.Parse("https://api.trakt.tv/sync/history")
	if err != nil {
		return nil, false, err
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))

	if !endAt.IsZero() {
		q.Set("end_at", endAt.Format(time.RFC3339Nano))
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("trakt-api-key", traktClientID)
	req.Header.Set("trakt-api-version", "2")
	req.Header.Set("Authorization", "Bearer "+user.TraktOauth.AccessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	currentPage, err := strconv.Atoi(res.Header.Get("X-Pagination-Page"))
	if err != nil {
		return nil, false, err
	}

	totalPages, err := strconv.Atoi(res.Header.Get("X-Pagination-Page-Count"))
	if err != nil {
		return nil, false, err
	}

	var history traktHistory

	err = json.NewDecoder(res.Body).Decode(&history)
	if err != nil {
		return nil, false, err
	}

	return history, currentPage < totalPages, nil
}

func traktImport(user *user) {
	page := 1

	// First get the last activity
	// https://trakt.docs.apiary.io/#reference/sync/last-activities/get-last-activity
	// Compare to what we have now
	// Send only new
	// Save failures id's

	for {
		history, hasNext, err := importRequest(user, page, user.LastFetchedTime)
		if err != nil {
			log.Println(err)
			return
		}

		for _, record := range history {
			fmt.Println(record)
			// TODO: import
		}

		if hasNext {
			page = page + 1
		} else {
			break
		}
	}
}

func traktImportHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := mustUser(w, r)
	if user == nil {
		return
	}

	go traktImport(user)
	http.Redirect(w, r, "/?import=async", http.StatusTemporaryRedirect)
}
