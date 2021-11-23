package main

import (
	"time"

	"github.com/hacdias/indieauth"
	"golang.org/x/oauth2"
)

type user struct {
	ProfileURL         string
	IndieAuthEndpoints indieauth.Endpoints
	MicropubEndpoint   string
	IndieToken         *oauth2.Token
	TraktToken         *oauth2.Token
	NewestFetchedTime  time.Time
	NewestFetchedID    int64
	OldestFetchedTime  time.Time
	OldestFetchedID    int64
}
