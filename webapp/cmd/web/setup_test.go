package main

import (
	"github.com/spacesedan/testing-course/webapp/pkg/repository/dbrepo"
	"os"
	"testing"
)

var app application

func TestMain(m *testing.M) {
	pathToTemplates = "./../../templates/"

	app.Session = getSession()
	app.DB = &dbrepo.TestDBRepo{}

	os.Exit(m.Run())
}
