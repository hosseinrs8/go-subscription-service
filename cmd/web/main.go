package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	"log"
	"net/http"
	"os"
	"os/signal"
	"subscription-service/data"
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const webPort = 3000

func main() {
	db := initDB()

	session := initSession()

	wg := sync.WaitGroup{}

	app := Config{
		Session:       session,
		DB:            db,
		InfoLog:       log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		ErrorLog:      log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		Wait:          &wg,
		Models:        data.New(db),
		ErrorChan:     make(chan error),
		ErrorChanDone: make(chan bool),
	}
	app.Mailer = app.createMailer()

	go app.listenForMail()
	go app.listenForErrors()
	go app.listenForShutdown()

	app.serve()
}

func (app *Config) listenForErrors() {
	for {
		select {
		case err := <-app.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.ErrorChanDone:
			return
		}
	}
}

func (app *Config) serve() {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}
	app.InfoLog.Println("Starting web server")

	err := srv.ListenAndServe()
	if err != nil {
		app.ErrorLog.Fatal("ListenAndServe: ", err)
	}
}

func initDB() *sql.DB {
	conn := connectToDB()
	if conn == nil {
		log.Panic("Failed to connect to database")
	}
	return conn
}

func connectToDB() *sql.DB {
	tries := 0
	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)
		if err != nil {
			log.Println("postgres not ready yet: ", err)
			tries++
		} else {
			log.Println("Successfully connected to database")
			return connection
		}
		if tries > 10 {
			return nil
		}
		log.Println("Backing off for 1 second")
		time.Sleep(time.Second)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		log.Printf("Failed to ping database: %v\n", err)
		return nil, err
	}
	return db, nil
}

func initSession() *scs.SessionManager {
	gob.Register(data.User{})

	session := scs.New()
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session
}

func initRedis() *redis.Pool {
	pool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
	return pool
}

func (app *Config) listenForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	app.shutdown()
	close(quit)
	os.Exit(0)
}

func (app *Config) shutdown() {
	app.InfoLog.Println("running cleanup tasks...")

	app.Wait.Wait()
	app.Mailer.DoneChan <- true
	app.ErrorChanDone <- true

	app.Mailer.terminate()
	close(app.ErrorChan)
	close(app.ErrorChanDone)

	app.InfoLog.Println("performing application shutdown...")
}

func (app *Config) createMailer() Mail {
	errChan := make(chan error)
	mailerChan := make(chan Message, 100)
	doneChan := make(chan bool)

	return Mail{
		Domain:      "127.0.0.1",
		Host:        "127.0.0.1",
		Port:        1025,
		Username:    "your-email@your-domain.com",
		Password:    "your-password",
		Encryption:  "none",
		FromAddress: "info@myco.com",
		FromName:    "no-reply",
		Wait:        app.Wait,
		ErrorChan:   errChan,
		MailChan:    mailerChan,
		DoneChan:    doneChan,
	}
}
