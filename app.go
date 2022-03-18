package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/hacdias/indieauth"
	"golang.org/x/oauth2"
)

type app struct {
	*config
	db        *database
	oauth2    *oauth2.Config
	indieauth *indieauth.Client
	importMu  sync.Mutex
	importing map[string]bool
}

func newApp(config *config) (*app, error) {
	a := &app{
		config:    config,
		importing: map[string]bool{},
		indieauth: indieauth.NewClient(config.BaseURL+"/", config.BaseURL+"/callback", nil),
		oauth2: &oauth2.Config{
			ClientID:     config.TraktClientID,
			ClientSecret: config.TraktClientSecret,
			RedirectURL:  config.BaseURL + "/trakt/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://trakt.tv/oauth/authorize",
				TokenURL: "https://trakt.tv/oauth/token",
			},
		},
	}

	db, err := newDatabase(config.Database)
	if err != nil {
		return nil, err
	}
	a.db = db

	return a, nil
}

func (a *app) close() error {
	return a.db.close()
}

func (a *app) getTraktAuthURL(state string) string {
	return a.oauth2.AuthCodeURL(state)
}

func (a *app) getTraktToken(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	return a.oauth2.Exchange(ctx, code)
}

func (a *app) getTraktClient(user *user) (*http.Client, error) {
	if user.TraktToken == nil {
		return nil, errors.New("user does not have trakt token")
	}

	return a.oauth2.Client(context.Background(), user.TraktToken), nil
}

func (a *app) getMicropubClient(user *user) (*http.Client, error) {
	oo := a.indieauth.GetOAuth2(&user.IndieAuthEndpoints)
	if user.IndieToken == nil {
		return nil, errors.New("user does not have indie token")
	}

	return oo.Client(context.Background(), user.IndieToken), nil
}

func (a *app) resetTrakt(user *user) error {
	user.OldestFetchedTime = time.Now()
	user.OldestFetchedID = 0
	user.NewestFetchedTime = user.OldestFetchedTime
	user.NewestFetchedID = 0

	return a.db.save(user)
}

func (a *app) importRequest(user *user, page int, startAt time.Time, endAt time.Time) (traktHistory, bool, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, false, err
	}

	httpClient, err := a.getTraktClient(user)
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("trakt-api-key", a.TraktClientID)
	req.Header.Set("trakt-api-version", "2")

	res, err := httpClient.Do(req)
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

func (a *app) importTrakt(user *user, older bool, fetchNext bool) {
	page := 1

	a.importMu.Lock()
	a.importing[user.ProfileURL] = true
	a.importMu.Unlock()

	for {
		var err error
		var history traktHistory
		var hasNext bool

		newestFetchedID := user.NewestFetchedID
		oldestFetchedID := user.OldestFetchedID

		if older {
			// Fetch older items
			history, hasNext, err = a.importRequest(user, page, time.Time{}, user.OldestFetchedTime)
		} else {
			// Fetch newer items
			history, hasNext, err = a.importRequest(user, page, user.NewestFetchedTime, time.Time{})
		}

		if err != nil {
			log.Printf("%s - could not fetch trakt: %v\n", user.ProfileURL, err)
			break
		}

		failed := false

		for _, record := range history {
			if record.ID == newestFetchedID || record.ID == oldestFetchedID {
				// Do not copy already copied ID.
				continue
			}

			err = a.sendMicropub(user, record)
			if err != nil {
				// Stop sending more if the micropub action is not successfull. Requires user
				// action or wait for next cron job.
				log.Printf("%s - could not send micropub: %v\n", user.ProfileURL, err)
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

			err = a.db.save(user)
			if err != nil {
				log.Fatalf("%s - could not save user: %v\n", user.ProfileURL, err)
				break
			}
		}

		if hasNext && fetchNext && !failed {
			page = page + 1
		} else {
			break
		}
	}

	a.importMu.Lock()
	defer a.importMu.Unlock()
	a.importing[user.ProfileURL] = false
}

func (a *app) sendMicropub(user *user, item traktHistoryItem) error {
	micro, err := traktToMicroformats(item)
	if err != nil {
		return err
	}

	data, err := json.Marshal(micro)
	if err != nil {
		return err
	}

	httpClient, err := a.getMicropubClient(user)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", user.MicropubEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
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

	return errors.New(user.ProfileURL +
		": status from micropub endpoint was " +
		strconv.Itoa(resp.StatusCode) +
		" body: " +
		string(bodyBytes),
	)
}

func (a *app) importEveryone() {
	log.Println("Running cron import")
	users, err := a.db.getAll()
	if err != nil {
		log.Printf("error while getting users: %v\n", err)
	}

	for _, user := range users {
		if user.IndieToken == nil || user.TraktToken == nil {
			continue
		}

		a.importTrakt(user, false, false)
	}

	log.Println("Finished running cron import")
}

func (a *app) scheduleImports(ctx context.Context) {
	a.importEveryone()

	t := time.NewTicker(time.Minute * 30)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			a.importEveryone()
		case <-ctx.Done():
			return
		}
	}
}
