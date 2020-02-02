package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
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
		logError(w, r, nil, http.StatusBadRequest, errors.New("invalid domain provided"))
		return
	}

	url.Path = "/"
	me = url.String()

	endpoints, err := discoverEndpoints(me)
	if err != nil {
		logError(w, r, nil, http.StatusBadRequest, err)
		return
	}

	state := randString(10)

	session, err := store.Get(r, "ownyourtrakt")
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	session.Values["auth_state"] = state
	session.Values["auth_me"] = me

	scope := "create update"

	authURL, err := buildAuthorizationURL(endpoints.IndieAuth, me, clientID+"/auth/callback", clientID, state, scope)
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	err = session.Save(r, w)
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	u, err := users.get(me)
	if err == nil {
		log.Printf("user %s already existed\n", me)
		u.Domain = me
		u.Endpoints = *endpoints
	} else {
		log.Printf("user %s is new or error fetching from the DB: %s\n", me, err)
		u = &user{
			Domain:    me,
			Endpoints: *endpoints,
		}
	}

	err = users.save(u)
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "ownyourtrakt")
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	originalState, ok := session.Values["auth_state"].(string)
	if !ok || originalState == "" {
		// log in session was not started, restart
		http.Redirect(w, r, "/auth/start", http.StatusTemporaryRedirect)
		return
	}

	me, ok := session.Values["auth_me"].(string)
	if !ok {
		// log in session was not started, restart
		http.Redirect(w, r, "/auth/start", http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		logError(w, r, nil, http.StatusBadRequest, errors.New("code was empty"))
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		logError(w, r, nil, http.StatusBadRequest, errors.New("state was empty"))
		return
	}

	if state != originalState {
		logError(w, r, nil, http.StatusBadRequest, errors.New("state was invalid"))
		return
	}

	u, err := users.get(me)
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	if u.AccessToken == "" {
		token, err := getToken(me, code, clientID+"/auth/callback", clientID, state, u.Endpoints.Tokens)
		if err != nil {
			logError(w, r, u, http.StatusInternalServerError, err)
			return
		}

		session.Values["me"] = token.Me
		u.AccessToken = token.AccessToken
	} else {
		session.Values["me"] = u.Domain
	}

	err = users.save(u)
	if err != nil {
		logError(w, r, u, http.StatusInternalServerError, err)
		return
	}

	delete(session.Values, "auth_state")
	delete(session.Values, "auth_me")

	err = session.Save(r, w)
	if err != nil {
		logError(w, r, u, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func authLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "ownyourtrakt")
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	session.Values = map[interface{}]interface{}{}
	err = session.Save(r, w)
	if err != nil {
		logError(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

type endpoints struct {
	Micropub  string
	IndieAuth string
	Tokens    string
}

func link(doc *html.Node, which string) (string, error) {
	var href string

	var crawler func(node *html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "link" {
			for _, m := range node.Attr {
				if m.Key == "rel" && m.Val == which {
					for _, m := range node.Attr {
						if m.Key == "href" {
							href = m.Val
							return
						}
					}
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}

	crawler(doc)

	if href == "" {
		return "", errors.New("could not find link tag")
	}

	return href, nil
}

func discoverEndpoints(domain string) (*endpoints, error) {
	resp, err := http.Get(domain)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("code is not 200")
	}

	node, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	micropub, err := link(node, "micropub")
	if err != nil {
		return nil, err
	}

	indieauth, err := link(node, "authorization_endpoint")
	if err != nil {
		return nil, err
	}

	tokens, err := link(node, "token_endpoint")
	if err != nil {
		return nil, err
	}

	return &endpoints{
		Micropub:  micropub,
		IndieAuth: indieauth,
		Tokens:    tokens,
	}, nil
}

type token struct {
	AccessToken string `json:"access_token"`
	Me          string `json:"me"`
	Scope       string `json:"scope"`
}

func getToken(me, code, redirectURI, clientID, codeVerifier, endpoint string) (*token, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("me", me)
	q.Set("grant_type", "authorization_code")
	q.Set("code", code)
	q.Set("redirect_uri", redirectURI)
	q.Set("client_id", clientID)
	q.Set("code_verifier", codeVerifier)

	u.RawQuery = q.Encode()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res token
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	if res.AccessToken == "" {
		return nil, errors.New("no access token was provided")
	}

	return &res, nil
}
