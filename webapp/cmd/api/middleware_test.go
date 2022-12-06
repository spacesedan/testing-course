package main

import (
	"fmt"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplication_enableCORS(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	tests := []struct {
		name         string
		method       string
		expectHeader bool
	}{
		{"preflight", http.MethodOptions, true},
		{"get", http.MethodGet, false},
		{"post", http.MethodPost, false},
		{"put", http.MethodPut, false},
		{"patch", http.MethodPatch, false},
		{"delete", http.MethodDelete, false},
	}

	for _, e := range tests {
		handlerToTest := app.enableCORS(nextHandler)

		req := httptest.NewRequest(e.method, "http://testing", nil)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		if e.expectHeader && rr.Header().Get("Access-Control-Allow-Credentials") == "" {
			t.Errorf("%s: expected header but did not find it", e.name)
		}

		if !e.expectHeader && rr.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("%s: expected no header but got one", e.name)
		}
	}
}

func TestApplication_authRequired(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	testUser := data.User{ID: 1, FirstName: "Admin", LastName: "User", Email: "admin@example.com"}

	tokens, _ := app.generateTokenPair(&testUser)

	tests := []struct {
		name             string
		token            string
		expectAuthorized bool
		setHeader        bool
	}{
		{"valid", fmt.Sprintf("Bearer %s", tokens.Token), true, true},
		{"no token", "", false, false},
		{"invalid token", fmt.Sprintf("Bearer %s", expiredToken), false, true},
	}
	for _, e := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if e.setHeader {
			req.Header.Set("Authorization", e.token)
		}
		rr := httptest.NewRecorder()
		handlerToTest := app.authRequired(nextHandler)
		handlerToTest.ServeHTTP(rr, req)

		if e.expectAuthorized && rr.Code == http.StatusUnauthorized {
			t.Errorf("%s: got code 401, and should not have", e.name)
		}

		if !e.expectAuthorized && rr.Code != http.StatusUnauthorized {
			t.Errorf("%s: did not get 401, and should have", e.name)
		}
	}
}
