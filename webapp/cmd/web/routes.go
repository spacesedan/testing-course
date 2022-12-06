package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	// register middle ware
	mux.Use(middleware.Recoverer)
	mux.Use(app.addIPtoContext)
	mux.Use(app.Session.LoadAndSave)
	mux.Use(middleware.Heartbeat("/ping"))
	mux.Use(cors.Handler(cors.Options{}))

	// register routes
	mux.Get("/", app.Home)
	mux.Post("/login", app.Login)

	// protect the "/user/profile" page with auth middleware
	mux.Route("/user", func(mux chi.Router) {
		mux.Use(app.auth)
		mux.Get("/profile", app.Profile)
		mux.Post("/upload-profile-pic", app.UploadProfilePic)
	})

	// static assets
	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}
