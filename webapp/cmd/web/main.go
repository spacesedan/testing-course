package main

import (
	"encoding/gob"
	"flag"
	"github.com/alexedwards/scs/v2"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"github.com/spacesedan/testing-course/webapp/pkg/repository"
	"github.com/spacesedan/testing-course/webapp/pkg/repository/dbrepo"
	"log"
	"net/http"
)

const webPort string = "8080"

type application struct {
	DSN     string
	Session *scs.SessionManager
	DB      repository.DataBaseRepo
}

func main() {
	gob.Register(data.User{})
	// set up an application
	app := application{
		Session: getSession(),
	}

	flag.StringVar(
		&app.DSN,
		"dsn",
		"host=localhost port=5431 user=postgres password=postgres dbname=users sslmode=disable timezone=UTC connect_timeout=5",
		"Postgres connection",
	)
	flag.Parse()

	conn, err := app.connectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	app.DB = &dbrepo.PostgresDBRepo{DB: conn}

	// print out a message
	log.Println("Starting server on port: ", webPort)

	// start the server
	err = http.ListenAndServe(":"+webPort, app.routes())
	if err != nil {
		log.Fatal(err)
	}

}
