package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfig_AddDefaultData(t *testing.T) {
	req, ctx := buildCtx()

	testApp.Session.Put(ctx, "flash", "flash")
	testApp.Session.Put(ctx, "warning", "warning")
	testApp.Session.Put(ctx, "error", "error")

	td := testApp.AddDefaultData(&TemplateData{}, req)
	if td.Flash != "flash" {
		t.Errorf("failed to get flash data")
	}
	if td.Warning != "warning" {
		t.Errorf("failed to get warning data")
	}
	if td.Error != "error" {
		t.Errorf("failed to get error data")
	}
}

func TestConfig_IsAuthenticated(t *testing.T) {
	req, ctx := buildCtx()

	auth := testApp.IsAuthenticated(req)
	if auth {
		t.Error("got true without authentication")
	}

	testApp.Session.Put(ctx, "userId", 1)

	auth = testApp.IsAuthenticated(req)
	if !auth {
		t.Error("got false after authentication")
	}
}

func TestConfig_render(t *testing.T) {
	templatesPath = "./templates"

	rw := httptest.NewRecorder()
	req, _ := buildCtx()

	testApp.render(rw, req, "home.page.gohtml", &TemplateData{})

	if rw.Code != 200 {
		t.Error("failed to render the homepage")
	}
}

func buildCtx() (*http.Request, context.Context) {
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)
	return req, ctx
}
