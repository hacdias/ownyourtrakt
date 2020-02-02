package main

import "time"

type user struct {
	Domain          string
	Endpoints       endpoints
	AccessToken     string
	TraktOauth      oauthResponse
	LastFetchedTime time.Time
	LastFetchedID   int
}
