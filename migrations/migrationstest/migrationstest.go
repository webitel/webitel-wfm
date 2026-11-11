// Package migrationstest gives a test its own PostgreSQL database with the platform
// stand-ins the migrations need, dropped when the test ends.
package migrationstest

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"

	_ "embed"
)

// DSNEnv names the server to test on, a URL such as
// postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable. Unset, the test is skipped.
const DSNEnv = "WFM_TEST_POSTGRES_DSN"

//go:embed platform.sql
var platform string

// New creates the database and returns its URL, a connection to it and a provider of
// the migrations set up as cmd/migrate sets it up. No migration is applied.
func New(tb testing.TB, migrations fs.FS) (string, *sql.DB, *goose.Provider) {
	tb.Helper()

	dsn := os.Getenv(DSNEnv)
	if dsn == "" {
		tb.Skipf("%s is not set", DSNEnv)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		tb.Fatalf("%s must be a URL: %v", DSNEnv, err)
	}

	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		tb.Fatalf("connect: %v", err)
	}

	tb.Cleanup(func() { _ = admin.Close(ctx) })

	// Random, not time-based: parallel go test runs share the server.
	name := fmt.Sprintf("wfm_test_%x", rand.Uint64())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		tb.Fatalf("create database: %v", err)
	}

	tb.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			tb.Errorf("drop database: %v", err)
		}
	})

	u.Path = "/" + name
	dsn = u.String()

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		tb.Fatal(err)
	}

	db := stdlib.OpenDB(*cfg)

	tb.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, platform); err != nil {
		tb.Fatalf("platform: %v", err)
	}

	store, err := database.NewStore(database.DialectPostgres, "wfm_schema_version")
	if err != nil {
		tb.Fatal(err)
	}

	p, err := goose.NewProvider("", db, migrations, goose.WithStore(store))
	if err != nil {
		tb.Fatal(err)
	}

	return dsn, db, p
}

// Up is New with every migration applied.
func Up(tb testing.TB, migrations fs.FS) string {
	tb.Helper()

	dsn, _, p := New(tb, migrations)
	if _, err := p.Up(context.Background()); err != nil {
		tb.Fatalf("migrate: %v", err)
	}

	return dsn
}
