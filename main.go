// TODO: make this simple CLI with a few commands
// ownyourtrakt login - handles the trakt authentication and outputs required tokens in .env format
// ownyourtrakt sync - syncs... once per day...
package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/joho/godotenv/autoload"
	"github.com/unrolled/render"
)

var (
	clientID          = os.Getenv("BASE_URL")
	traktClientID     = os.Getenv("TRAKT_ID")
	traktClientSecret = os.Getenv("TRAKT_SECRET")
	host              = os.Getenv("HOST")
	port              = os.Getenv("PORT")
	sessionKey        = os.Getenv("SESSION_KEY")
	databasePath      = os.Getenv("DATABASE_PATH")

	store     *sessions.CookieStore
	users     *usersDB
	renderer  *render.Render
	processes = struct {
		sync.Mutex
		DomainRunning map[string]bool
	}{
		DomainRunning: map[string]bool{},
	}
)

func init() {
	gob.Register(token{})

	if traktClientID == "" {
		panic(errors.New("TRAKT_ID must be set"))
	}

	if traktClientSecret == "" {
		panic(errors.New("TRAKT_SECRET must be set"))
	}

	if sessionKey == "" {
		panic(errors.New("SESSION_KEY must be set"))
	}

	if databasePath == "" {
		panic(errors.New("DATABASE_PATH must be set"))
	}

	if host == "" {
		host = "127.0.0.1"
	}

	if port == "" {
		port = "8050"
	}

	if clientID == "" {
		clientID = "http://" + host + ":" + port
	}

	store = sessions.NewCookieStore([]byte(sessionKey))
}

func main() {
	var err error
	users, err = newUsersDB("./database.db")
	if err != nil {
		panic(err)
	}
	defer users.close()

	me, _ := users.get("https://dev.hacdias.com/")
	me.Endpoints.Micropub = "http://localhost:3030/micropub"
	users.save(me)

	// TODO: define micropub layout
	// store already sent requests with URL
	// store failed requests

	renderer = render.New(render.Options{
		Layout: "layout",
	})

	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user, _ := getUser(w, r)
		importing := false

		if user != nil {
			processes.Lock()
			if imp, ok := processes.DomainRunning[user.Domain]; ok {
				importing = imp
			}
			processes.Unlock()
		}

		renderer.HTML(w, http.StatusOK, "home", map[string]interface{}{
			"User":      user,
			"Importing": importing,
		})
	})

	r.HandleFunc("/auth/logout", authLogoutHandler)
	r.HandleFunc("/auth/start", authStartHandler)
	r.HandleFunc("/auth/callback", authCallbackHandler)

	r.HandleFunc("/trakt/start", traktStartHandler)
	r.HandleFunc("/trakt/callback", traktCallbackHandler)
	r.HandleFunc("/trakt/import/reset", traktResetHandler)
	r.HandleFunc("/trakt/import/newer", traktNewerHandler)
	r.HandleFunc("/trakt/import/older", traktOlderHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler:      r,
		Addr:         "localhost:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Listening on " + srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
