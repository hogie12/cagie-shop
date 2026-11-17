// Package handler holds the HTTP layer of the auth service.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"cagie-shop/auth/internal/db"
	"cagie-shop/auth/internal/password"
	"cagie-shop/auth/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id string) (db.User, error)
	CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) error
	GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) (int64, error)
	RevokeAllUserTokens(ctx context.Context, userID string) error
}

type Auth struct {
	store  Store
	signer *token.Signer
	now    func() time.Time
	// secureCookie adds the Secure flag (HTTPS only). Off for local http.
	secureCookie bool
	// dummyHash is verified when an email is unknown, so a login for a
	// missing user takes as long as one for an existing user.
	dummyHash string
}

func NewAuth(store Store, signer *token.Signer, now func() time.Time, secureCookie bool) (*Auth, error) {
	dummy, err := password.Hash("not-a-real-password")
	if err != nil {
		return nil, err
	}
	return &Auth{store: store, signer: signer, now: now, secureCookie: secureCookie, dummyHash: dummy}, nil
}

func (h *Auth) Routes(r gin.IRouter) {
	r.POST("/register", h.register)
	r.POST("/login", h.login)
	r.POST("/refresh", h.refresh)
	r.POST("/logout", h.logout)
}

type credentials struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// The refresh token lives only in an HttpOnly cookie, so page scripts (and
// any script injected into the page) can never read it. Path /auth is the
// path browsers see through the gateway, so the cookie is sent nowhere else.
const (
	RefreshCookie = "cagie_refresh"
	cookiePath    = "/auth"
)

// accessResponse carries only the short-lived access token. The web app keeps
// it in memory and gets a new one from /refresh when it expires or on reload.
type accessResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int    `json:"expiresIn"`
}

var errBadLogin = "invalid email or password"

func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (h *Auth) register(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := password.Hash(in.Password)
	if err != nil {
		serverError(c, err)
		return
	}
	u, err := h.store.CreateUser(c.Request.Context(), db.CreateUserParams{
		Email: normalizeEmail(in.Email), PasswordHash: hash, Role: token.RoleCustomer,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(c, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "role": u.Role})
}

func (h *Auth) login(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.store.GetUserByEmail(c.Request.Context(), normalizeEmail(in.Email))
	if errors.Is(err, pgx.ErrNoRows) {
		password.Verify(in.Password, h.dummyHash)
		fail(c, http.StatusUnauthorized, errBadLogin)
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	if !password.Verify(in.Password, u.PasswordHash) {
		fail(c, http.StatusUnauthorized, errBadLogin)
		return
	}
	h.issuePair(c, http.StatusOK, u)
}

// refresh swaps a valid refresh token for a new pair. Each token works once;
// presenting an already-used token means it leaked, so every session of that
// user is revoked.
func (h *Auth) refresh(c *gin.Context) {
	plain, err := c.Cookie(RefreshCookie)
	if err != nil || plain == "" || len(plain) > 200 {
		h.rejectRefresh(c)
		return
	}
	ctx := c.Request.Context()
	hash := token.HashRefresh(plain)
	rt, err := h.store.GetRefreshToken(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		h.rejectRefresh(c)
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	if rt.RevokedAt.Valid {
		if err := h.store.RevokeAllUserTokens(ctx, rt.UserID); err != nil {
			serverError(c, err)
			return
		}
		h.rejectRefresh(c)
		return
	}
	if !rt.ExpiresAt.Time.After(h.now()) {
		h.rejectRefresh(c)
		return
	}
	n, err := h.store.RevokeRefreshToken(ctx, hash)
	if err != nil {
		serverError(c, err)
		return
	}
	if n == 0 { // lost a race with a concurrent refresh
		h.rejectRefresh(c)
		return
	}
	u, err := h.store.GetUserByID(ctx, rt.UserID)
	if err != nil {
		serverError(c, err)
		return
	}
	h.issuePair(c, http.StatusOK, u)
}

// logout revokes the session and removes the cookie. It always answers 204,
// so it is safe to call twice or with no session at all.
func (h *Auth) logout(c *gin.Context) {
	if plain, err := c.Cookie(RefreshCookie); err == nil && plain != "" && len(plain) <= 200 {
		if _, err := h.store.RevokeRefreshToken(c.Request.Context(), token.HashRefresh(plain)); err != nil {
			serverError(c, err)
			return
		}
	}
	h.clearCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Auth) issuePair(c *gin.Context, status int, u db.User) {
	now := h.now()
	access, err := h.signer.Issue(u.ID, u.Role, now)
	if err != nil {
		serverError(c, err)
		return
	}
	plain, hash, err := token.NewRefresh()
	if err != nil {
		serverError(c, err)
		return
	}
	err = h.store.CreateRefreshToken(c.Request.Context(), db.CreateRefreshTokenParams{
		UserID: u.ID, TokenHash: hash,
		ExpiresAt: pgtype.Timestamptz{Time: now.Add(token.RefreshTTL), Valid: true},
	})
	if err != nil {
		serverError(c, err)
		return
	}
	h.setCookie(c, plain, int(token.RefreshTTL.Seconds()))
	c.JSON(status, accessResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: int(token.AccessTTL.Seconds())})
}

// SameSite=Lax: the browser sends the cookie only for requests from the same
// site (shop.example.com and api.example.com count as one site), which blocks
// cross-site request forgery against /refresh and /logout.
func (h *Auth) setCookie(c *gin.Context, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookie, value, maxAge, cookiePath, "", h.secureCookie, true)
}

func (h *Auth) clearCookie(c *gin.Context) { h.setCookie(c, "", -1) }

func (h *Auth) rejectRefresh(c *gin.Context) {
	h.clearCookie(c)
	fail(c, http.StatusUnauthorized, "invalid refresh token")
}

func fail(c *gin.Context, status int, msg string) { c.JSON(status, gin.H{"error": msg}) }

// serverError hides internal details from the client but keeps them in logs.
func serverError(c *gin.Context, err error) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, "internal error")
}
