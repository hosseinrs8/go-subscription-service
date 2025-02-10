package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"subscription-service/data"
	"testing"
)

var pageTests = []struct {
	name         string
	url          string
	expectedCode int
	handler      http.HandlerFunc
	sessionData  map[string]any
	expectedHTML string
}{
	{
		name:         "home",
		url:          "/",
		expectedCode: http.StatusOK,
		handler:      testApp.HomePage,
	},
	{
		name:         "login page",
		url:          "/login",
		expectedCode: http.StatusOK,
		handler:      testApp.LoginPage,
		expectedHTML: `<h1 class="mt-5">Login</h1>`,
	},
	{
		name:         "logout",
		url:          "/logout",
		expectedCode: http.StatusSeeOther,
		handler:      testApp.Logout,
		sessionData: map[string]any{
			"userId": 1,
			"user":   data.User{},
		},
	},
}

func Test_Pages(t *testing.T) {
	templatesPath = "./templates"

	for _, e := range pageTests {
		rw := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", e.url, nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		if len(e.sessionData) > 0 {
			for k, v := range e.sessionData {
				testApp.Session.Put(ctx, k, v)
			}
		}

		e.handler.ServeHTTP(rw, req)

		if rw.Code != e.expectedCode {
			t.Errorf("%s failed: expected %d but got %d", e.name, e.expectedCode, rw.Code)
		}

		if len(e.expectedHTML) > 0 {
			html := rw.Body.String()
			if !strings.Contains(html, e.expectedHTML) {
				t.Errorf("%s failed: did not find %s", e.name, e.expectedHTML)
			}
		}
	}
}

func TestConfig_Login(t *testing.T) {
	templatesPath = "./templates"

	postedData := url.Values{
		"email":    {"admin@example.com"},
		"password": {"abc123abc123abc123abc123"},
	}

	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(postedData.Encode()))
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	handler := http.HandlerFunc(testApp.Login)
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusSeeOther {
		t.Errorf("expected %d status code but got %d", http.StatusSeeOther, rw.Code)
	}

	if !testApp.Session.Exists(ctx, "userId") {
		t.Error("did not find userId in session")
	}
}

func TestConfig_SubscribeToPlan(t *testing.T) {
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/subscribe?id=1", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	testApp.Session.Put(ctx, "userId", 1)
	testApp.Session.Put(ctx, "user", data.User{
		ID:        1,
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "Admin",
		Active:    1,
	})

	handler := http.HandlerFunc(testApp.SubscribeToPlan)
	handler.ServeHTTP(rw, req)

	testApp.Wait.Wait()

	if rw.Code != http.StatusSeeOther {
		t.Errorf("expected status code %d but got %d", http.StatusSeeOther, rw.Code)
	}
}
