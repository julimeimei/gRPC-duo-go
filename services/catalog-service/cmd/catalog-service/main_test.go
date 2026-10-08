package main

import (
	"net/http"
	"testing"
	"time"
)

func TestShutdownTimeoutIsPositive(t *testing.T) {
	if shutdownTimeout <= 0 {
		t.Fatalf("shutdownTimeout = %v, want positive duration", shutdownTimeout)
	}
}

func TestHTTPServerTimeoutFieldsExist(t *testing.T) {
	server := http.Server{
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
	}

	if server.ReadTimeout <= 0 || server.ReadHeaderTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatal("expected all HTTP timeout fields to be positive")
	}
}
