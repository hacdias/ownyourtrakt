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

func importRequest(user *user, page int, startAt time.Time) (traktHistory, bool, error) {
	limit := 100
	u, err := url.Parse("https://api.trakt.tv/sync/history")
	if err != nil {
		return nil, false, err
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))

	if !startAt.IsZero() {
		q.Set("start_at", startAt.Format(time.RFC3339Nano))
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

	for {
		history, hasNext, err := importRequest(user, page, user.LastFetchedTime)
		if err != nil {
			log.Println(err)
			return
		}

		for i, record := range history {
			if record.WatchedAt.Equal(user.LastFetchedTime) && record.ID == user.LastFetchedID {
				continue
			}

			if i == 0 && page == 1 {
				user.LastFetchedTime = record.WatchedAt
				user.LastFetchedID = record.ID

				err = users.save(user)
				if err != nil {
					log.Println("could not save user while importing", err)
					break
				}
			}

			fmt.Println(record)
			// TODO: import
		}

		if hasNext {
			page = page + 1
		} else {
			break
		}
	}

	processes.Lock()
	processes.DomainRunning[user.Domain] = false
	processes.Unlock()
}

func traktImportHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := mustUser(w, r)
	if user == nil {
		return
	}

	processes.Lock()
	running, ok := processes.DomainRunning[user.Domain]

	if running && ok {
		// Already being imported... just redirect!
		processes.Unlock()
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	processes.DomainRunning[user.Domain] = true
	processes.Unlock()

	go traktImport(user)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
