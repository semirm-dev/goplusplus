// 🌩 Go++ 🌩 #28 · Auth: SameSite and CORS
package main

import (
	"crypto/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	// check email and password, save the refresh token hash (see #25)...
	refresh := rand.Text()

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh",
		Value:    refresh,
		Path:     "/auth",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   true,
		// SameSite looks at the site: sent from app.example.com, not from evil.com.
		// Not: http.SameSiteNoneMode to fix a CORS error.
		SameSite: http.SameSiteLaxMode,
	})

	// return the access token...
}

func main() {
	r := gin.New()

	// CORS looks at the origin: app.example.com and api.example.com are two origins.
	r.Use(cors.New(cors.Config{
		// Exact origin: scheme + host + port.
		// Not: AllowAllOrigins, or an AllowOriginFunc that returns true for any origin.
		AllowOrigins:     []string{"https://app.example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true, // the SPA sends cookies with fetch(url, { credentials: 'include' })
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/auth/login", Login)

	// Configure server and listen for requests...
}
