package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

type Credentials struct {
	Username string `json:"email"`
	Password string `json:"password"`
}

func (app *application) authenticate(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	// read a json payload
	err := app.readJSON(w, r, &creds)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// look up the user by email address
	user, err := app.DB.GetUserByEmail(creds.Username)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// generate tokens
	tokenPairs, err := app.generateTokenPair(user)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "_Host-refresh_token",
		Path:     "/",
		Value:    tokenPairs.RefreshToken,
		Domain:   "localhost",
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	//	send token to user
	_ = app.writeJSON(w, http.StatusOK, tokenPairs)

}

func (app *application) refresh(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshToken := r.Form.Get("refresh_token")
	claims := &Claims{}

	_, err = jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (any, error) {
		return []byte(app.JWTSecret), nil
	})
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
	}

	if time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now()) > 30*time.Second {
		app.errorJSON(w, errors.New("refresh token does not need to be renewed yet"), http.StatusTooEarly)
	}

	// get the user id from the claims
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	user, err := app.DB.GetUser(userID)
	if err != nil {
		app.errorJSON(w, errors.New("unknown user"), http.StatusBadRequest)
		return
	}

	tokenPairs, err := app.generateTokenPair(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "_Host-refresh_token",
		Path:     "/",
		Value:    tokenPairs.RefreshToken,
		Domain:   "localhost",
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	_ = app.writeJSON(w, http.StatusOK, tokenPairs)

}

func (app *application) refreshUsingCookie(w http.ResponseWriter, r *http.Request) {
	for _, cookie := range r.Cookies() {
		if cookie.Name == "_Host-refresh_token" {
			claims := &Claims{}
			refreshToken := cookie.Value

			_, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (any, error) {
				return []byte(app.JWTSecret), nil
			})
			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			//if time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now()) > 30*time.Second {
			//	app.errorJSON(w, errors.New("refresh token does not need to be renewed yet"), http.StatusTooEarly)
			//}

			// get the user id from the claims
			userID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			user, err := app.DB.GetUser(userID)
			if err != nil {
				app.errorJSON(w, errors.New("unknown user"), http.StatusBadRequest)
				return
			}

			tokenPairs, err := app.generateTokenPair(user)
			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "_Host-refresh_token",
				Path:     "/",
				Value:    tokenPairs.RefreshToken,
				Domain:   "localhost",
				Expires:  time.Now().Add(refreshTokenExpiry),
				MaxAge:   int(refreshTokenExpiry.Seconds()),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})

			_ = app.writeJSON(w, http.StatusOK, tokenPairs)
			return
		}
	}
	app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)

}

func (app *application) allUsers(w http.ResponseWriter, r *http.Request) {
	users, err := app.DB.AllUsers()
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, users)
}

func (app *application) getUser(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	user, err := app.DB.GetUser(userId)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, user)
}

func (app *application) updateUser(w http.ResponseWriter, r *http.Request) {
	var user data.User

	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = app.DB.UpdateUser(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) deleteUser(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = app.DB.DeleteUser(userId)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) insertUser(w http.ResponseWriter, r *http.Request) {
	var user data.User

	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	_, err = app.DB.InsertUser(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) deleteRefreshCookies(w http.ResponseWriter, r *http.Request) {
	delCookie := http.Cookie{
		Name:     "_Host-refresh_token",
		Path:     "/",
		Domain:   "localhost",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &delCookie)

	w.WriteHeader(http.StatusAccepted)
}
