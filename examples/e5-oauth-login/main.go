// 🌩 Go++ 🌩 #29 · Auth: OAuth login
package main

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type Auth struct {
	google  *oauth2.Config
	idToken *oidc.IDTokenVerifier
}

// The SPA navigates here (a link, not a fetch).
func (a *Auth) GoogleLogin(c *gin.Context) {
	state := rand.Text()
	verifier := oauth2.GenerateVerifier() // PKCE

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth",
		Value:    state + "." + verifier,
		Path:     "/auth/google/callback",
		MaxAge:   10 * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Redirect(http.StatusFound, a.google.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)))
}

func (a *Auth) GoogleCallback(c *gin.Context) {
	cookie, _ := c.Cookie("oauth")
	state, verifier, _ := strings.Cut(cookie, ".")
	if state == "" || c.Query("state") != state {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// state and verifier are used once, so can immediately invalidate the cookie
	http.SetCookie(c.Writer, &http.Cookie{Name: "oauth", Path: "/auth/google/callback", MaxAge: -1})

	ctx := c.Request.Context()

	token, err := a.google.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	rawID, _ := token.Extra("id_token").(string)
	id, err := a.idToken.Verify(ctx, rawID) // signature, iss, aud, exp
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	slog.Info("google login", "sub", id.Subject)

	// TODO: find or create the user by id.Subject in database...
	// Avoid email because it can change. sub never does.

	// TODO: from e1-e2 -> a.issue(c, userID, rand.Text()), a new login starts a new family

	c.Redirect(http.StatusFound, "https://app.example.com/")
}

func main() {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		panic(err)
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	a := &Auth{
		google: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  "https://api.example.com/auth/google/callback",
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID},
		},
		idToken: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}
	r := gin.New()

	r.GET("/auth/google/login", a.GoogleLogin)
	r.GET("/auth/google/callback", a.GoogleCallback)

	// Configure server and listen for requests...
}
