package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/hacdias/ownyourtrakt/trakt"
	_ "github.com/joho/godotenv/autoload"
	"github.com/unrolled/render"
)

var (
	clientID          = os.Getenv("BASE_URL")
	traktClientID     = os.Getenv("TRAKT_ID")
	traktClientSecret = os.Getenv("TRAKT_SECRET")
	port              = os.Getenv("PORT")
	store             = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	db                *database
)

func init() {
	gob.Register(token{})
}

func main() {
	var err error
	db, err = newDatabase("./database.db")
	if err != nil {
		panic(err)
	}
	defer db.close()

	renderer := render.New(render.Options{
		Layout: "layout",
	})

	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user, _ := getUser(w, r)

		renderer.HTML(w, http.StatusOK, "example", map[string]interface{}{
			"User": user,
		})
	})

	r.HandleFunc("/auth/logout", authLogoutHandler)
	r.HandleFunc("/auth/start", authStartHandler)
	r.HandleFunc("/auth/callback", authCallbackHandler)
	r.HandleFunc("/trakt/start", traktStartHandler)
	r.HandleFunc("/trakt/callback", traktCallbackHandler)

	r.HandleFunc("/import", importHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler:      r,
		Addr:         "localhost:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func importHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := mustUser(w, r)
	if user == nil {
		return
	}

	/*
		X-Pagination-Page	Current page.
		X-Pagination-Limit	Items per page.
		X-Pagination-Page-Count	Total number of pages.
		X-Pagination-Item-Count	Total number of items.
	*/

	res, err := http.NewRequest("GET", "https://api.trakt.tv/sync/history", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header.Set("Content-Type", "application/json")
	res.Header.Set("Accept", "application/json")
	res.Header.Set("trakt-api-key", traktClientID)
	res.Header.Set("trakt-api-version", "2")
	res.Header.Set("Authorization", "Bearer "+user.TraktOauth.AccessToken)

	resp, err := http.DefaultClient.Do(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var history trakt.History

	err = json.NewDecoder(resp.Body).Decode(&history)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Println(history)
	io.Copy(w, resp.Body)
}
