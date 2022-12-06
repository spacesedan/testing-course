package main

import (
	"github.com/alexedwards/scs/v2"
	"net/http"
	"time"
)

func getSession() *scs.SessionManager {
	session := scs.New()
	session.Lifetime = time.Hour * 24
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session

}
