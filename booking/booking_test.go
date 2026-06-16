package booking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testClient returns a client pointed at a test server with no pacing and caching
// off, so unit tests run fast and touch no disk.
func testClient() *Client {
	return NewClient(Config{UserAgent: DefaultUserAgent, NoCache: true})
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := testClient().get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient(Config{NoCache: true, Retries: 5})
	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetBlockedOn403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := testClient().get(context.Background(), srv.URL)
	if err != ErrBlocked {
		t.Errorf("err = %v, want ErrBlocked", err)
	}
}

func TestGetNotFoundOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := testClient().get(context.Background(), srv.URL)
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestGetChallengeBodyIsBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><head><title>Are you a robot?</title></head></html>"))
	}))
	defer srv.Close()

	_, err := testClient().get(context.Background(), srv.URL)
	if err != ErrBlocked {
		t.Errorf("err = %v, want ErrBlocked for a challenge body", err)
	}
}

func TestIsChallenge(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{"<html>a real page about a hotel</html>", false},
		{"<div id=px-captcha></div>", true},
		{"please prove ARE YOU HUMAN", true},
		{"<script src=captcha-delivery.com></script>", true},
		{"normal content", false},
	}
	for _, tc := range cases {
		if got := isChallenge([]byte(tc.body)); got != tc.want {
			t.Errorf("isChallenge(%q) = %v, want %v", tc.body, got, tc.want)
		}
	}
}
