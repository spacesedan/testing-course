package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestApplication_authenticate(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        string
		expectedStatusCode int
	}{
		{"valid user", `{"email": "admin@example.com", "password": "secret"}`, http.StatusOK},
		{"empty email", `{"email": "", "password": "secret"}`, http.StatusUnauthorized},
		{"empty password", `{"email": "admin@example.com", "password": ""}`, http.StatusUnauthorized},
		{"not JSON", `I'm not JSON'`, http.StatusUnauthorized},
		{"invalid user", `{"email": "WRONG@USER.com", "password": "secret"}`, http.StatusUnauthorized},
		{"wrong password", `{"email": "admin@example.com", "password": "WRONG_PASSWORD"}`, http.StatusUnauthorized},
		{"missing body", `{}`, http.StatusUnauthorized},
	}

	for _, e := range tests {
		var reader io.Reader
		reader = strings.NewReader(e.requestBody)
		req := httptest.NewRequest(http.MethodPost, "/auth", reader)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.authenticate)
		handler.ServeHTTP(rr, req)

		if e.expectedStatusCode != rr.Code {
			t.Errorf("%s: returned wrong status code; expected %d but got %d", e.name, e.expectedStatusCode, rr.Code)
		}
	}
}

func TestApplication_refresh(t *testing.T) {
	tests := []struct {
		name               string
		token              string
		expectedStatusCode int
		resetRefreshTime   bool
	}{
		{"valid", "", http.StatusOK, true},
		{"valid but not yet ready to expire", "", http.StatusTooEarly, false},
		{"expired token", expiredToken, http.StatusBadRequest, false},
	}

	testUser := data.User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
		Email:     "admin@example.com",
	}

	oldRefreshTime := refreshTokenExpiry

	for _, e := range tests {
		var tkn string
		if e.token == "" {
			if e.resetRefreshTime {
				refreshTokenExpiry = time.Second * 1
			}
			tokens, _ := app.generateTokenPair(&testUser)
			tkn = tokens.RefreshToken
		} else {
			tkn = e.token
		}

		postedData := url.Values{
			"refresh_token": {tkn},
		}

		req := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(postedData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.refresh)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s: expected status of %d, but got %d", e.name, e.expectedStatusCode, rr.Code)
		}

		refreshTokenExpiry = oldRefreshTime
	}

}

func TestApplication_userHandlers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		json           string
		paramID        string
		handler        http.HandlerFunc
		expectedStatus int
	}{
		{"allUsers", http.MethodGet, "", "", app.allUsers, http.StatusOK},
		{"deleteUser", http.MethodDelete, "", "1", app.deleteUser, http.StatusNoContent},
		{"deleteUser invalid param", http.MethodDelete, "", "one", app.deleteUser, http.StatusBadRequest},
		{"getUser valid", http.MethodGet, "", "1", app.getUser, http.StatusOK},
		{"getUser invalid", http.MethodGet, "", "2", app.getUser, http.StatusBadRequest},
		{"getUser invalid param", http.MethodGet, "", "one", app.getUser, http.StatusBadRequest},
		{
			"updateUser valid",
			http.MethodPatch,
			`{"id": 1, "first_name": "Admin", "last_name": "User", "email": "admin@example.com"}`,
			"",
			app.updateUser,
			http.StatusNoContent,
		},
		{
			"updateUser invalid",
			http.MethodPatch,
			`{"id": 2, "first_name": "INVALID", "last_name": "USER", "email": "INVALID@USER.com"}`,
			"",
			app.updateUser,
			http.StatusBadRequest,
		},
		{
			"updateUser invalid json",
			http.MethodPatch,
			`{"id": 2, first_name: "INVALID", "last_name": "USER", "email": "INVALID@USER.com"}`,
			"",
			app.updateUser,
			http.StatusBadRequest,
		},
		{
			"insertUser valid",
			http.MethodPut,
			`{"first_name": "jack", "last_name": "smith", "email": "jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusNoContent,
		},
		{
			"insertUser invalid",
			http.MethodPut,
			`{"foo": "bar", "first_name": "jack", "last_name": "smith", "email": "jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusBadRequest,
		},
		{
			"insertUser invalid json",
			http.MethodPut,
			`{"first_name: "jack", "last_name": "smith", "email": "jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusBadRequest,
		},
	}

	for _, e := range tests {
		var req *http.Request
		if e.json == "" {
			req = httptest.NewRequest(e.method, "/", nil)
		} else {
			req = httptest.NewRequest(e.method, "/", strings.NewReader(e.json))
		}

		if e.paramID != "" {
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("userID", e.paramID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
		}

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(e.handler)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatus {
			t.Errorf("%s: wrong status returned; expected %d, but got %d", e.name, e.expectedStatus, rr.Code)
		}
	}
}

func TestApplication_refreshUsingCookie(t *testing.T) {
	testUser := data.User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
		Email:     "admin@example.com",
	}

	tokens, _ := app.generateTokenPair(&testUser)

	testCookie := &http.Cookie{
		Name:     "_Host-refresh_token",
		Path:     "/",
		Value:    tokens.RefreshToken,
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   "localhost",
		HttpOnly: true,
		Secure:   true,
	}

	badCookie := &http.Cookie{
		Name:     "_Host-refresh_token",
		Path:     "/",
		Value:    "BAD STRING",
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   "localhost",
		HttpOnly: true,
		Secure:   true,
	}

	tests := []struct {
		name           string
		addCookie      bool
		cookie         *http.Cookie
		expectedStatus int
	}{
		{"valid cookie", true, testCookie, http.StatusOK},
		{"invalid cookie", true, badCookie, http.StatusBadRequest},
		{"no cookie", false, nil, http.StatusUnauthorized},
	}

	for _, e := range tests {
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if e.addCookie {
			req.AddCookie(e.cookie)
		}

		handler := http.HandlerFunc(app.refreshUsingCookie)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatus {
			t.Errorf("%s: wrong status code returned; expected %d, but got %d", e.name, e.expectedStatus, rr.Code)
		}
	}
}

func TestApplication_deleteRefreshCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.deleteRefreshCookies)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("wrong status; expected %d but got %d", http.StatusAccepted, rr.Code)
	}

	foundCookie := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "_Host-refresh_token" {
			foundCookie = true
			if c.Expires.After(time.Now()) {
				t.Errorf("cookie expiration in future, and shoudl not be: %v", c.Expires.UTC())
			}
		}
	}

	if !foundCookie {
		t.Errorf("_Host-refresh_token cookie not found")
	}
}
