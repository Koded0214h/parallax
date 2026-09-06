package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return srv
}

func TestHealthz(t *testing.T) {
	srv := testServer(t)
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}

func TestEventThenIntent(t *testing.T) {
	srv := testServer(t)

	body := `{"session_id":"s1","user_id":"u1","type":"LOGIN"}`
	res, err := http.Post(srv.URL+"/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /v1/events status = %d, want 202", res.StatusCode)
	}

	res, err = http.Get(srv.URL + "/v1/sessions/s1/intent")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET intent status = %d, want 200", res.StatusCode)
	}
}

func TestEventValidation(t *testing.T) {
	srv := testServer(t)
	res, err := http.Post(srv.URL+"/v1/events", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}

func TestIntentUnknownSession(t *testing.T) {
	srv := testServer(t)
	res, err := http.Get(srv.URL + "/v1/sessions/nope/intent")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}
