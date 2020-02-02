package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"
	"unsafe"

	"github.com/gorilla/sessions"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func getUser(w http.ResponseWriter, r *http.Request) (*user, *sessions.Session) {
	session, err := store.Get(r, "ownyourtrakt")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, nil
	}

	me, ok := session.Values["me"].(string)
	if !ok {
		return nil, session
	}

	u, err := users.get(me)
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

func logError(w http.ResponseWriter, r *http.Request, user *user, code int, err error) {
	if user == nil {
		log.Println(err)
	} else {
		log.Println(user.Domain, err)
	}

	renderer.HTML(w, code, "error", map[string]interface{}{
		"User":  user,
		"Error": err.Error(),
	})
}
