package main

import (
	"time"

	"github.com/hacdias/indieauth/v2"
	"golang.org/x/oauth2"
)

type user struct {
	ProfileURL        string
	IndieAuthMetadata indieauth.Metadata
	MicropubEndpoint  string
	IndieToken        *oauth2.Token
	TraktToken        *oauth2.Token
	NewestFetchedTime time.Time
	NewestFetchedID   int64
	OldestFetchedTime time.Time
	OldestFetchedID   int64
}
