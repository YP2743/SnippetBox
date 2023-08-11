package models

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type postgres struct {
	pool *pgxpool.Pool
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

func openDB(dsn string) (*postgres, error) {
	var err error
	pgOnce.Do(func() {
		var db *pgxpool.Pool
		db, err = pgxpool.New(context.Background(), dsn)
		if err == nil {
			err = db.Ping(context.Background())
			if err == nil {
				pgInstance = &postgres{pool: db}
			}
		}
	})

	return pgInstance, err
}

func newTestDB(t *testing.T) *pgxpool.Pool {

	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	db, err := openDB(os.Getenv("TEST_DB_URL"))
	if err != nil {
		t.Fatal(err)
	}

	script, err := os.ReadFile("./testdata/setup.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.pool.Exec(context.Background(), string(script))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		script, err := os.ReadFile("./testdata/teardown.sql")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.pool.Exec(context.Background(), string(script))
		if err != nil {
			t.Fatal(err)
		}
	})

	return db.pool
}
