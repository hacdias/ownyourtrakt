package main

import (
	"encoding/gob"
	"log"
	"net/http"
	"os"
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
	port              = os.Getenv("PORT")
	store             = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	users             *usersDB
)

func init() {
	gob.Register(token{})
}

func main() {
	var err error
	users, err = newUsersDB("./database.db")
	if err != nil {
		panic(err)
	}
	defer users.close()

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

	r.HandleFunc("/trakt/import", traktImportHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler:      r,
		Addr:         "localhost:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
