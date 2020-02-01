package main

type user struct {
	Domain      string
	Endpoints   endpoints
	AccessToken string
	TraktOauth  oauthResponse
}
