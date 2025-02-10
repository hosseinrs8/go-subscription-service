package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(app.SessionLoad)

	mux.Get("/", app.HomePage)

	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.Login)
	mux.Get("/logout", app.Logout)
	mux.Get("/register", app.RegisterPage)
	mux.Post("/register", app.Register)
	mux.Get("/activate-acc", app.ActivateAccount)
	//mux.Get("/email", func(writer http.ResponseWriter, request *http.Request) {
	//	m := Mail{
	//		Domain:      "127.0.0.1",
	//		Host:        "127.0.0.1",
	//		Port:        1025,
	//		Username:    "your-email@your-domain.com",
	//		Password:    "your-password",
	//		Encryption:  "none",
	//		FromAddress: "info@myco.com",
	//		FromName:    "no-reply",
	//		ErrorChan:   make(chan error),
	//	}
	//	msg := Message{
	//		To:      []string{"me@here.com"},
	//		Subject: "Welcome to MyCo",
	//		Data:    "Your account has been activated.",
	//	}
	//	m.send(msg)
	//})

	mux.Mount("/members", app.authRoutes())
	return mux
}

func (app *Config) authRoutes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(app.Auth)

	mux.Get("/plans", app.ChooseSubscription)
	mux.Get("/subscribe", app.SubscribeToPlan)

	return mux
}
