package main

import (
	"path/filepath"
	"time"

	rice "github.com/GeertJohan/go.rice"
)

type user struct {
	Domain            string
	Endpoints         endpoints
	AccessToken       string
	TraktOauth        oauthResponse
	NewestFetchedTime time.Time
	NewestFetchedID   int64
	OldestFetchedTime time.Time
	OldestFetchedID   int64
}

type assetsFS struct {
	box *rice.Box
}

func (a assetsFS) Walk(root string, walkFn filepath.WalkFunc) error {
	return a.box.Walk(root, walkFn)
}

func (a assetsFS) ReadFile(filename string) ([]byte, error) {
	return a.box.Bytes(filename)
}
