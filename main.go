//go:generate go install github.com/GeertJohan/go.rice/rice
//go:generate rice embed-go
package main

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	rice "github.com/GeertJohan/go.rice"
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

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// TODO: renew token when needed
	// TODO: lock user savings

	renderer = render.New(render.Options{
		Layout:     "layout",
		Directory:  "/",
		FileSystem: assetsFS{rice.MustFindBox("templates")},
	})

	r := mux.NewRouter()

	fs := http.FileServer(rice.MustFindBox("static").HTTPBox())
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
		Addr:         "127.0.0.1:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Run server in paralell.
	go func() {
		log.Printf("Listening on http://%s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Run scheduled imports in parallel and cancel before returning,
	// i.e., after getting the signal.
	impCtx, impCancel := context.WithCancel(context.Background())
	defer impCancel()
	go scheduleImports(impCtx)

	<-stop

	log.Printf("Shutting down the server...\n")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		log.Printf("error while shutting down server: %v\n", err)
	}

	log.Printf("Server gracefully stopped\n")
}
