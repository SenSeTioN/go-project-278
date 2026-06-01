package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupRouter(nil, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want %d, got %d", http.StatusOK, w.Code)
	}
	if body := w.Body.String(); body != "pong" {
		t.Fatalf("body: want %q, got %q", "pong", body)
	}
}

func TestNoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupRouter(nil, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: want %d, got %d", http.StatusNotFound, w.Code)
	}
}
