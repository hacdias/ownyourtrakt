package main

import (
	"embed"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/hacdias/indieauth"
	"github.com/unrolled/render"
)

const sessionKey = "ownyourtrakt"

//go:embed static/*
var static embed.FS

//go:embed templates/*
var templates embed.FS

type server struct {
	*app
	srv    *http.Server
	store  *sessions.CookieStore
	render *render.Render
}

func newServer(app *app) (*server, error) {
	s := &server{
		store: sessions.NewCookieStore([]byte(app.SessionKey)),
		app:   app,
		render: render.New(render.Options{
			Layout:     "layout",
			Directory:  "templates",
			FileSystem: render.FS(templates),
		}),
	}

	return s, nil
}

func (s *server) start() error {
	r := chi.NewMux()

	r.Handle("/static*", http.FileServer(http.FS(static)))

	r.Get("/", s.rootGet)
	r.Get("/login", s.loginGet)
	r.Get("/logout", s.logoutGet)
	r.Get("/callback", s.callbackGet)

	r.Get("/trakt/start", s.traktStartGet)
	r.Get("/trakt/callback", s.traktCallbackGet)

	r.Get("/trakt/reset", s.traktResetGet)
	r.Post("/trakt/reset", s.traktResetPost)

	r.Get("/trakt/newer", s.traktNewerGet)
	r.Get("/trakt/older", s.traktOlderGet)

	addr := ":" + strconv.Itoa(s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.srv = &http.Server{
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("listening on %s", ln.Addr().String())
	log.Printf("public address is %s", s.BaseURL)
	return s.srv.Serve(ln)
}

type rootData struct {
	Importing bool
	User      *user
}

func (s *server) rootGet(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.getUser(w, r)
	if !ok {
		return
	}

	importing := false

	if user != nil {
		s.importMu.Lock()
		importing = s.importing[user.ProfileURL]
		s.importMu.Unlock()
	}

	err := s.render.HTML(w, http.StatusOK, "home", &rootData{
		User:      user,
		Importing: importing,
	})
	if err != nil {
		log.Print(err)
	}
}

func (s *server) loginGet(w http.ResponseWriter, r *http.Request) {
	me := r.URL.Query().Get("me")
	url, err := url.Parse(me)
	if err != nil || url.Host == "" {
		s.error(w, r, nil, http.StatusBadRequest, errors.New("invalid url provided"))
		return
	}

	url.Path = "/"
	me = url.String()

	authInfo, redirect, err := s.indieauth.Authenticate(me, "create")
	if err != nil {
		s.error(w, r, nil, http.StatusBadRequest, err)
		return
	}

	micropub, err := s.indieauth.DiscoverEndpoint(me, "micropub")
	if err != nil {
		s.error(w, r, nil, http.StatusBadRequest, err)
		return
	}

	_, session, ok := s.getUser(w, r)
	if !ok {
		return
	}

	session.Values["auth_me"] = authInfo.Me
	session.Values["auth_state"] = authInfo.State
	session.Values["auth_code_verifier"] = authInfo.CodeVerifier

	err = session.Save(r, w)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	u, err := s.db.get(me)
	if err == nil {
		log.Printf("user %s already existed\n", me)
		u.ProfileURL = me
		u.MicropubEndpoint = micropub
		u.IndieAuthEndpoints = authInfo.Endpoints
	} else if s.DisableSignups {
		s.error(w, r, nil, http.StatusForbidden, errors.New("new users are disabled"))
		return
	} else {
		log.Printf("user %s is new or error fetching from the DB: %s\n", me, err)
		u = &user{
			ProfileURL:         me,
			MicropubEndpoint:   micropub,
			IndieAuthEndpoints: authInfo.Endpoints,
			OldestFetchedTime:  time.Now(),
		}
		u.NewestFetchedTime = u.OldestFetchedTime
	}

	err = s.db.save(u)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *server) callbackGet(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.getUser(w, r)
	if !ok {
		return
	}

	me, ok := session.Values["auth_me"].(string)
	if !ok {
		// log in session was not started, go home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	state, ok := session.Values["auth_state"].(string)
	if !ok {
		// log in session was not started, go home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	codeVerifier, ok := session.Values["auth_code_verifier"].(string)
	if !ok {
		// log in session was not started, go home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, err := s.db.get(me)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	authInfo := &indieauth.AuthInfo{
		Me:           me,
		State:        state,
		CodeVerifier: codeVerifier,
		Endpoints:    user.IndieAuthEndpoints,
	}

	code, err := s.indieauth.ValidateCallback(authInfo, r)
	if err != nil {
		s.error(w, r, nil, http.StatusBadRequest, err)
		return
	}

	tok, _, err := s.indieauth.GetToken(authInfo, code)
	if err != nil {
		s.error(w, r, nil, http.StatusBadRequest, err)
		return
	}

	user.IndieToken = tok
	session.Values["me"] = user.ProfileURL

	err = s.db.save(user)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	delete(session.Values, "auth_me")
	delete(session.Values, "auth_state")
	delete(session.Values, "auth_code_verifier")

	err = session.Save(r, w)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) logoutGet(w http.ResponseWriter, r *http.Request) {
	session, err := s.store.Get(r, sessionKey)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	session.Values = map[interface{}]interface{}{}
	err = session.Save(r, w)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) traktStartGet(w http.ResponseWriter, r *http.Request) {
	user, session := s.mustUser(w, r)
	if user == nil {
		return
	}

	state := randString(10)
	url := s.getTraktAuthURL(state)
	session.Values["trakt_state"] = state

	err := session.Save(r, w)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (s *server) traktCallbackGet(w http.ResponseWriter, r *http.Request) {
	user, session := s.mustUser(w, r)
	if user == nil {
		return
	}

	originalState, ok := session.Values["trakt_state"].(string)
	if !ok {
		// redirect to login because trakt session was not started
		http.Redirect(w, r, "/trakt/start", http.StatusSeeOther)
		return
	}

	state := r.URL.Query().Get("state")
	if state != originalState {
		s.error(w, r, user, http.StatusBadRequest, errors.New("state was invalid"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		s.error(w, r, user, http.StatusBadRequest, errors.New("code was empty"))
		return
	}

	tok, err := s.getTraktToken(code)
	if err != nil {
		s.error(w, r, user, http.StatusUnauthorized, err)
		return
	}

	user.TraktToken = tok

	err = s.db.save(user)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	delete(session.Values, "trakt_state")

	err = session.Save(r, w)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) traktResetGet(w http.ResponseWriter, r *http.Request) {
	user, _ := s.mustUser(w, r)
	if user == nil {
		return
	}

	err := s.render.HTML(w, http.StatusOK, "reset", &rootData{User: user})
	if err != nil {
		log.Print(err)
	}
}

func (s *server) traktResetPost(w http.ResponseWriter, r *http.Request) {
	user, _ := s.mustUser(w, r)
	if user == nil {
		return
	}

	err := s.resetTrakt(user)
	if err != nil {
		s.error(w, r, user, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) error(w http.ResponseWriter, r *http.Request, user *user, code int, err error) {
	if user == nil {
		log.Println(err)
	} else {
		log.Println(user.ProfileURL, err)
	}

	err = s.render.HTML(w, code, "error", map[string]interface{}{
		"User":  user,
		"Error": err.Error(),
	})
	if err != nil {
		log.Print(err)
	}
}

func (s *server) close() error {
	return s.srv.Close()
}

func (s *server) getUser(w http.ResponseWriter, r *http.Request) (*user, *sessions.Session, bool) {
	session, err := s.store.Get(r, sessionKey)
	if err != nil {
		s.error(w, r, nil, http.StatusInternalServerError, err)
		return nil, nil, false
	}

	me, ok := session.Values["me"].(string)
	if !ok {
		return nil, session, true
	}

	u, err := s.db.get(me)
	if err != nil {
		return nil, session, true
	}

	return u, session, true
}

func (s *server) mustUser(w http.ResponseWriter, r *http.Request) (*user, *sessions.Session) {
	user, session, ok := s.getUser(w, r)
	if !ok {
		return nil, nil
	}

	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil, nil
	}

	return user, session
}

func (s *server) checkTrakt(w http.ResponseWriter, r *http.Request) (user *user, ok bool) {
	user, _ = s.mustUser(w, r)
	if user == nil {
		return nil, false
	}

	if user.TraktToken == nil {
		http.Redirect(w, r, "/trakt/start", http.StatusSeeOther)
		return nil, false
	}

	s.importMu.Lock()
	defer s.importMu.Unlock()
	running := s.importing[user.ProfileURL]

	if running {
		// Already being imported... just redirect!
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil, false
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return user, true
}

func (s *server) traktNewerGet(w http.ResponseWriter, r *http.Request) {
	user, ok := s.checkTrakt(w, r)
	if !ok {
		return
	}

	go s.importTrakt(user, false, false)
}

func (s *server) traktOlderGet(w http.ResponseWriter, r *http.Request) {
	user, ok := s.checkTrakt(w, r)
	if !ok {
		return
	}

	go s.importTrakt(user, true, false)
}
