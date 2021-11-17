package main

import (
	"time"

	"golang.org/x/oauth2"
)

type user struct {
	ProfileURL        string
	Endpoints         endpoints
	IndieToken        *oauth2.Token
	TraktToken        *oauth2.Token
	NewestFetchedTime time.Time
	NewestFetchedID   int64
	OldestFetchedTime time.Time
	OldestFetchedID   int64
}
