package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
)

func getUser(w http.ResponseWriter, r *http.Request) (*user, *sessions.Session) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, nil
	}

	me, ok := session.Values["me"].(string)
	if !ok {
		return nil, session
	}

	u, err := db.getUser(me)
	if err != nil {
		return nil, session
	}

	return u, session
}

func mustUser(w http.ResponseWriter, r *http.Request) (*user, *sessions.Session) {
	user, session := getUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, nil
	}

	return user, session
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

	err = db.saveUser(user)
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
