// 🌩 Go++ 🌩 #27 · Auth: CSRF
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// The SPA calls this once on load, keeps the token from the body and
// sends it back in the X-CSRF-Token header.
func IssueCSRF(c *gin.Context) {
	token := rand.Text()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf",
		Value:    token,
		Path:     "/auth",
		HttpOnly: true, // the SPA reads the body, not the cookie
		Secure:   true,
		SameSite: http.SameSiteLaxMode, // None if the API is on another site
	})
	c.JSON(http.StatusOK, gin.H{"csrf_token": token})
}

// Only for routes authenticated by a cookie. /api uses the Authorization header,
// which the browser never attaches on its own.
func RequireCSRF(c *gin.Context) {
	// load from config. Not: allow a missing Origin, browsers always send it on POST.
	if c.GetHeader("Origin") != "https://app.example.com" {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Another site can't read the token, so it can't put it in the header.
	// Not: cookie == header, it leaks timing.
	cookie, _ := c.Cookie("csrf")
	header := c.GetHeader("X-CSRF-Token")
	if header == "" || subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.Next()
}

func main() {
	r := gin.New()

	r.GET("/auth/csrf", IssueCSRF)

	// Not: GET /auth/logout, a link or an <img> could log the user out.
	auth := r.Group("/auth", RequireCSRF)
	auth.POST("/refresh", func(c *gin.Context) { /* see #26 */ })
	auth.POST("/logout", func(c *gin.Context) { /* see #26 */ })

	// Configure server and listen for requests...
}
