package main

import (
	"context"
	"encoding/gob"
	"github.com/alexedwards/scs/v2"
	"log"
	"net/http"
	"os"
	"subscription-service/data"
	"sync"
	"testing"
	"time"
)

var testApp Config

func TestMain(m *testing.M) {
	gob.Register(data.User{})

	tmpPath = "./../../tmp"
	manualPath = "./../../pdf"

	session := scs.New()
	//session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	testApp = Config{
		Session:       session,
		DB:            nil,
		InfoLog:       log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		ErrorLog:      log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		Wait:          &sync.WaitGroup{},
		Models:        data.TestNew(nil),
		ErrorChan:     make(chan error),
		ErrorChanDone: make(chan bool),
	}

	testApp.Mailer = Mail{
		Wait:      testApp.Wait,
		ErrorChan: make(chan error),
		MailChan:  make(chan Message),
		DoneChan:  make(chan bool),
	}

	go func() {
		for {
			select {
			case <-testApp.Mailer.MailChan:
				testApp.Wait.Done()
			case <-testApp.Mailer.ErrorChan:
			case <-testApp.Mailer.DoneChan:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case err := <-testApp.ErrorChan:
				testApp.ErrorLog.Println(err)
			case <-testApp.ErrorChanDone:
				return
			}
		}
	}()

	os.Exit(m.Run())
}

func getCtx(req *http.Request) context.Context {
	ctx, err := testApp.Session.Load(req.Context(), req.Header.Get("X-Session"))
	if err != nil {
		log.Println(err)
	}
	return ctx
}
