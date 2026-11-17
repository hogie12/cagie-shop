package db_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"cagie-shop/auth/internal/db"
	"cagie-shop/auth/internal/migrate"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
func TestUserAndRefreshTokenLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := migrate.Up(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := db.New(pool)

	email := "test-" + time.Now().Format("150405.000000") + "@example.com"
	u, err := q.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: "h", Role: "customer"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = q.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: "h", Role: "customer"})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("duplicate email should violate unique constraint, got %v", err)
	}
	if _, err := q.CreateUser(ctx, db.CreateUserParams{Email: "x" + email, PasswordHash: "h", Role: "root"}); err == nil {
		t.Fatal("role CHECK constraint should reject 'root'")
	}

	hash := "hash-" + u.ID
	exp := pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}
	if err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{UserID: u.ID, TokenHash: hash, ExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}
	if n, err := q.RevokeRefreshToken(ctx, hash); err != nil || n != 1 {
		t.Fatalf("first revoke: n=%d err=%v", n, err)
	}
	if n, _ := q.RevokeRefreshToken(ctx, hash); n != 0 {
		t.Fatalf("second revoke must affect 0 rows, got %d", n)
	}
	got, _ := q.GetRefreshToken(ctx, hash)
	if !got.RevokedAt.Valid {
		t.Fatal("token should be marked revoked")
	}
}
