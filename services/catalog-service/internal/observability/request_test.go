package observability

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeRequestID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "valid", in: "request-123_abc.DEF", want: "request-123_abc.DEF"},
		{name: "trim", in: " request-123 ", want: "request-123"},
		{name: "empty", in: "", want: ""},
		{name: "too long", in: strings.Repeat("a", 129), want: ""},
		{name: "invalid char", in: "../../etc/passwd", want: ""},
		{name: "space", in: "bad id", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRequestID(tt.in); got != tt.want {
				t.Fatalf("NormalizeRequestID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContextWithRequestID(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "request-123")

	if got := RequestIDFromContext(ctx); got != "request-123" {
		t.Fatalf("RequestIDFromContext() = %q, want %q", got, "request-123")
	}
}

func TestContextWithRequestIDGeneratesWhenInvalid(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "bad id")

	if got := RequestIDFromContext(ctx); got == "" || got == "bad id" {
		t.Fatalf("RequestIDFromContext() = %q, want generated id", got)
	}
}
