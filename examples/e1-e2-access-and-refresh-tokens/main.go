// 🌩 Go++ 🌩 #25, #26, #28 · Auth: Access and refresh tokens, rotation, SameSite and CORS
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/semirm-dev/jwt"
)

type Auth struct {
	signer   *jwt.Signer
	verifier *jwt.Verifier
}

func (a *Auth) Login(c *gin.Context) {
	// TODO: check email and password, return 401 if invalid
	userID := "10"

	// A new family per login (device). Every rotation after it keeps the same one.
	a.issue(c, userID, rand.Text())
}

// The SPA calls this on a 401, and after a page reload.
func (a *Auth) Refresh(c *gin.Context) {
	refresh, err := c.Cookie("refresh")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	hash := sha256.Sum256([]byte(refresh))
	_ = hash
	// TODO: in a transaction, lock the row:
	//
	// BEGIN
	// SELECT user_id, family_id, used FROM sessions
	// WHERE refresh_token_hash = $hash AND expires_at > now() AND NOT revoked
	// FOR UPDATE
	//
	// return 401 if no row
	userID, familyID, used := "10", "family-1", false

	// Used token detected, possible theft, revoke the whole family
	if used {
		// TODO: UPDATE sessions SET revoked = true WHERE family_id = $familyID
		// COMMIT
		slog.Warn("refresh token reuse", "user_id", userID, "family_id", familyID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// TODO: UPDATE sessions SET used = true WHERE refresh_token_hash = $hash
	// COMMIT

	a.issue(c, userID, familyID)
}

func (a *Auth) Logout(c *gin.Context) {
	if refresh, err := c.Cookie("refresh"); err == nil {
		hash := sha256.Sum256([]byte(refresh))
		_ = hash
		// TODO:
		// UPDATE sessions SET revoked = true
		// WHERE family_id = (SELECT family_id FROM sessions WHERE refresh_token_hash = $hash)
	}

	setRefreshCookie(c, "", -1)
	c.Status(http.StatusNoContent)
}

// New access token, new refresh token, same family.
func (a *Auth) issue(c *gin.Context, userID, familyID string) {
	// optionally pass data to the access token: map[string]any{"role": "admin"}
	access, err := a.signer.Sign(userID, nil)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	refresh := rand.Text()
	hash := sha256.Sum256([]byte(refresh))
	_ = hash
	// TODO:
	// INSERT INTO sessions (refresh_token_hash, user_id, family_id, expires_at)
	// VALUES ($hash, $userID, $familyID, now() + interval '30 days')

	setRefreshCookie(c, refresh, 30*24*60*60)

	// a response with a token must never be cached, by the browser or a proxy
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"access_token": access})
}

func (a *Auth) RequireAccessToken(c *gin.Context) {
	token, _ := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")

	claims, err := a.verifier.Verify(token)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Set("userID", claims.Subject)
	c.Next()
}

func setRefreshCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh",
		Value:    token,
		Path:     "/auth", // so we can use it in /logout too
		MaxAge:   maxAge,
		HttpOnly: true,                 // JavaScript can't read it
		Secure:   true,                 // only send over HTTPS
		SameSite: http.SameSiteLaxMode, // None if the API is on another site
	})
}

func main() {
	// load keys from your secret manager
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	jwtCfg := jwt.Config{Issuer: "auth", Audience: "api", TTL: 15 * time.Minute}

	signer, err := jwt.NewSigner("2026-11", priv, jwtCfg)
	if err != nil {
		panic(err)
	}

	verifier, err := jwt.NewVerifier(map[string]ed25519.PublicKey{"2026-11": pub}, jwtCfg)
	if err != nil {
		panic(err)
	}

	a := &Auth{signer: signer, verifier: verifier}
	r := gin.New()

	// CORS looks at the origin: app.example.com and api.example.com are two origins.
	r.Use(cors.New(cors.Config{
		// Exact origin: scheme + host + port.
		AllowOrigins: []string{"https://app.example.com"},
		AllowMethods: []string{"GET", "POST"},
		// X-CSRF-Token if you use the CSRF token
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true, // the SPA sends cookies with fetch(url, { credentials: 'include' })
		// How long the browser caches the preflight.
		// Chrome caps it at 2h anyway.
		MaxAge: 2 * time.Hour,
	}))

	r.POST("/auth/login", a.Login)
	r.POST("/auth/refresh", a.Refresh)
	r.POST("/auth/logout", a.Logout)

	api := r.Group("/api", a.RequireAccessToken)
	api.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("userID")})
	})

	// Configure server and listen for requests...
}
