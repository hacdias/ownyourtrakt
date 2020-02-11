package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func importRequest(user *user, page int, startAt time.Time, endAt time.Time) (traktHistory, bool, error) {
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

func traktToMicroformats(item traktHistoryItem) (interface{}, error) {
	watch := map[string]interface{}{}
	watch["trakt-id"] = []int64{item.ID}

	if item.Type == "episode" {
		show := map[string]interface{}{}

		show["title"] = []string{item.Show.Title}
		show["year"] = []int{item.Show.Year}
		show["url"] = []string{"https://trakt.tv/shows/" + item.Show.IDs.Slug}
		show["ids"] = item.Show.IDs

		watch["title"] = []string{item.Episode.Title}
		watch["season"] = []int{item.Episode.Season}
		watch["episode"] = []int{item.Episode.Number}
		watch["url"] = []string{
			"https://trakt.tv/shows/" +
				item.Show.IDs.Slug +
				"/seasons/" +
				strconv.Itoa(item.Episode.Season) +
				"/episodes/" +
				strconv.Itoa(item.Episode.Number),
		}
		watch["ids"] = item.Episode.IDs
		watch["show"] = []interface{}{
			map[string]interface{}{
				"type":       []string{"h-card"},
				"properties": show,
			},
		}
	} else if item.Type == "movie" {
		watch["title"] = []string{item.Movie.Title}
		watch["year"] = []int{item.Movie.Year}
		watch["url"] = []string{"https://trakt.tv/movies/" + item.Movie.IDs.Slug}
		watch["ids"] = item.Movie.IDs
	} else {
		return nil, errors.New("invalid type " + item.Type)
	}

	mf2 := map[string]interface{}{
		"type": []string{"h-entry"},
		"properties": map[string]interface{}{
			"published": []string{item.WatchedAt.Format(time.RFC3339)},
			"watch-of": []interface{}{
				map[string]interface{}{
					"type":       []string{"h-card"},
					"properties": watch,
				},
			},
		},
	}

	return mf2, nil
}

func sendMicropub(user *user, item traktHistoryItem) error {
	micro, err := traktToMicroformats(item)
	if err != nil {
		return err
	}

	data, err := json.Marshal(micro)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", user.Endpoints.Micropub, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusCreated {
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return errors.New(user.Domain +
		": status from micropub endpoint was " +
		strconv.Itoa(resp.StatusCode) +
		" body: " +
		string(bodyBytes),
	)
}

func traktImport(user *user, older bool, fetchNext bool) {
	page := 1

	for {
		var err error
		var history traktHistory
		var hasNext bool

		if older {
			// Fetch older items
			history, hasNext, err = importRequest(user, page, time.Time{}, user.OldestFetchedTime)
		} else {
			// Fetch newer items
			history, hasNext, err = importRequest(user, page, user.NewestFetchedTime, time.Time{})
		}

		if err != nil {
			log.Printf("%s - could not fetch trakt: %v\n", user.Domain, err)
			return
		}

		failed := false

		for _, record := range history {
			if record.WatchedAt.Equal(user.NewestFetchedTime) && record.ID == user.NewestFetchedID {
				continue
			}

			if record.WatchedAt.Equal(user.OldestFetchedTime) && record.ID == user.OldestFetchedID {
				continue
			}

			err = sendMicropub(user, record)
			if err != nil {
				// Stop sending more if the micropub action is not successfull. Requires user
				// action or wait for next cron job.
				log.Printf("%s - could not send micropub: %v\n", user.Domain, err)
				failed = true
				break
			}

			if user.NewestFetchedTime.IsZero() || record.WatchedAt.After(user.NewestFetchedTime) {
				user.NewestFetchedTime = record.WatchedAt
				user.NewestFetchedID = record.ID
			}

			if user.OldestFetchedTime.IsZero() || record.WatchedAt.Before(user.OldestFetchedTime) {
				user.OldestFetchedTime = record.WatchedAt
				user.OldestFetchedID = record.ID
			}

			err = users.save(user)
			if err != nil {
				log.Fatalf("%s - could not save user: %v\n", user.Domain, err)
				break
			}
		}

		if hasNext && fetchNext && !failed {
			page = page + 1
		} else {
			break
		}
	}

	processes.Lock()
	processes.DomainRunning[user.Domain] = false
	processes.Unlock()
}

func isTraktOk(w http.ResponseWriter, r *http.Request) (user *user, ok bool) {
	user, _ = mustUser(w, r)
	if user == nil {
		return nil, false
	}

	if user.TraktOauth.AccessToken == "" {
		http.Redirect(w, r, "/trakt/start", http.StatusTemporaryRedirect)
		return nil, false
	}

	processes.Lock()
	running, ok := processes.DomainRunning[user.Domain]

	if running && ok {
		// Already being imported... just redirect!
		processes.Unlock()
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, false
	}

	processes.DomainRunning[user.Domain] = true
	processes.Unlock()

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return user, true
}

func traktNewerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := isTraktOk(w, r)
	if !ok {
		return
	}

	go traktImport(user, false, false)
}

func traktOlderHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := isTraktOk(w, r)
	if !ok {
		return
	}

	go traktImport(user, true, false)
}

func traktResetHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := mustUser(w, r)
	if user == nil {
		return
	}

	user.OldestFetchedTime = time.Now()
	user.OldestFetchedID = 0
	user.NewestFetchedTime = user.OldestFetchedTime
	user.NewestFetchedID = 0

	err := users.save(user)
	if err != nil {
		logError(w, r, user, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
