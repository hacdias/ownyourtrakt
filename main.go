package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	app, err := newApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer app.close()

	server, err := newServer(app)
	if err != nil {
		log.Fatal(err)
	}
	defer server.close()

	go func() {
		err := server.start()
		if err != nil && http.ErrServerClosed != err {
			log.Fatal(err)
		}
	}()

	// Run scheduled imports in parallel and cancel before returning,
	// i.e., after getting the signal.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.scheduleImports(ctx)

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Print("stopping server")
	// .close() is deffered
}
