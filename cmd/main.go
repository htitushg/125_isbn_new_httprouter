package main

import (
	"125_isbn_new/donnees"
	"125_isbn_new/internal/assert"
	"125_isbn_new/internal/models"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
)

// Add a db struct field to hold the configuration settings for our database connection
// pool. For now this only holds the DSN, which we will read in from a command-line flag.
// Add maxOpenConns, maxIdleConns and maxIdleTime fields to hold the configuration
// settings for the connection pool.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
}
type application struct {
	logger         *slog.Logger
	snippets       *models.SnippetModel
	user           *models.UsersModel
	livres         *models.LivresModel
	editeurs       *models.EditeursModel
	auteurs        *models.AuteursModel
	users          *models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	username       string
}

var logs *os.File

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Ajouté le 29/05/2024 10h54
	//var jsonHandler *slog.JSONHandler
	//var err error
	//var filename string
	//filename = assert.Path + "logs/logs_" + time.Now().Format(time.DateOnly) + ".log"
	////models.closeLog()
	//logs, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	log.Println(models.GetCurrentFuncName(), slog.Any("output", err))
	//}
	//jsonHandler = slog.NewJSONHandler(logs, nil)
	//logger := slog.New(jsonHandler)
	// Fin Ajouté le 29/05/2024 10h54
	addr := flag.String("addr", ":8090", "HTTP network address")
	dsn := flag.String("dsn", donnees.NSD, "MySQL data source name")
	flag.Parse()

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	formDecoder := form.NewDecoder()
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	// Make sure that the Secure attribute is set on our session cookies.
	// Setting this means that the cookie will only be sent by a user's web
	// browser when a HTTPS connection is being used (and won't be sent over an
	// unsecure HTTP connection).
	sessionManager.Cookie.Secure = true

	// Initialize a models.SnippetModel et models.UserModel instance
	// containing the connection pool and add it
	// to the application dependencies.
	app := &application{
		logger:         logger,
		snippets:       &models.SnippetModel{DB: db},
		user:           &models.UsersModel{DB: db},
		livres:         &models.LivresModel{DB: db},
		editeurs:       &models.EditeursModel{DB: db},
		auteurs:        &models.AuteursModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}
	// Supprimer s'il elles existent les sessions en cours

	// Initialize a tls.Config struct to hold the non-default TLS settings we
	// want the server to use. In this case the only thing that we're changing
	// is the curve preferences value, so that only elliptic curves with
	// assembly implementations are used.
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}
	srv := &http.Server{
		Addr:      *addr,
		Handler:   app.routes(),
		ErrorLog:  slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig: tlsConfig,
		// Add Idle, Read and Write timeouts to the server.
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}
	logger.Info("starting server", "addr", srv.Addr)
	// Use the ListenAndServeTLS() method to start the HTTPS server. We
	// pass in the paths to the TLS certificate and corresponding private key as
	// the two parameters.
	fmt.Printf("Path= %v\n", assert.Path)
	err = srv.ListenAndServeTLS(assert.Path+"tls/cert.pem", assert.Path+"tls/key.pem")

	logger.Error(err.Error())
	os.Exit(1)

}

// The openDB() function wraps sql.Open() and returns a sql.DB connection pool
// for a given DSN.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
