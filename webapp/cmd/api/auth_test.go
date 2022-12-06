package main

import (
	"fmt"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplication_getTokenFromHeaderAndVerify(t *testing.T) {
	testUser := data.User{ID: 1, FirstName: "Admin", LastName: "User", Email: "admin@example.com"}

	tokens, _ := app.generateTokenPair(&testUser)

	tests := []struct {
		name          string
		token         string
		errorExpected bool
		setHeader     bool
		issuer        string
	}{
		{"valid", fmt.Sprintf("Bearer %s", tokens.Token), false, true, app.Domain},
		{"valid but expired", fmt.Sprintf("Bearer %s", expiredToken), true, true, app.Domain},
		{"no header", "", true, false, app.Domain},
		{"invalid token", fmt.Sprintf("Bearer %sINVALID", tokens.Token), true, true, app.Domain},
		{"no bearer", fmt.Sprintf("Bear %s", tokens.Token), true, true, app.Domain},
		{"three header parts", fmt.Sprintf("Bearer %s INVALID", tokens.Token), true, true, app.Domain},
		// make sure the next test is the last one run.
		{"wrong issuer", fmt.Sprintf("Bearer %s", tokens.Token), true, true, "WRONG_ISSUER.com"},
	}

	for _, e := range tests {
		if e.issuer != app.Domain {
			app.Domain = e.issuer
			tokens, _ = app.generateTokenPair(&testUser)
		}

		req := httptest.NewRequest(http.MethodPost, "/auth", nil)
		if e.setHeader {
			req.Header.Set("Authorization", e.token)
		}

		rr := httptest.NewRecorder()

		_, _, err := app.getTokenFromHeaderAndVerify(rr, req)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect error but got one - %s", e.name, err)
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error, but did not get one", e.name)
		}
		app.Domain = "example.com"
	}
}

func TestApplication_generateTokenPair(t *testing.T) {}
