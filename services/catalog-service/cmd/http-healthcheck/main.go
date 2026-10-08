package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

const timeout = 2 * time.Second

func main() {
	target := "http://127.0.0.1:8080/ready"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "unexpected health status: %s\n", resp.Status)
		os.Exit(1)
	}
}
