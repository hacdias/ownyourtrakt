package main

import (
	"log"
	"net/http"
	"net/url"
)

func buildAuthorizationURL(endpoint, me, redir, clientid, state, scope string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("me", me)
	q.Set("redirect_uri", redir)
	q.Set("client_id", clientid)
	q.Set("state", state)

	if scope != "" {
		q.Set("scope", scope)
		q.Set("response_type", "code")
	} else {
		q.Set("response_type", "id")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func authStartHandler(w http.ResponseWriter, r *http.Request) {
	me := r.URL.Query().Get("me")
	url, err := url.Parse(me)
	if err != nil || url.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	url.Path = "/"
	me = url.String()

	endpoints, err := discoverEndpoints(me)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	state := randString(10)

	session, err := store.Get(r, "session-name")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session.Values["auth_state"] = state
	session.Values["auth_me"] = me

	scope := "create update"

	authURL, err := buildAuthorizationURL(endpoints.IndieAuth, me, clientID+"/auth/callback", clientID, state, scope)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = session.Save(r, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	u, err := db.getUser(me)
	if err == nil {
		log.Printf("user %s already existed\n", me)
		u.Domain = me
		u.Endpoints = *endpoints
	} else {
		log.Println(err)
		log.Printf("user %s is new\n", me)
		u = &user{
			Domain:    me,
			Endpoints: *endpoints,
		}
	}

	err = db.saveUser(u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	originalState, ok := session.Values["auth_state"].(string)
	if !ok || originalState == "" {
		// TODO: redirect to start
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	me, ok := session.Values["auth_me"].(string)
	if !ok {
		// TODO: redirect to start
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		// TODO: redirect to start
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		// TODO: redirect to start
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if state != originalState {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	u, err := db.getUser(me)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if u.AccessToken == "" {
		token, err := getToken(me, code, clientID+"/auth/callback", clientID, state, u.Endpoints.Tokens)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		session.Values["me"] = token.Me
		u.AccessToken = token.AccessToken
	} else {
		session.Values["me"] = u.Domain
	}

	err = db.saveUser(u)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	delete(session.Values, "auth_state")
	delete(session.Values, "auth_me")

	err = session.Save(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func authLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session.Values = map[interface{}]interface{}{}
	err = session.Save(r, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
