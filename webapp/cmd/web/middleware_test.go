package main

import (
	"context"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_application_addIPToContext(t *testing.T) {
	tests := []struct {
		HeaderName   string
		HeaderValue  string
		addr         string
		emptyAddress bool
	}{
		{"", "", "", false},
		{"", "", "", true},
		{"X-Forwarded-For", "192.3.2.1", "", false},
		{"", "", "hello:world", false},
	}

	// create a dummy handler that we'll use to check the context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure that the value exists in the context
		val := r.Context().Value(contextUserKey)
		if val == nil {
			t.Error(contextUserKey, "not present")
		}
		// make sure we get a string back
		ip, ok := val.(string)

		if !ok {
			t.Error("no string")
		}
		t.Log(ip)
	})

	for _, e := range tests {
		// create the handler to test
		handlerToTest := app.addIPtoContext(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "http://testing", nil)

		if e.emptyAddress {
			req.RemoteAddr = ""
		}

		if len(e.HeaderName) > 0 {
			req.Header.Add(e.HeaderName, e.HeaderValue)
		}

		if (len(e.addr)) > 0 {
			req.RemoteAddr = e.addr
		}

		handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func Test_application_ipFromContext(t *testing.T) {

	// get a context
	ctx := context.Background()

	// put something in the context
	ctx = context.WithValue(ctx, contextUserKey, "hello:world")

	// call the function
	ip := app.ipFromContext(ctx)

	// preform the test
	if !strings.EqualFold("hello:world", ip) {
		t.Errorf("expected hello:world, but got %s", ip)
	}
}

func TestApplication_auth(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})

	tests := []struct {
		name   string
		isAuth bool
	}{
		{"authenticated", true},
		{"not authenticated", false},
	}

	for _, e := range tests {
		handlerToTest := app.auth(nextHandler)
		req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
		req = addContextAndSessionToRequest(req, app)
		if e.isAuth {
			app.Session.Put(req.Context(), "user", data.User{ID: 1})
		}
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if e.isAuth && rr.Code != http.StatusOK {
			t.Errorf("%s: expected statuse code of 200 but got %d", e.name, rr.Code)
		}

		if !e.isAuth && rr.Code != http.StatusTemporaryRedirect {
			t.Errorf("%s: expected status code 307, but got %d", e.name, rr.Code)
		}
	}
}
