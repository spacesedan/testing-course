package main

import (
	"github.com/spacesedan/testing-course/webapp/pkg/repository/dbrepo"
	"os"
	"testing"
)

var app application
var expiredToken string = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZG1pbiI6dHJ1ZSwiYXVkIjoiZXhhbXBsZS5jb20iLCJleHAiOjE2Njk3NDI4ODMsImlzcyI6ImV4YW1wbGUuY29tIiwibmFtZSI6IkpvaG4gRG9lIiwic3ViIjoiMSJ9.ES0_Om9MGHopyk_OsF5ucFb3P-rrwKnXKoHPE1BnaxA"

func TestMain(m *testing.M) {
	app.DB = &dbrepo.TestDBRepo{}
	app.Domain = "example.com"
	app.JWTSecret = "secret"
	os.Exit(m.Run())
}
