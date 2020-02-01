package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

type oauthRequest struct {
	Code         string `json:"code,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"` // refresh_token, authorization_code
}

type oauthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int    `json:"created_at"`
}

func authorize(endpoint string, oauthReq *oauthRequest) (*oauthResponse, error) {
	js, err := json.Marshal(oauthReq)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	res := &oauthResponse{}
	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func traktStartHandler(w http.ResponseWriter, r *http.Request) {
	user, session := mustUser(w, r)
	if user == nil {
		return
	}

	u, err := url.Parse("https://trakt.tv/oauth/authorize")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	state := randString(10)

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", traktClientID)
	q.Set("redirect_uri", clientID+"/trakt/callback")
	q.Set("state", state)
	u.RawQuery = q.Encode()

	session.Values["trakt_state"] = state

	err = session.Save(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func traktCallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, session := mustUser(w, r)
	if user == nil {
		return
	}

	originalState, ok := session.Values["trakt_state"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	state := r.URL.Query().Get("state")
	if state != originalState {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tk, err := authorize("https://api.trakt.tv/oauth/token", &oauthRequest{
		Code:         code,
		ClientID:     traktClientID,
		ClientSecret: traktClientSecret,
		RedirectURI:  clientID + "/trakt/callback",
		GrantType:    "authorization_code",
	})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user.TraktOauth = *tk

	err = users.save(user)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	delete(session.Values, "trakt_state")

	err = session.Save(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
