package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"testing"
)

var routes = []string{
	"/",
	"/login",
	"/logout",
	"/register",
	"/activate-acc",
	"/members/plans",
	"/members/subscribe",
}

func Test_RoutesExists(t *testing.T) {
	testRoutes := testApp.routes()
	chiRoutes := testRoutes.(chi.Router)

	for _, route := range routes {
		routeExists(t, chiRoutes, route)
	}
}

func routeExists(t *testing.T, chiRoutes chi.Router, route string) {
	found := false

	chi.Walk(chiRoutes, func(method string, chiRoute string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if route == chiRoute {
			found = true
			return nil
		}
		return nil
	})

	if !found {
		t.Errorf("Did not find %s in registered routes", route)
	}
}
