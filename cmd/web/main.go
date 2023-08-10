package main

import (
	"context"
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"snippetbox.yp2743.me/internal/models"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	snippets       models.SnippetModelInterface
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

// Singleton pattern to make sure that only one connection pool exists.
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

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Flags will be stored in the variables at runtime.
	addr := flag.String("addr", os.Getenv("PORT"), "HTTP network address")
	dsn := flag.String("dsn", os.Getenv("DB_URL"), "PostgreSQL data source name")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}

	//Close the pool before the main() function exits.
	defer db.pool.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(db.pool)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true

	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		snippets:       &models.SnippetModel{DB: db.pool},
		users:          &models.UserModel{DB: db.pool},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}
