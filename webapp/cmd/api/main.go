package main

import (
	"flag"
	"fmt"
	"github.com/spacesedan/testing-course/webapp/pkg/repository"
	"github.com/spacesedan/testing-course/webapp/pkg/repository/dbrepo"
	"log"
	"net/http"
)

const port int = 8090

type application struct {
	DSN       string
	DB        repository.DataBaseRepo
	Domain    string
	JWTSecret string
}

func main() {
	var app application

	flag.StringVar(&app.Domain, "domain", "example.com", "Domain for application, e.g. company.com")
	flag.StringVar(&app.DSN, "dsn", "host=localhost port=5431 user=postgres password=postgres dbname=users sslmode=disable timezone=UTC connect_timeout=5", "Postgres connection")
	flag.StringVar(&app.JWTSecret, "jwt-secret", "secret", "signing secret")
	flag.Parse()

	conn, err := app.connectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	app.DB = &dbrepo.PostgresDBRepo{DB: conn}

	log.Printf("Starting API on port %d\n", port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), app.routes())
	if err != nil {
		log.Fatal(err)
	}
}
