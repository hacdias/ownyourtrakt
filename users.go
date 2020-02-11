package main

import "time"

type user struct {
	Domain            string
	Endpoints         endpoints
	AccessToken       string
	TraktOauth        oauthResponse
	NewestFetchedTime time.Time
	NewestFetchedID   int64
	OldestFetchedTime time.Time
	OldestFetchedID   int64
	FailedIDs         []int64
}
